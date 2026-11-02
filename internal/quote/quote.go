// Package quote marks a hunk whose SEARCH side quoted every line it replaces
// (ADR-190). The compiler of --format=search_replace signs the quoted lines
// with a key only this process holds; apply recomputes the signature from the
// file it holds under the write lock. A caller who types quote= into a plan
// cannot produce a valid mark, so the mark proves the quote and not a claim.
package quote

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// key is made once per process and never written anywhere: a mark from
// another process, or from a plan someone typed, does not verify.
var key = sync.OnceValue(func() []byte {
	k := make([]byte, 32)
	_, err := rand.Read(k)
	if err == nil {
		return k
	}
	panic("quote: cannot draw a key: " + err.Error())
})

// sign is the MAC over the path, the hunk's range, the kept tail and the
// quoted lines, each length-prefixed so no two inputs share a byte string.
func sign(path string, start, end, tail int, quoted []string) string {
	m := hmac.New(sha256.New, key())
	put := func(s string) {
		var n [8]byte
		binary.BigEndian.PutUint64(n[:], uint64(len(s)))
		m.Write(n[:])
		m.Write([]byte(s))
	}
	put(filepath.Clean(path))
	put(strconv.Itoa(start) + "-" + strconv.Itoa(end) + "+" + strconv.Itoa(tail))
	for _, l := range quoted {
		put(l)
	}
	return hex.EncodeToString(m.Sum(nil))
}

// Mark returns the quote= value for a hunk that replaces lines start..end of
// path when the SEARCH block quoted those lines and tail kept lines after
// end (the closer SEARCH and REPLACE share). quoted holds all of them.
func Mark(path string, start, end, tail int, quoted []string) string {
	return strconv.Itoa(tail) + "." + sign(path, start, end, tail, quoted)
}

// Check verifies mark against file, the lines the caller holds under the
// write lock: this process made it for exactly the bytes now at start..end
// plus its tail, and that block occurs once in the file. It returns the tail,
// the number of lines after end the quote covers.
func Check(mark, path string, start, end int, file []string) (int, error) {
	t, mac, ok := strings.Cut(mark, ".")
	tail, err := strconv.Atoi(t)
	if !ok || err != nil || tail < 0 || tail > 1 {
		return 0, fmt.Errorf("quote= is not a mark this mrw made")
	}
	if start < 1 || end < start || end+tail > len(file) {
		return 0, fmt.Errorf("quote= names lines the file no longer has")
	}
	block := file[start-1 : end+tail]
	got, err := hex.DecodeString(mac)
	want, _ := hex.DecodeString(sign(path, start, end, tail, block))
	if err != nil || !hmac.Equal(got, want) {
		return 0, fmt.Errorf("quote= no longer matches %d-%d: the quoted lines changed since the SEARCH block was compiled, or the mark was not made by this mrw", start, end+tail)
	}
	if n := count(file, block); n != 1 {
		return 0, fmt.Errorf("quote= matches %d times now; it must match once, at %d-%d", n, start, end+tail)
	}
	return tail, nil
}

// count is how many times block occurs in file as whole lines.
func count(file, block []string) int {
	n := 0
	for i := range file {
		if i+len(block) <= len(file) && slices.Equal(file[i:i+len(block)], block) {
			n++
		}
	}
	return n
}
