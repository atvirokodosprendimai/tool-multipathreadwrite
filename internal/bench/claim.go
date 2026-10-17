package bench

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

// StatusSentence is the one sentence ADR-162 Decision 3 appends to every
// instruction of a stage run with --status-line, the same in every
// configuration; its SHA-256 is in run.json.
const StatusSentence = "When you have finished, end your reply with one line, exactly `STATUS: done` if the task is complete, or `STATUS: not done: <reason>` if it is not."

// PromptTemplate is how a stage run with --status-line builds every
// instruction: the task's own, a blank line, then StatusSentence. Its SHA-256
// is in run.json as prompt_sha256, so a change to how the prompt is built shows
// as a different digest (the review of T2).
const PromptTemplate = "{instruction}\n\n" + StatusSentence

func promptSHA256() string {
	s := sha256.Sum256([]byte(PromptTemplate))
	return hex.EncodeToString(s[:])
}

var statusLine = regexp.MustCompile(`^STATUS: (done|not done: .+)$`)

func statusLineSHA256() string {
	s := sha256.Sum256([]byte(StatusSentence))
	return hex.EncodeToString(s[:])
}

// claimStatus reads the agent's claim from its final message without
// classifying prose: "done" or "not done" when exactly one line is a STATUS
// line in the registered form, "unreadable" otherwise — none, two, or another
// value.
func claimStatus(result string) string {
	var got []string
	for _, l := range strings.Split(result, "\n") {
		if m := statusLine.FindStringSubmatch(l); m != nil {
			got = append(got, m[1])
		}
	}
	if len(got) != 1 {
		return "unreadable"
	}
	if got[0] == "done" {
		return "done"
	}
	return "not done"
}

// settle is a graded attempt's verdict as the receipt records it. A SWE-bench
// attempt keeps the harness's own verdict in HarnessResolved, and broke_tests
// is always present for it; then ADR-156's budget makes an attempt over it
// unsolved, which Solved alone carries (ADR-162 Decision 3).
func settle(v Verdict, overBudget, swe bool) Verdict {
	if swe {
		resolved := v.Solved
		v.HarnessResolved = &resolved
		if v.BrokeTests == nil {
			broke := false
			v.BrokeTests = &broke
		}
	}
	if overBudget {
		v.Solved = false
	}
	return v
}
