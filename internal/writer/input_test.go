package writer

import (
	"reflect"
	"regexp"
	"testing"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/plan"
)

func TestInputFromHunkCarriesTheResolvedPathAndEveryApplyField(t *testing.T) {
	start, end := regexp.MustCompile("start"), regexp.MustCompile("end")
	h := plan.Hunk{
		Path: "@1", Op: "replace", Addr: plan.Addr{Start: 2, End: 3, StartPat: start, EndPat: end, RelEnd: 4},
		Body: []string{"replacement"}, SHA: "sha", Lines: 5, Anchor: "anchor", Quote: "quote",
		Occurrence: 6, CountedBody: true, SrcLine: 7,
		BodyFrom: "source", BodyFromAddr: plan.Addr{Start: 8, End: 9, RelEnd: 10},
	}
	want := apply.Input{
		Path: "target", Start: 2, End: 3, Op: "replace", StartPat: start, EndPat: end, RelEnd: 4,
		Body: []string{"replacement"}, SHA: "sha", Lines: 5, Anchor: "anchor", Quote: "quote",
		Occurrence: 6, CountedBody: true, SrcLine: 7,
		From: "source", FromStart: 8, FromEnd: 9, FromRelEnd: 10,
	}
	if got := InputFromHunk(h, "target"); !reflect.DeepEqual(got, want) {
		t.Fatalf("converted hunk = %+v, want %+v", got, want)
	}
}
