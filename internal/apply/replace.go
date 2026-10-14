package apply

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/atvirokodosprendimai/tool-multipathreadwrite/internal/regular"
)

// sameFileFn is the seam the identity check (ADR-106, ADR-107) asks whether a
// file's identity could be read; on Windows os.SameFile opens the file to load
// its ID and answers false, error swallowed, when that open is refused.
var sameFileFn = os.SameFile

// replaceableFn is the seam Apply asks, before the first rename, whether an
// existing target can be replaced (ADR-125). On Windows a file another process
// holds without sharing delete cannot be renamed over or removed, and found
// at commit that left the files before it written. On unix it answers yes.
var replaceableFn = replaceable

// nameRefused is the cause for a name the system will not make (ADR-125,
// ADR-132). Such a name cannot exist, so nothing mrw tried to make under it is
// left behind (noteIfLeft).
const nameRefused = "not a valid name on this system"

// causeOf names, in the words a caller acts on, why an open of a target
// failed (ADR-125): a permission, another process holding the file, a name the
// system refuses. It answers "" when the error says none of these.
func causeOf(err error) string {
	if errors.Is(err, fs.ErrPermission) {
		return "permission denied"
	}
	return platformCause(err)
}

// openRefusal is the reason a hunk carries when its file could not be opened
// for reading at validation (ADR-125): before, the error left Apply bare and
// the caller saw no receipt.
func openRefusal(path string, err error) string {
	if c := causeOf(err); c != "" {
		return fmt.Sprintf("%s cannot be opened: %s (%v)", path, c, err)
	}
	return fmt.Sprintf("%s cannot be opened: %v", path, err)
}

// identityRefusal is the reason a hunk carries when the file's identity could
// not be read (ADR-107). An open of the file says why when anything can; the
// old advice to send the plan again is kept for when the open succeeds and
// nothing explains the failure (ADR-125).
func identityRefusal(path, full string) string {
	f, _, err := regular.Open(full)
	if err == nil {
		_ = f.Close()
		return fmt.Sprintf("%s: its identity could not be read, so a change before the commit could not be seen; send the plan again", path)
	}
	return fmt.Sprintf("%s: its identity could not be read: %s", path, openRefusal(path, err))
}
