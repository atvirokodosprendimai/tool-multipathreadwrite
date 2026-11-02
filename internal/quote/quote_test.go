package quote

import (
	"math"
	"testing"
)

func TestARangeNearTheIntegerLimitIsRefusedNotPanicked(t *testing.T) {
	file := []string{`a`, `b`}
	for _, end := range []int{math.MaxInt, math.MaxInt - 1} {
		_, err := Check(`1.x`, `fx`, 1, end, file)
		if err == nil {
			t.Fatal(end)
		}
	}
}

func TestAnOutOfRangeQuoteReadsLikeAChangedOne(t *testing.T) {
	file := []string{`a`, `b`}
	_, short := Check(`0.00`, `fx`, 1, 9, file)
	_, moved := Check(`0.00`, `fx`, 1, 2, file)
	if short == nil || moved == nil {
		t.Fatal(short, moved)
	}
	if short.Error()[:19] != moved.Error()[:19] {
		t.Error(short, moved)
	}
}
