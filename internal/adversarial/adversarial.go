// Package adversarial holds tests whose only goal is to make mrw fail loudly,
// and to prove it when it does not.
//
// It is deliberately separate from each package's own tests. Those are written
// by the author of the behaviour and ask "does this do what I meant"; these are
// written against the promises the README and the ADRs make to a caller, and
// ask the opposite question — where can a caller be told everything went well
// while something did not.
//
// The rule every test here follows: assert the PROMISE, never the current
// behaviour. A test that documents a defect by expecting it goes green on the
// day the defect is fixed and green on the day it gets worse, which is the
// silent-success failure this whole tool exists to refuse.
//
// TestKnownGap_… is the exception, and it is deliberately narrow. It is for a
// behaviour NO promise settles yet: the test pins today's answer, fails in
// EITHER direction, and its failure message names the decision that has to be
// made and says to delete the test once it is. It is a tripwire on an open
// question, not an endorsement of the behaviour — a gap pinned this way must
// never be read as "this is how it should work".
package adversarial
