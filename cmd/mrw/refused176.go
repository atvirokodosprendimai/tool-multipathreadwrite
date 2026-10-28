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

// keptPlanGrace is how long after its last write a kept plan is out of the
// trim's reach: another run may be saving it (ADR-176).
const keptPlanGrace = time.Minute

// keptPlanLockWait is how long a run waits for the lock that covers creating a
// kept plan and trimming around it (ADR-176).
const keptPlanLockWait = 10 * time.Second

// planTee copies what the plan parser reads from standard input, after the
// CRLF trim, so a refused plan can be kept as the parser saw it. It copies what
// the reader above it asked for, which can run ahead of the last line parsed
// when the parser stops before EOF: the scanner's buffer and the byte-order-mark
// count's bufio.Reader (4096 bytes) both read ahead. A refused plan is kept
// whole, and past keptPlanCap it drops its copy.
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
	// One lock from before the create through the trim: no other run trims
	// between this plan's creation and the trim that spares it (ADR-176). It is
	// a file in the kept directory, not in mrw's state directory, so a run
	// whose home reaches the same directory through a link shares it.
	release, err := state.HoldFileWithin(filepath.Join(dir, ".lock"), keptPlanLockWait)
	if err != nil {
		return "", fmt.Errorf("%s: %w", dir, err)
	}
	defer release()
	var salt [4]byte
	if _, err := rand.Read(salt[:]); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s-%d-%s.plan", time.Now().UTC().Format("20060102T150405.000000000Z"), os.Getpid(), hex.EncodeToString(salt[:]))
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
	trimKept(dir, name)
	return path, nil
}

// trimKept removes plans until the newest keptPlansMax stay, by name, whose
// prefix is the time. It never removes the one this run just kept: a name can
// sort before others written in the same instant, and the path it returns must
// exist. Creation and trim run under one lock, so no other run trims between a
// plan's creation and the trim that spares it. It also spares a plan modified
// within keptPlanGrace, a backstop for a lock file removed while held (the
// directory deleted under a run): then two runs trimming at once could each
// remove the other's plan, which is still being saved. So the
// directory may briefly hold more than keptPlansMax, as long as a plan's
// modification time is in the future (a clock rolled back) or recent.
func trimKept(dir, keep string) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	type plan struct {
		name   string
		recent bool
	}
	var plans []plan
	for _, e := range ents {
		if !strings.HasSuffix(e.Name(), ".plan") || e.Name() == keep {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		plans = append(plans, plan{e.Name(), time.Since(fi.ModTime()) < keptPlanGrace})
	}
	sort.Slice(plans, func(i, j int) bool { return plans[i].name < plans[j].name })
	excess := len(plans) - (keptPlansMax - 1)
	for _, p := range plans {
		if excess <= 0 {
			return
		}
		if p.recent {
			continue
		}
		_ = os.Remove(filepath.Join(dir, p.name))
		excess--
	}
}
