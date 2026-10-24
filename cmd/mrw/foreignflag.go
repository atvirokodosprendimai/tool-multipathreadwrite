package main

import (
	"fmt"
	"slices"
	"strings"

	"github.com/urfave/cli/v3"
)

// foreignFlags maps a flag a caller types on `mrw read` — one grep, ripgrep or
// ack has and read does not — to what mrw spells instead (ADR-140). Every
// equivalent was run against the built binary on 2026-10-10, not read from a
// document. -v and -o were left out then for having no equivalent; ADR-164 T5
// gives them the honest pointer that there is none. -z still has no pointer.
// A flag mrw itself defines, typed after a subcommand that refused it, is
// answered before this table, with its home spelling (rootFlagHint, ADR-165).
var foreignFlags = map[string]string{
	"i":       "case-insensitive: start the pattern with (?i), as in --grep '(?i)PATTERN'",
	"n":       "line numbers are printed by default; -N drops them",
	"r":       "a directory you name is walked; there is no recursion flag",
	"R":       "a directory you name is walked; there is no recursion flag",
	"l":       "--grep PATTERN --stat lists the matching files with their length, size and sha",
	"c":       "--grep PATTERN --count counts the lines that match in each file",
	"A":       "context is symmetric: -C N prints N lines either side of a match",
	"B":       "context is symmetric: -C N prints N lines either side of a match",
	"e":       "the pattern is the value of --grep; one that starts with a dash is written --grep=-PATTERN",
	"E":       "--grep patterns are already extended regular expressions (Go RE2)",
	"P":       "--grep patterns are Go RE2: no look-around and no backreferences",
	"F":       `--grep patterns are regular expressions; wrap a literal in \Q…\E, or escape its metacharacters one by one if it contains \E`,
	"w":       `wrap the word in \b…\b; Go's \b is ASCII-only, so a word with non-ASCII letters is not bounded correctly`,
	"H":       "every file is announced by a ==> header; there is no flag for it",
	"include": "there is no --include: name the directories to walk or pipe a list to --files-from; --exclude GLOB drops files",
	"m":       "--max-lines N caps the lines served per spec; what is withheld is reported",
	"v":       "there is no inverted match: --grep serves the lines that match",
	"o":       "there is no match-only output: --max-cols N cuts each line to a window round its match",
}

// planFlags maps a flag a caller types on `mrw write` to name the plan — the
// argument other tools take as --file or --input — to the argument's spelling
// (ADR-151). -f and -p are left out: they read as --force and patch -p, so the
// plan sentence would be a guess.
var planFlags = map[string]string{
	"plan":  planArgument,
	"file":  planArgument,
	"input": planArgument,
	"stdin": planArgument,
}

const planArgument = "the plan is an argument, not a flag: mrw write PLAN, or mrw write - to read it from standard input"

// flagTables is each command's table of flags it lacks and what it spells
// instead, by the command's full name.
var flagTables = map[string]map[string]string{
	"mrw read":  foreignFlags,
	"mrw write": planFlags,
}

// foreignFlagHint is the sentence for flag on the command named full, or "". A
// command with no table is worded as before.
func foreignFlagHint(full, flag string) string {
	return flagTables[full][flag]
}

// refusedFlag is the flag name urfave/cli refused, as typed without its dash, or
// "" when err is not that refusal (the same handle subcommandForFlag uses).
func refusedFlag(err error) string {
	name, ok := strings.CutPrefix(err.Error(), "flag provided but not defined: -")
	if !ok {
		return ""
	}
	return name
}

// patternFlagHint is the sentence for a --grep pattern that is exactly a dash
// and one letter of the table: `mrw read --grep -i needle` takes -i as the
// pattern and needle as a path, and ends "no file matched /-i/".
func patternFlagHint(pattern string) string {
	if len(pattern) != 2 || pattern[0] != '-' {
		return ""
	}
	return foreignFlags[pattern[1:]]
}

// readFlagPatternRefusal is the refusal for a --grep value that is exactly the
// spelling of a flag the command defines: `mrw read --grep -C 2 PATTERN f` takes
// -C as the pattern and PATTERN as a path. "" when the pattern is no such
// spelling. --grep=-C reaches the same value, so both are refused; a pattern
// that merely starts with a dash is untouched.
func readFlagPatternRefusal(flags []cli.Flag, pattern string) string {
	name, ok := strings.CutPrefix(pattern, "-")
	if !ok {
		return ""
	}
	long := strings.HasPrefix(name, "-")
	name = strings.TrimPrefix(name, "-")
	for _, f := range flags {
		if slices.Contains(f.Names(), name) && long == (len(name) > 1) {
			// Double quotes: cmd.exe keeps single quotes as characters, and every
			// supported shell takes this form (ADR-078's cmdExeForm trap).
			return fmt.Sprintf("--grep took %q as its pattern; put the pattern right after --grep: mrw read --grep PATTERN %s … (to search for the literal, write --grep=\"(?:%s)\")", pattern, pattern, pattern)
		}
	}
	return ""
}
