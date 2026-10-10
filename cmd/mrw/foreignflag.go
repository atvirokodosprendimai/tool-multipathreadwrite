package main

import "strings"

// foreignFlags maps a flag a caller types on `mrw read` — one grep, ripgrep or
// ack has and read does not — to what mrw spells instead (ADR-140). Every
// equivalent was run against the built binary on 2026-10-10, not read from a
// document; a flag with no equivalent and no honest pointer (-v, -o, -z) is not
// here and gets the old message.
var foreignFlags = map[string]string{
	"i":       "case-insensitive: start the pattern with (?i), as in --grep '(?i)PATTERN'",
	"n":       "line numbers are printed by default; -N drops them",
	"r":       "a directory you name is walked; there is no recursion flag",
	"R":       "a directory you name is walked; there is no recursion flag",
	"l":       "--grep PATTERN --stat lists the matching files with their length, size and sha",
	"c":       "there is no match count; --grep PATTERN --stat lists the matching files",
	"count":   "there is no match count; --grep PATTERN --stat lists the matching files",
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
}

// foreignFlagHint is the sentence for flag on the command named full, or "". Only
// `mrw read` has a table; any other command is worded as before.
func foreignFlagHint(full, flag string) string {
	if full != "mrw read" {
		return ""
	}
	return foreignFlags[flag]
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
