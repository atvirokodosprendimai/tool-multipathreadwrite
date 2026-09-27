// Package refusal names the kinds of refusal the plan parser and the apply
// engine share (ADR-087), so the two sites — and the MCP surface that adds a
// remedy to one of them — compare kinds instead of message text.
package refusal

import (
	"errors"
	"fmt"
)

// Kind names one refusal. The zero value means "not classified": most
// refusals carry no kind, and a caller must not read "" as any of these.
type Kind string

// The kinds the parser and the engine both refuse (ADR-030 mirrors each rule
// at the engine boundary), and the engine's not-read refusal, which the MCP
// surface extends with the acknowledgement remedy.
const (
	CreateEmptyBody  Kind = "create-empty-body"
	CreateAddress    Kind = "create-address"
	CreateRelEnd     Kind = "create-relative-end"
	CreateGuard      Kind = "create-guard"
	InsertRange      Kind = "insert-range"
	InsertEmptyBody  Kind = "insert-empty-body"
	ReplaceEmptyBody Kind = "replace-empty-body"
	NotRead          Kind = "not-read"
)

// Error is a refusal with its kind. Its text is the message exactly, so a
// caller that prints it sees what it saw before kinds existed.
type Error struct {
	Kind Kind
	Msg  string
}

func (e *Error) Error() string { return e.Msg }

// New builds a kinded refusal from a format, as fmt.Errorf does.
func New(k Kind, format string, a ...any) error {
	return &Error{Kind: k, Msg: fmt.Sprintf(format, a...)}
}

// KindOf reports the kind err carries, through any wrapping, or "" for none.
func KindOf(err error) Kind {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return ""
}
