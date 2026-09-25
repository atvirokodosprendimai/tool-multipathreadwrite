"""Unit tests for blind-score.py: one synthetic transcript per shape the scorer
misread (docs/adr/BACKLOG.md, reading 03 and the Codex review of #206; ADR-070
T3). Run: python3 -m unittest discover -s scripts -p 'test_blind_score.py' -v
"""
import importlib.util
import json
import tempfile
import unittest
from pathlib import Path

_spec = importlib.util.spec_from_file_location("blind_score", Path(__file__).with_name("blind-score.py"))
bs = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(bs)

KEY = {
    "t1": [["a.go", 1, "x"]], "t2": 2, "t3": "three", "t4": [["b.go", 4]], "t5": ["five"],
    "t6": "six.go", "t7": "seven", "t8": {"ambiguous_exit": 1, "final": "meta\n"},
    "t9": {"exit": 0, "notes": "notes\n"},
}
ANSWER = {
    "t1": [["a.go", 1, "x"]], "t2": 2, "t3": "three", "t4": [["b.go", 4]], "t5": ["five"],
    "t6": "six.go", "t7": "seven", "t8": {"ambiguous_exit": 1}, "t9": {"exit": 0},
}


def trial(commands, final_text):
    """A trial directory and a transcript: one Bash call per command, then one
    assistant text carrying final_text."""
    d = Path(tempfile.mkdtemp())
    (d / "answer-key.json").write_text(json.dumps(KEY))
    (d / "tree" / "docs").mkdir(parents=True)
    (d / "tree/docs/meta.yaml").write_text("meta\n")
    (d / "tree/docs/notes.txt").write_text("notes\n")
    lines = [json.dumps({"message": {"role": "assistant", "content": [
        {"type": "tool_use", "name": "Bash", "input": {"command": c}}]}}) for c in commands]
    lines.append(json.dumps({"message": {"role": "assistant", "content": [
        {"type": "text", "text": final_text}]}}))
    t = d / "transcript.jsonl"
    t.write_text("\n".join(lines) + "\n")
    return d, t


def fence(obj):
    return "```json\n" + json.dumps(obj) + "\n```"


class T(unittest.TestCase):
    def test_backslash_newline_is_one_command(self):
        self.assertEqual(bs.command_words("mrw read a.go \\\n  b.go"), ["mrw"])

    def test_heredoc_body_is_not_commands(self):
        words = bs.command_words("mrw write - <<'EOF'\n@@ a 1 replace\nls\ngrep x\nEOF\nmrw read a")
        self.assertEqual(words, ["mrw", "mrw"])

    def test_command_cat_is_banned(self):
        d, t = trial(["command cat f"], fence(ANSWER))
        self.assertEqual(bs.score(d, t)["verdict"], "VOID")

    def test_cat_in_substitution_is_banned(self):
        d, t = trial(['x="$(cat f)"; mrw read a'], fence(ANSWER))
        self.assertEqual(bs.score(d, t)["verdict"], "VOID")

    def test_env_mrw_is_counted(self):
        d, t = trial(["env mrw read a", "env -i FOO=1 mrw read b"], fence(ANSWER))
        self.assertEqual(bs.score(d, t)["mrw_calls"], 2)

    def test_quoted_banned_word_is_not_a_violation(self):
        d, t = trial(['mrw read a; echo "next: cat or --help"'], fence(ANSWER))
        r = bs.score(d, t)
        self.assertEqual(r["verdict"], "MEETS", r)
        d, t = trial(["mrw write --help"], fence(ANSWER))
        self.assertEqual(bs.score(d, t)["verdict"], "VOID")
        d, t = trial(["mrw read a; git --help"], fence(ANSWER))
        self.assertEqual(bs.score(d, t)["verdict"], "VOID")

    def test_non_object_final_fence_is_not_skipped(self):
        d, t = trial(["mrw read a"], fence(ANSWER) + "\n" + "```json\n[1, 2]\n```")
        r = bs.score(d, t)
        self.assertEqual(r["verdict"], "MISS", r)
        self.assertIn("not an object", r.get("reason", ""))

    def test_mistyped_answer_does_not_crash(self):
        bad = dict(ANSWER, t1="not a list", t4=[1, 2], t8="not a dict")
        d, t = trial(["mrw read a"], fence(bad))
        r = bs.score(d, t)
        self.assertEqual(r["correct"], 6, r)

    def test_zero_mrw_calls_cannot_meet(self):
        d, t = trial(["echo hi"], fence(ANSWER))
        r = bs.score(d, t)
        self.assertEqual(r["correct"], 9, r)
        self.assertEqual(r["verdict"], "MISS", r)
    # ADR-070 T4, from the Codex review of v1.25.0.
    def test_command_v_is_a_lookup_not_a_call(self):
        d, t = trial(["command -v mrw"], fence(ANSWER))
        r = bs.score(d, t)
        self.assertEqual((r["mrw_calls"], r["verdict"]), (0, "MISS"), r)
        d, t = trial(["mrw read a; command -v cat"], fence(ANSWER))
        self.assertEqual(bs.score(d, t)["verdict"], "MEETS")

    def test_wrapper_option_operands_are_skipped(self):
        d, t = trial(["env -u SOME_VAR mrw read a", "xargs -n 1 mrw read < list"], fence(ANSWER))
        self.assertEqual(bs.score(d, t)["mrw_calls"], 2)
        d, t = trial(["mrw read a; xargs -I {} cat {} < list"], fence(ANSWER))
        self.assertEqual(bs.score(d, t)["verdict"], "VOID")

    def test_wrapper_help_is_banned(self):
        d, t = trial(["mrw read a; env --help"], fence(ANSWER))
        self.assertEqual(bs.score(d, t)["verdict"], "VOID")

    def test_expandable_heredoc_substitution_is_scanned(self):
        d, t = trial(["mrw read a; printf x <<EOF\n$(cat f)\nEOF"], fence(ANSWER))
        self.assertEqual(bs.score(d, t)["verdict"], "VOID")
        d, t = trial(["mrw read a; printf x <<'EOF'\n$(cat f)\nEOF"], fence(ANSWER))
        self.assertEqual(bs.score(d, t)["verdict"], "MEETS")

    def test_escaped_heredoc_substitution_is_literal(self):
        d, t = trial(["mrw read a; printf x <<EOF\n\\$(cat f) \\`cat g\\`\nEOF"], fence(ANSWER))
        self.assertEqual(bs.score(d, t)["verdict"], "MEETS")
        d, t = trial(["mrw read a; printf x <<EOF\n\\\\$(cat f)\nEOF"], fence(ANSWER))
        self.assertEqual(bs.score(d, t)["verdict"], "VOID")

    def test_escaped_backslash_does_not_manufacture_a_substitution(self):
        d, t = trial(["mrw read a; printf x <<EOF\n$\\\\(cat f)\nEOF"], fence(ANSWER))
        self.assertEqual(bs.score(d, t)["verdict"], "MEETS")

    def test_env_split_string_runs_its_operand(self):
        d, t = trial(["mrw read a; env -S cat go.mod"], fence(ANSWER))
        self.assertEqual(bs.score(d, t)["verdict"], "VOID")
        d, t = trial(["env -S 'mrw read a'"], fence(ANSWER))
        self.assertEqual(bs.score(d, t)["mrw_calls"], 1)



if __name__ == "__main__":
    unittest.main()
