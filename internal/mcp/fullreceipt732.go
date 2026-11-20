package mcp

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/state"
)

// keptReceiptsMax is how many kept receipts stay, newest by name, plus any younger
// than trimGrace until the next save trims them.
const keptReceiptsMax = 16

// keptReceiptMaxBytes caps one kept receipt, the truncation note included.
const keptReceiptMaxBytes = 1 << 20

// keptTruncationNote ends a receipt the cap cut. It is written inside the cap.
const keptTruncationNote = "\n[truncated: the receipt is larger than 1048576 bytes; this is its beginning]\n"

// trimGrace is how young a receipt must be to be spared by a trim whatever the
// count: a writer that has just published one is about to name it in an answer,
// and 16 others publishing meanwhile must not take it from under that answer.
// A variable so a test can ask for the bare count.
var trimGrace = 10 * time.Second

// beforePublish runs after a receipt is written and before it gets its name: the
// instant other writers can publish and trim. A seam for the test of that race.
var beforePublish = func() {}

// receiptClock is the clock a receipt's name is stamped from. A seam, so a test
// can freeze it: a coarse clock (Windows') gives receipts kept in one burst the
// same instant.
var receiptClock = time.Now

var (
	stampMu   sync.Mutex
	lastStamp time.Time
)

// nextStamp is the instant a receipt is named for: the clock's, but always later
// than the last one this process gave, by a nanosecond when the clock stood still
// or stepped back. The names sort as the receipts were kept, which the trim's
// "newest" is, so no tie is left to the random salt.
func nextStamp() time.Time {
	stampMu.Lock()
	defer stampMu.Unlock()
	now := receiptClock()
	if !now.After(lastStamp) {
		now = lastStamp.Add(time.Nanosecond)
	}
	lastStamp = now
	return now
}

// capWriter passes on at most left bytes, drops the rest and remembers that it did.
type capWriter struct {
	w    io.Writer
	left int
	cut  bool
}

func (c *capWriter) Write(p []byte) (int, error) {
	n := len(p)
	if n > c.left {
		n, c.cut = max(c.left, 0), true
	}
	if n > 0 {
		if _, err := c.w.Write(p[:n]); err != nil {
			return 0, err
		}
		c.left -= n
	}
	return len(p), nil
}

// countWriter counts what is written to it.
type countWriter int

func (c *countWriter) Write(p []byte) (int, error) { *c += countWriter(len(p)); return len(p), nil }

// keptReceiptBytes is the size writeReceiptFile would write for these.
func keptReceiptBytes(report string, structured any) int {
	var n countWriter
	_ = writeReceiptFile(&n, report, structured)
	return int(n)
}

// keepReceipt is keepFullReceipt, as a variable so a test can count and fail it.
var keepReceipt = keepFullReceipt

// keepFullReceipt writes the whole receipt (#732): its text report and its
// structured value, to a new file in <state home>/mrw-receipts, beside mrw's
// state directory and not in it, as ADR-176 keeps a refused plan: mode 0600 (on
// Windows the file takes the directory's inherited ACL, as mrw's other state
// files do), the newest keptReceiptsMax, one file at most keptReceiptMaxBytes.
// The caller hands it the receipt with its file content already taken off. It
// returns the path, or an error when none could be kept; the caller then names
// none.
//
// The file is written under a temporary name and given its own, stamped now, only
// when it is whole: a trim by another writer never sees it half-written, and the
// receipt it publishes is the newest, so the 16 that trim spares are not older
// than it. The lock guards only the trim; one that cannot take it at once is
// skipped until the next keep.
func keepFullReceipt(report string, structured any) (string, error) {
	base, err := state.Base()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(filepath.Dir(base), "mrw-receipts")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	var salt [4]byte
	if _, err := rand.Read(salt[:]); err != nil {
		return "", err
	}
	tmp := filepath.Join(dir, fmt.Sprintf(".tmp-%d-%s", os.Getpid(), hex.EncodeToString(salt[:])))
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", err
	}
	cw := &capWriter{w: f, left: keptReceiptMaxBytes - len(keptTruncationNote)}
	werr := writeReceiptFile(cw, report, structured)
	if werr == nil && cw.cut {
		_, werr = io.WriteString(f, keptTruncationNote)
	}
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		_ = os.Remove(tmp)
		return "", werr
	}
	beforePublish()
	name := fmt.Sprintf("%s-%d-%s.receipt", nextStamp().UTC().Format("20060102T150405.000000000Z"), os.Getpid(), hex.EncodeToString(salt[:]))
	path := filepath.Join(dir, name)
	// A hard link refuses a name that is taken, as O_EXCL did; where none can be
	// made, a rename, whose name carries a random salt.
	if err := os.Link(tmp, path); err != nil {
		if err := os.Rename(tmp, path); err != nil {
			_ = os.Remove(tmp)
			return "", err
		}
	} else {
		_ = os.Remove(tmp)
	}
	if release, err := state.HoldFileWithin(filepath.Join(dir, ".lock"), 0); err == nil {
		trimReceipts(dir, name)
		release()
	}
	return path, nil
}

func writeReceiptFile(w io.Writer, report string, structured any) error {
	if _, err := io.WriteString(w, strings.TrimSuffix(report, "\n")+"\n"); err != nil {
		return err
	}
	if structured == nil {
		return nil
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return err
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(structured)
}

// trimReceipts removes kept receipts until the newest keptReceiptsMax stay,
// never the one just kept and never one younger than trimGrace, and removes the
// temporary files of a writer that died.
func trimReceipts(dir, keep string) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var names []string
	for _, e := range ents {
		switch {
		case strings.HasPrefix(e.Name(), ".tmp-"):
			if fi, err := e.Info(); err == nil && time.Since(fi.ModTime()) > time.Hour {
				_ = os.Remove(filepath.Join(dir, e.Name()))
			}
		case strings.HasSuffix(e.Name(), ".receipt") && e.Name() != keep:
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for i := 0; i < len(names)-(keptReceiptsMax-1); i++ {
		p := filepath.Join(dir, names[i])
		if fi, err := os.Stat(p); err == nil && time.Since(fi.ModTime()) < trimGrace {
			continue
		}
		_ = os.Remove(p)
	}
}
