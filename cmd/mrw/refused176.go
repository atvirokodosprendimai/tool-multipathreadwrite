package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// keptPlanCap is the most a kept plan holds: plan.Parse's own line limit, the
// number the parser already refuses past (ADR-176).
const keptPlanCap = 16 * 1024 * 1024

// keptPlansMax is how many kept plans stay in the directory.
const keptPlansMax = 16

// planTee copies what the plan parser reads from standard input, after the
// CRLF trim, so a refused plan can be kept as the parser saw it. It never reads
// ahead of the parser; past keptPlanCap it drops its copy.
type planTee struct {
	src  io.Reader
	buf  bytes.Buffer
	over bool
}

func (t *planTee) Read(p []byte) (int, error) {
	n, err := t.src.Read(p)
	if !t.over && n > 0 {
		if t.buf.Len()+n > keptPlanCap {
			t.over = true
			t.buf = bytes.Buffer{}
		} else {
			t.buf.Write(p[:n])
		}
	}
	return n, err
}

// keep writes what the tee copied to a new file in <state home>/mrw-refused,
// beside mrw's state directory and not in it (ADR-077 stays whole), and
// returns its path with the stderr line. A plan the parser never read keeps
// nothing and says nothing; one that cannot be kept says why.
func (t *planTee) keep() (path, line string) {
	if t.over {
		return "", "mrw: the plan from standard input was not kept: it is over 16 MiB"
	}
	if t.buf.Len() == 0 {
		return "", ""
	}
	path, err := t.write()
	if err != nil {
		return "", fmt.Sprintf("mrw: the plan from standard input was not kept: %v", err)
	}
	sum := sha256.Sum256(t.buf.Bytes())
	return path, fmt.Sprintf("mrw: the plan from standard input is kept at %s (%d bytes, sha %s): fix it there and run the same command with %s in place of -",
		path, t.buf.Len(), hex.EncodeToString(sum[:4]), path)
}

func (t *planTee) write() (string, error) {
	base, err := state.Base()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(filepath.Dir(base), "mrw-refused")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	var salt [4]byte
	if _, err := rand.Read(salt[:]); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s-%d-%s.plan", time.Now().UTC().Format("20060102T150405Z"), os.Getpid(), hex.EncodeToString(salt[:]))
	path := filepath.Join(dir, name)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", err
	}
	if _, err := f.Write(t.buf.Bytes()); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	trimKept(dir)
	return path, nil
}

// trimKept removes all but the newest keptPlansMax plans by name, whose prefix
// is the time. Best effort: two runs trimming at once may leave one more.
func trimKept(dir string) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var names []string
	for _, e := range ents {
		if strings.HasSuffix(e.Name(), ".plan") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for len(names) > keptPlansMax {
		_ = os.Remove(filepath.Join(dir, names[0]))
		names = names[1:]
	}
}
