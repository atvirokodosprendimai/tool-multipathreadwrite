package guide

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// skillParts splits a SKILL.md into its frontmatter lines and its body, or
// fails the test: a host reads nothing past a frontmatter it cannot find.
func skillParts(t *testing.T, s string) ([]string, string) {
	t.Helper()
	rest, ok := strings.CutPrefix(s, "---\n")
	if !ok {
		t.Fatalf("the skill does not open with a --- frontmatter line:\n%s", s)
	}
	front, body, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		t.Fatalf("the skill's frontmatter is never closed:\n%s", s)
	}
	return strings.Split(front, "\n"), body
}

// ADR-168 Decision 1. The frontmatter keeps the Agent Skills rules
// (agentskills.io/specification, read 2026-10-22): a name of lowercase letters,
// digits and single hyphens, at most 64, equal to the folder it is installed in;
// a description of 1 to 1024 characters; metadata a map of strings. The
// description is a folded block, the form ADR-063 T1's fence reads, and names
// the trigger. Each stamp, whatever it holds, comes back as the string it was.
func TestTheSkillIsAValidAgentSkill(t *testing.T) {
	for _, stamp := range []string{"v1.72.0 (18ebc2f)", "dev", "1.0", `we"ird \ stamp: yes`} {
		front, body := skillParts(t, Skill(stamp))
		if front[0] != "name: mrw" {
			t.Errorf("the first frontmatter line is %q, want name: mrw (the folder --dir installs into)", front[0])
		}
		if name := strings.TrimPrefix(front[0], "name: "); len(name) > 64 || !regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`).MatchString(name) {
			t.Errorf("name %q breaks the Agent Skills name rule", name)
		}
		if front[1] != "description: >-" {
			t.Fatalf("the second frontmatter line is %q, want the folded block description: >-", front[1])
		}
		var desc []string
		i := 2
		for ; i < len(front) && strings.HasPrefix(front[i], "  "); i++ {
			desc = append(desc, strings.TrimSpace(front[i]))
		}
		d := strings.Join(desc, " ")
		if len(d) == 0 || len(d) > 1024 {
			t.Errorf("the description is %d bytes, want 1 to 1024", len(d))
		}
		for _, want := range []string{"every file edit", "mrw write", "--grep", "PATH"} {
			if !strings.Contains(d, want) {
				t.Errorf("the description does not name %q: %s", want, d)
			}
		}
		meta := front[i:]
		if len(meta) != 4 || meta[0] != "metadata:" {
			t.Fatalf("after the description want metadata: and three entries, got %q", meta)
		}
		got := map[string]string{}
		for _, l := range meta[1:] {
			k, v, ok := strings.Cut(strings.TrimPrefix(l, "  "), ": ")
			if !ok || !strings.HasPrefix(l, "  ") {
				t.Fatalf("metadata line %q is not an indented key: value", l)
			}
			u, err := strconv.Unquote(v)
			if err != nil || !strings.HasPrefix(v, `"`) {
				t.Fatalf("metadata %s is %s, not a double-quoted string", k, v)
			}
			got[k] = u
		}
		if got["mrw-version"] != stamp || got["generated-by"] != "mrw skill" || got["body-sha256"] != BodySHA(body) {
			t.Errorf("metadata %q, want mrw-version %q, generated-by \"mrw skill\" and the body's sha-256", got, stamp)
		}
		if n := strings.Count(body, "\n"); n >= 500 {
			t.Errorf("the body is %d lines, want under 500", n)
		}
	}
}

// ADR-168 Decision 1. The body is the core and the recipes as the binary prints
// them, in that order, then a closing paragraph that names the full contract
// backticked (the form agentsdoc_test reads), how to reinstall, and the stamp.
func TestTheSkillBodyIsTheCoreAndTheRecipes(t *testing.T) {
	const stamp = "v9.8.7 (abcdef0)"
	_, body := skillParts(t, Skill(stamp))
	core, recipes := strings.Index(body, Core()), strings.Index(body, Recipes())
	if core < 0 || recipes < 0 || recipes < core {
		t.Fatalf("the body does not hold Core() then Recipes() byte for byte (core at %d, recipes at %d)", core, recipes)
	}
	closing := body[recipes+len(Recipes()):]
	for _, want := range []string{"`mrw instructions`", "mrw skill --dir", stamp} {
		if !strings.Contains(closing, want) {
			t.Errorf("the closing paragraph does not name %q:\n%s", want, closing)
		}
	}
}

// ADR-168 Decision 6. The agent that loads the skill is told that a host's
// per-path rules for its Edit and Write tools do not reach mrw, that their
// absence is not permission, and that mrw is not a way past a path the host
// refused: the skill moves the agent off those tools, so it must say so.
func TestTheSkillSaysHostPathRulesDoNotReachMrw(t *testing.T) {
	_, body := skillParts(t, Skill("v9.8.7"))
	closing := body[strings.Index(body, Recipes())+len(Recipes()):]
	for _, want := range []string{
		"per-path rules for its Edit and Write tools do not reach mrw",
		"not permission",
		"a path the host refused",
	} {
		if !strings.Contains(closing, want) {
			t.Errorf("the skill's closing paragraph does not say %q:\n%s", want, closing)
		}
	}
}
