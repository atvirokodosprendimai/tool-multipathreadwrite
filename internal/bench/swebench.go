package bench

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
)

// SelectSWEBench is Stage 3's registered selection (BACKLOG, "Pre-registered
// for ADR-156"): the n instance ids with the lowest sha256("adr-156:" + id),
// whatever order the dataset lists them in, returned in that hash order.
func SelectSWEBench(ids []string, n int) []string {
	type h struct{ id, hash string }
	hs := make([]h, 0, len(ids))
	for _, id := range ids {
		s := sha256.Sum256([]byte("adr-156:" + id))
		hs = append(hs, h{id, hex.EncodeToString(s[:])})
	}
	sort.Slice(hs, func(i, j int) bool { return hs[i].hash < hs[j].hash })
	if n > len(hs) {
		n = len(hs)
	}
	out := make([]string, n)
	for i := range out {
		out[i] = hs[i].id
	}
	return out
}
