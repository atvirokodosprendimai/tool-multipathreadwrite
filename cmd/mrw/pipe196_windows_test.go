package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

var (
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	procCreateNamedPipe = kernel32.NewProc("CreateNamedPipeW")
	procConnectPipe     = kernel32.NewProc("ConnectNamedPipe")
	procDisconnectPipe  = kernel32.NewProc("DisconnectNamedPipe")
)

// servePipeOnce serves text to the first client of a one-instance pipe and
// closes it, as a one-shot server does.
func servePipeOnce(t *testing.T, name, text string) <-chan error {
	t.Helper()
	p, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		t.Fatal(err)
	}
	const outbound, byteWait = 0x2, 0x0
	h, _, callErr := procCreateNamedPipe.Call(uintptr(unsafe.Pointer(p)), outbound, byteWait, 1, 4096, 4096, 0, 0)
	if syscall.Handle(h) == syscall.InvalidHandle {
		t.Fatalf("CreateNamedPipeW: %v", callErr)
	}
	done := make(chan error, 1)
	go func() {
		defer func() { _ = syscall.CloseHandle(syscall.Handle(h)) }()
		if r, _, err := procConnectPipe.Call(h, 0); r == 0 && err != syscall.Errno(535) { // ERROR_PIPE_CONNECTED
			done <- fmt.Errorf("ConnectNamedPipe: %w", err)
			return
		}
		var n uint32
		err := syscall.WriteFile(syscall.Handle(h), []byte(text), &n, nil)
		_ = syscall.FlushFileBuffers(syscall.Handle(h))
		_, _, _ = procDisconnectPipe.Call(h)
		done <- err
	}()
	return done
}

// ADR-196 T5, from the Windows re-probe of v1.86.1: a plan served by a one-shot
// pipe failed "All pipe instances are busy", since the state check opened the
// path first and spent the pipe's one instance. A pipe plan and a pipe list are
// each opened once.
func TestAPlanOrListServedOverAPipeIsOpenedOnce(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, map[string]string{"a.txt": "listed\n"})
	t.Chdir(root)
	name := fmt.Sprintf(`\\.\pipe\mrw-t5-%d`, time.Now().UnixNano())
	done := servePipeOnce(t, name, "@@ c.txt 0 create\nmade\n")
	if out, code := runIn(t, root, "write", name); code != 0 {
		t.Fatalf("write %s: exit %d:\n%s", name, code, out)
	}
	if b, err := os.ReadFile(filepath.Join(root, "c.txt")); err != nil || string(b) != "made\n" {
		t.Fatalf("c.txt = %q, %v", b, err)
	}
	if err := <-done; err != nil {
		t.Fatalf("the plan pipe: %v", err)
	}
	name += "-list"
	done = servePipeOnce(t, name, "a.txt\n")
	if out, code := runIn(t, root, "read", "--files-from", name); code != 0 {
		t.Fatalf("read --files-from %s: exit %d:\n%s", name, code, out)
	}
	if err := <-done; err != nil {
		t.Fatalf("the list pipe: %v", err)
	}
}
