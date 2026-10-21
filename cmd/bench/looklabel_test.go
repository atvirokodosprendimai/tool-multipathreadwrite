package main

import (
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/bench"
)

// TestAnOpenPairOfACompleteStageSaysTheStageEnded: a pair still open when the
// stage is complete has no next look, so its row must not read as if one is due.
func TestAnOpenPairOfACompleteStageSaysTheStageEnded(t *testing.T) {
	for _, c := range []struct {
		rep  bench.LookReport
		row  bench.LookRow
		want string
	}{
		{bench.LookReport{Blocks: 3}, bench.LookRow{}, "open after look 3"},
		{bench.LookReport{Blocks: 5, Complete: true}, bench.LookRow{}, "open when the stage ended at look 5"},
		{bench.LookReport{Blocks: 5, Complete: true}, bench.LookRow{ClosedAt: 1}, "closed at look 1"},
		{bench.LookReport{Blocks: 2}, bench.LookRow{ClosedAt: 2}, "closed at look 2"},
	} {
		if got := lookLabel(c.rep, c.row); got != c.want {
			t.Errorf("lookLabel(%+v, closed at %d) = %q, want %q", c.rep, c.row.ClosedAt, got, c.want)
		}
	}
}
