package mcp

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// Windows' clock is coarse enough that receipts kept in one burst carry the same
// instant. The newest-by-name trim then ordered them by the random salt, and
// spared arbitrary ones (the Windows shard of #739). Two receipts of one process
// are ordered by when they were kept, whatever the clock says.
func TestReceiptsKeptAtOneInstantAreTrimmedInTheOrderTheyWereKept(t *testing.T) {
	restoreGrace, restoreClock := trimGrace, receiptClock
	t.Cleanup(func() { trimGrace, receiptClock = restoreGrace, restoreClock })
	trimGrace = 0
	frozen := time.Date(2026, 11, 20, 15, 0, 0, 0, time.UTC)
	receiptClock = func() time.Time { return frozen }
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	var paths []string
	for i := 0; i < keptReceiptsMax+8; i++ {
		p, err := keepFullReceipt(fmt.Sprint("receipt ", i), nil)
		if err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	for i, p := range paths {
		_, err := os.Stat(p)
		if wantKept := i >= len(paths)-keptReceiptsMax; (err == nil) != wantKept {
			t.Errorf("receipt %d: kept=%v, want %v (the newest %d stay, whatever the clock says)", i, err == nil, wantKept, keptReceiptsMax)
		}
	}
	// The clock going backwards is no different from a standing still one.
	receiptClock = func() time.Time { return frozen.Add(-time.Hour) }
	p, err := keepFullReceipt("later", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Errorf("a receipt kept after the clock stepped back is not there: %v", err)
	}
	if _, err := os.Stat(paths[len(paths)-keptReceiptsMax]); err == nil {
		t.Error("the receipt kept after the clock stepped back did not take the oldest survivor's place: it was ordered before it")
	}
}
