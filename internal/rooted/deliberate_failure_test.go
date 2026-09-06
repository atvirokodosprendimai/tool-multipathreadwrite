package rooted

import "testing"

// TestDeliberateFailureForShardProof exists for ONE CI run and is deleted with
// the branch that carries it. It proves the `windows` aggregate job reports
// FAILURE when a shard fails — an aggregate that cannot go red would be worse
// than the single job it replaced, because it would look like a gate.
func TestDeliberateFailureForShardProof(t *testing.T) {
	t.Fatal("deliberate: proving the windows aggregate goes red when a shard fails")
}
