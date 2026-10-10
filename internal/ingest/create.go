package ingest

import (
	"fmt"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/lines"
	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/rooted"
)

// CompileCreate turns a file's content into the native plan text that creates
// path, so `mrw write --create PATH` is a plan the caller did not have to count
// (ADR-139). The file is what a create plan makes of the content's lines: each
// ends in a newline, so a missing final newline is added and CRLF becomes LF,
// and empty content makes an empty file. Content mrw cannot split into lines is
// refused, since a create plan cannot carry it either. The function writes
// nothing.
func CompileCreate(path string, content []byte) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("--create needs a PATH")
	}
	if rooted.IsRooted(path) {
		return nil, fmt.Errorf("--create %s: the path is not relative to the root", path)
	}
	if why := lines.Unsplittable(content); why != "" {
		return nil, fmt.Errorf("--create %s: the content %s, so mrw cannot carry it as lines", path, why)
	}
	ls, _, _ := lines.Split(string(content))
	return []byte(emit("create", path, 0, 0, ls, "")), nil
}
