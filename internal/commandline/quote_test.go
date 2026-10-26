package commandline

import "testing"

// A value is written so that the shell it is pasted into reads it as one
// argument and as nothing else, on both systems.
//
// The names with code in them are the ones an outside review gave the parser
// of PowerShell on 2026-10-22: the line carried them bare or in double quotes,
// and the parser found a second command in each.
func TestAValueReachesTheShellAsOneArgumentAndNothingElse(t *testing.T) {
	for _, given := range []struct {
		value, windows, posix string
	}{
		{"contacts.csv", "contacts.csv", "contacts.csv"},
		{"a-directory/out_1", "a-directory/out_1", "a-directory/out_1"},
		{"10%", "10%", "10%"},
		{`C:\data\sample.csv`, `C:\data\sample.csv`, `'C:\data\sample.csv'`},
		{`C:\My Documents\a.csv`, `'C:\My Documents\a.csv'`, `'C:\My Documents\a.csv'`},
		{`C:\data\sample;Get-Date;.csv`, `'C:\data\sample;Get-Date;.csv'`, `'C:\data\sample;Get-Date;.csv'`}, // punctuation: test data
		{`C:\data\sample$(Write-Output SAFE).csv`, `'C:\data\sample$(Write-Output SAFE).csv'`,
			`'C:\data\sample$(Write-Output SAFE).csv'`},
		{`say "hi".csv`, `'say "hi".csv'`, `'say "hi".csv'`},
		{"it's.csv", "'it''s.csv'", `'it'\''s.csv'`},
		{"it\u2019s.csv", "'it\u2019\u2019s.csv'", "'it\u2019s.csv'"},
		{",", "','", "','"},
		{"L2,L3", "'L2,L3'", "'L2,L3'"},
		{"~/data", "'~/data'", "'~/data'"},
		{"żółw.csv", "'żółw.csv'", "'żółw.csv'"},
	} {
		if got := quoteFor(given.value, true); got != given.windows {
			t.Errorf("on Windows %q is written %s, and PowerShell needs %s", given.value, got, given.windows)
		}
		if got := quoteFor(given.value, false); got != given.posix {
			t.Errorf("elsewhere %q is written %s, and sh needs %s", given.value, got, given.posix)
		}
	}
}
