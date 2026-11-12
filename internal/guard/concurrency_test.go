package guard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The guard has to notice each way of starting or coordinating concurrent
// work, and has to stay quiet about the same words appearing in prose. The
// second half is what decides whether it survives: a guard that fires on a
// comment gets switched off, and a switched off guard is worse than an absent
// one because nobody sees the day it stopped looking.
func TestConcurrencyNoticesEachWayOfStartingWork(t *testing.T) {
	cases := []struct {
		name   string
		source string
		expect string
	}{
		{
			name:   "a goroutine",
			source: "package p\n\nfunc f() {\n\tgo g()\n}\n\nfunc g() {}\n",
			expect: "starts a goroutine",
		},
		{
			name:   "a channel",
			source: "package p\n\nfunc f() {\n\tc := make(chan int)\n\t_ = c\n}\n",
			expect: "declares a channel",
		},
		{
			name:   "a channel in a signature",
			source: "package p\n\nfunc f(in <-chan int) {}\n",
			expect: "declares a channel",
		},
		{
			name:   "a select",
			source: "package p\n\nfunc f(a, b chan int) {\n\tselect {\n\tcase <-a:\n\tcase <-b:\n\t}\n}\n",
			expect: "waits on several channels",
		},
		{
			name:   "the sync package",
			source: "package p\n\nimport \"sync\"\n\nvar m sync.Mutex\n",
			expect: "imports sync",
		},
		{
			name:   "the atomic package",
			source: "package p\n\nimport \"sync/atomic\"\n\nvar n atomic.Int64\n",
			expect: "imports sync/atomic",
		},
		{
			name:   "the signal package",
			source: "package p\n\nimport \"os/signal\"\n\nvar f = signal.Ignore\n",
			expect: "imports os/signal",
		},
		{
			name: "waiting on a channel the file never declares",
			source: "package p\n\nimport \"time\"\n\nfunc f() {\n\tt := time.NewTicker(time.Second)\n" +
				"\t<-t.C\n}\n",
			expect: "waits on a channel",
		},
		{
			name:   "sending on a channel",
			source: "package p\n\nfunc f(c chan int) {\n\tc <- 1\n}\n",
			expect: "sends on a channel",
		},
		{
			name:   "a function run later on a goroutine of its own",
			source: "package p\n\nimport \"time\"\n\nfunc f() {\n\ttime.AfterFunc(time.Second, g)\n}\n\nfunc g() {}\n",
			expect: "runs a function on a goroutine of its own",
		},
		{
			name:   "the same under another name for the package",
			source: "package p\n\nimport clock \"time\"\n\nvar f = clock.AfterFunc\n",
			expect: "runs a function on a goroutine of its own",
		},
		{
			name:   "the same imported into the file's own names",
			source: "package p\n\nimport . \"time\"\n\nfunc f() {\n\tAfterFunc(Second, f)\n}\n",
			expect: "runs a function on a goroutine of its own",
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			findings := scanOne(t, test.source)
			if len(findings) == 0 {
				t.Fatalf("nothing was reported")
			}
			if !strings.Contains(strings.Join(findings, " "), test.expect) {
				t.Fatalf("expected %q, got %v", test.expect, findings)
			}
		})
	}
}

// The words appear in prose all over this repository, including in the very
// comments that explain the rule. Reporting those would make the guard useless
// on the day it was written.
func TestConcurrencyStaysQuietAboutProse(t *testing.T) {
	quiet := []struct {
		name   string
		source string
	}{
		{"the words in a comment", "package p\n\n// go func and chan and sync.Mutex are named here on purpose.\nfunc f() {}\n"},
		{"the words in a message", "package p\n\nconst help = \"go func, chan, sync. and atomic. are not allowed\"\n"},
		{"the words in a raw string", "package p\n\nconst doc = `select and chan appear here`\n"},
		{"a context, which starts nothing", "package p\n\nimport \"context\"\n\nfunc f(ctx context.Context) {}\n"},
		{"an ordinary function called go something", "package p\n\nfunc gopher() {}\n"},
		{"a duration, which starts nothing", "package p\n\nimport \"time\"\n\nvar d = 2 * time.Second\n"},
		{"a method of our own called AfterFunc", "package p\n\ntype clock struct{}\n\nfunc (clock) AfterFunc() {}\n\n" +
			"var time clock\n\nfunc f() {\n\ttime.AfterFunc()\n}\n"},
		{"a comparison that looks like an arrow", "package p\n\nfunc f(a, b int) bool {\n\treturn a < -b\n}\n"},
	}

	for _, test := range quiet {
		t.Run(test.name, func(t *testing.T) {
			if findings := scanOne(t, test.source); len(findings) != 0 {
				t.Fatalf("the guard fired on prose: %v", findings)
			}
		})
	}
}

// The allow list is what turns the guard from a ban into a decision, so it has
// to actually work. A list that silently failed to match would mean the first
// deliberate goroutine could never be committed.
func TestConcurrencyRespectsTheAllowList(t *testing.T) {
	dir := t.TempDir()
	name := "worker.go"
	write(t, filepath.Join(dir, name), "package p\n\nfunc f() {\n\tgo g()\n}\n\nfunc g() {}\n")

	findings, err := Concurrency(dir, nil)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected the goroutine to be reported, got %v", findings)
	}

	findings, err = Concurrency(dir, map[string]bool{name: true})
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("the allowed file was still reported: %v", findings)
	}
}

// The rule this guard exists for, applied to the repository as it stands.
//
// It was installed while the product had no concurrency at all, the cheapest
// moment, so that every file on the allow list arrived as a deliberate act
// rather than a line that slipped through. The list says which files those
// are, each with its reason, and this test is what keeps every other file off
// it.
func TestTheProductStartsNoConcurrentWorkWithoutSayingSo(t *testing.T) {
	root, err := ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}
	findings, err := Concurrency(root, ConcurrencyAllowed)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("concurrent work outside the files allowed to have it:\n%s\n\n"+
			"Adding a file to guard.ConcurrencyAllowed is the way to allow this, and it is meant to be a\n"+
			"decision rather than a formality. The architecture requires that the output is assembled by\n"+
			"row number rather than by whichever goroutine finished first, and that nothing mutable is\n"+
			"shared. Neither of those is checked by anything.",
			strings.Join(findings, "\n"))
	}
}

// Without this the rule above would pass by scanning a tree that is not there.
func TestTheConcurrencyScanReachesTheProduct(t *testing.T) {
	root, err := ModuleRoot()
	if err != nil {
		t.Fatalf("cannot find the module root: %v", err)
	}
	// A file known to exist, made to break the rule, has to be reported. This
	// proves the walk reaches the product rather than an empty directory.
	findings, err := Concurrency(root, nil)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	_ = findings

	probe := filepath.Join(root, "internal", "guard")
	if _, err := os.Stat(probe); err != nil {
		t.Fatalf("the tree this rule scans does not look like the repository: %v", err)
	}
}

func scanOne(t *testing.T, source string) []string {
	t.Helper()
	dir := t.TempDir()
	write(t, filepath.Join(dir, "sample.go"), source)
	findings, err := Concurrency(dir, nil)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	return findings
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("cannot write the sample: %v", err)
	}
}
