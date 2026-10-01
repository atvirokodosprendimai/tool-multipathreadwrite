package state

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Hold opens the lock file name in root's state directory and takes it
// exclusively, waiting while another holder has it, and returns the function
// that releases it; calling that again does nothing. The kernel releases the
// lock if the process dies holding it.
//
// It is here, beside the files it guards, because more than the ledger needs
// it (ADR-079): the working set and the tally were rewritten by truncate-and-
// write with no lock, so a process racing another read an emptied file,
// rebuilt from nothing and saved — wiping the whole tally or working set, not
// one count. A lock file is locked once per process at a time: one process
// opening it a second time waits on itself, so a lock is never taken inside
// another hold of the same name.
func Hold(root, name string) (func(), error) {
	path, err := Path(root, name)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	if err := lock(f); err != nil {
		_ = f.Close()
		return nil, err
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			_ = unlock(f)
			_ = f.Close()
		})
	}, nil
}

// HoldWithin is Hold with a bounded wait (ADR-110). It tries the lock without
// blocking and retries with a growing pause until wait has passed, then returns
// a LockTimeoutError naming the holder; a wait of 0 tries once. Hold waited for
// ever, so a live holder that had stopped blocked every writer after it with
// nothing said. A goroutine around the blocking call would not do: abandoned at
// the deadline, it takes the lock later and holds it with nobody to release it.
//
// Whoever takes a lock here writes its pid to <name>.holder beside it, for the
// next waiter to name — a separate file because Windows locks a byte range, and
// a waiter could not read the locked file itself.
func HoldWithin(root, name string, wait time.Duration) (func(), error) {
	path, err := Path(root, name)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(wait)
	pause := 10 * time.Millisecond
	for {
		ok, err := tryLock(f)
		if err != nil {
			_ = f.Close()
			return nil, err
		}
		if ok {
			break
		}
		if !time.Now().Before(deadline) {
			_ = f.Close()
			return nil, &LockTimeoutError{Name: name, Wait: wait, Holder: holderOf(path)}
		}
		time.Sleep(min(pause, time.Until(deadline)))
		pause = min(2*pause, 250*time.Millisecond)
	}
	// The name is a courtesy to the next waiter; failing to write it changes
	// nothing about the lock, which the kernel holds.
	_ = Write(path+".holder", []byte(strconv.Itoa(os.Getpid())+"\n"), 0o600)
	var once sync.Once
	return func() {
		once.Do(func() {
			_ = unlock(f)
			_ = f.Close()
		})
	}, nil
}

// holderOf reads the pid the lock's last taker wrote beside it, or "".
func holderOf(path string) string {
	b, err := os.ReadFile(path + ".holder")
	if err != nil {
		return ""
	}
	pid := strings.TrimSpace(string(b))
	if _, err := strconv.Atoi(pid); err != nil {
		return ""
	}
	return pid
}

// LockTimeoutError is HoldWithin's refusal: the lock Name was still held after
// Wait, by the process Holder names (empty when no taker wrote one).
type LockTimeoutError struct {
	Name   string
	Wait   time.Duration
	Holder string
}

func (e *LockTimeoutError) Error() string {
	who := "another process"
	if e.Holder != "" {
		who = "process " + e.Holder
	}
	return fmt.Sprintf("%s still holds %s after %s", who, e.Name, e.Wait)
}
