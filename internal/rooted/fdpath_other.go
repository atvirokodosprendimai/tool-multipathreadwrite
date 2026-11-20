//go:build !linux && !darwin

package rooted

import "os"

// fdPath is unavailable here: the fallback then compares the descriptor with
// where the name resolves again.
func fdPath(*os.File) (string, bool) { return "", false }
