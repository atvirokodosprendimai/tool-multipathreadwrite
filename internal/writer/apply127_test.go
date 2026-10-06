package writer

import "github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/apply"

// Apply is applyCounted without the counter, for the tests that drive a write
// under the lock and do not read the counter. Production lands through Land
// (ADR-127), so this lives with the tests (static analysis: nothing in
// production may be reached only by tests).
func Apply(root string, in []apply.Input, opt apply.Options) (apply.Result, error) {
	res, _, err := applyCounted(root, in, opt)
	return res, err
}
