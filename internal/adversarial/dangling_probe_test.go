package adversarial

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Dangling high-impact leftovers (docs/specs/2026-09-16-dangling-high-impact.md).
// Oracle is that spec's Facts, not the engine. Each test pins a recipe or an
// unrun≠coverage rule already filed in BACKLOG / ADR-002. Deleting the sentence
// is the mutant.

func backlog(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "docs", "adr", "BACKLOG.md"))
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join(strings.Fields(string(b)), " ")
}

func windowAfter(t *testing.T, got, needle string, n int) string {
	t.Helper()
	i := strings.Index(got, needle)
	if i < 0 {
		t.Fatalf("BACKLOG has no %q", needle)
	}
	w := got[i:]
	if len(w) > n {
		w = w[:n]
	}
	return w
}

// TestADesktopReachRecipeNamesTreesPerSessionAndRootsList is UC-1 happy:
// remaining reach work is a Desktop-population trees-per-session measure and
// whether Desktop sends roots/list.
func TestADesktopReachRecipeNamesTreesPerSessionAndRootsList(t *testing.T) {
	got := backlog(t)
	w := windowAfter(t, got, "MCP coverage for the Desktop population", 8000)
	if !strings.Contains(w, "Desktop-population measurement") {
		t.Error("Desktop reach receipt does not name a Desktop-population measurement")
	}
	if !strings.Contains(w, "how many trees one session actually needs") {
		t.Error("Desktop reach receipt does not name trees-per-session")
	}
	if !strings.Contains(w, "roots/list") {
		t.Error("Desktop reach receipt does not name roots/list")
	}
}

// TestACoderPlanCountIsNotDesktopReachEvidence: the ~40-plan coder count is
// the wrong population and must not be quoted as Desktop evidence.
func TestACoderPlanCountIsNotDesktopReachEvidence(t *testing.T) {
	got := backlog(t)
	w := windowAfter(t, got, "THE MULTI-ROOT MEASUREMENT ABOVE WAS TAKEN ON THE WRONG POPULATION", 1200)
	if !strings.Contains(w, "not evidence about Desktop") {
		t.Error("the wrong-population correction no longer says it is not evidence about Desktop")
	}
}

// TestAnUnderCeilingHostCutRecipeIsFiledAndNotClosedByAdr039 is UC-2 happy.
func TestAnUnderCeilingHostCutRecipeIsFiledAndNotClosedByAdr039(t *testing.T) {
	got := backlog(t)
	w := windowAfter(t, got, "A host-truncation measurement of an under-ceiling result", 800)
	if !strings.Contains(w, "reading-18") && !strings.Contains(w, "Reading-18") {
		t.Error("under-ceiling recipe does not name reading 18")
	}
	if !strings.Contains(w, "ADR-039") {
		t.Error("under-ceiling recipe does not name ADR-039")
	}
	if !strings.Contains(w, "rather than treating ADR-039") {
		t.Error("under-ceiling recipe no longer says not to treat ADR-039 as that evidence")
	}
}

// TestAnUnrunUnderCeilingProbeIsNotTheClassClosed: deferred stays deferred.
func TestAnUnrunUnderCeilingProbeIsNotTheClassClosed(t *testing.T) {
	got := backlog(t)
	w := windowAfter(t, got, "A host-truncation measurement of an under-ceiling result", 800)
	if !strings.Contains(w, "Deferred") {
		t.Error("unrun under-ceiling probe is no longer marked Deferred")
	}
}

// TestConcurrentLastWriterWinsIsFiledAsAnAcceptedSilentAppliedRisk is UC-3 happy.
func TestConcurrentLastWriterWinsIsFiledAsAnAcceptedSilentAppliedRisk(t *testing.T) {
	got := backlog(t)
	w := windowAfter(t, got, "Two concurrent writes to one file", 2500)
	if !strings.Contains(w, "last-writer-wins") {
		t.Error("concurrent-writes entry does not name last-writer-wins")
	}
	if !strings.Contains(w, "receipt that says applied") {
		t.Error("concurrent-writes entry does not record a silent applied receipt")
	}
}

// TestLockingStaysPermanentlyOutOfScopeForConcurrentWrites is UC-3 failure:
// a lock is not this spec's outcome.
func TestLockingStaysPermanentlyOutOfScopeForConcurrentWrites(t *testing.T) {
	got := backlog(t)
	w := windowAfter(t, got, "Two concurrent writes to one file", 2500)
	if !strings.Contains(w, "Locking stays permanently out of scope") {
		t.Error("concurrent-writes entry no longer says locking stays permanently out of scope")
	}
	if !strings.Contains(w, "ADR-002") {
		t.Error("concurrent-writes entry no longer names ADR-002 as that scope")
	}
}

// TestTheStrictBalanceDefaultCampaignCriterionIsFiled is UC-4 happy.
func TestTheStrictBalanceDefaultCampaignCriterionIsFiled(t *testing.T) {
	got := backlog(t)
	w := windowAfter(t, got, "Pre-registration: a default `--strict-balance`", 1600)
	if !strings.Contains(w, "Zeus") || !strings.Contains(w, "playtrix") {
		t.Error("campaign criterion does not name Zeus and playtrix")
	}
	if !strings.Contains(w, "this repository") {
		t.Error("campaign criterion does not name this repository")
	}
	if !strings.Contains(w, "false positives under 5%") {
		t.Error("campaign criterion does not name false positives under 5%")
	}
	if !strings.Contains(w, "at least 50 refusals") {
		t.Error("campaign criterion does not name at least 50 refusals")
	}
}

// TestAnUnrunStrictBalanceCampaignDoesNotQualifyADefault: TP-without-FP is not a pass.
func TestAnUnrunStrictBalanceCampaignDoesNotQualifyADefault(t *testing.T) {
	got := backlog(t)
	w := windowAfter(t, got, "Pre-registration: a default `--strict-balance`", 1600)
	if !strings.Contains(w, "true-positive count without the false-positive count does not qualify") {
		t.Error("campaign criterion no longer refuses a TP-without-FP report")
	}
	if !strings.Contains(w, "default stays off") {
		t.Error("campaign criterion no longer says the default stays off")
	}
}

// TestAJsxNestProbeRecipeIsFiledAsUnmeasured is UC-5 happy.
func TestAJsxNestProbeRecipeIsFiledAsUnmeasured(t *testing.T) {
	got := backlog(t)
	w := windowAfter(t, got, "balanced-but-wrongly-nested JSX", 800)
	if !strings.Contains(w, "tsc") {
		t.Error("JSX probe does not name tsc")
	}
	if !strings.Contains(strings.ToLower(w), "dom") {
		t.Error("JSX probe does not name DOM")
	}
}

// TestAnUnrunJsxNestProbeIsNotAFinding: unrun is a probe, not a finding.
func TestAnUnrunJsxNestProbeIsNotAFinding(t *testing.T) {
	got := backlog(t)
	w := windowAfter(t, got, "balanced-but-wrongly-nested JSX", 800)
	if !strings.Contains(w, "not as a finding") {
		t.Error("JSX probe no longer says it is not a finding")
	}
}

// TestAJsxNestFixtureKeepsTheExtraWrapper pins the 2026-09-16 probe artifact.
// Tidying App.tsx so #inner's parent is #intended-parent is the mutant.
func TestAJsxNestFixtureKeepsTheExtraWrapper(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "docs", "break", "jsx-nest", "App.tsx"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	wrap := strings.Index(s, `id="accidental-wrapper"`)
	inner := strings.Index(s, `id="inner"`)
	if !strings.Contains(s, `id="intended-parent"`) {
		t.Error("fixture lost #intended-parent")
	}
	if wrap < 0 || inner < 0 || wrap > inner {
		t.Error("#inner is no longer inside the extra wrapper; the attack was tidied")
	}
}
