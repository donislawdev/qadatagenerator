package guard

// SyscallAllowed names the files that may write syscall, together with the
// identifiers each of them may take from it.
//
// # Why an exception exists at all
//
// The network guard refuses the syscall package everywhere, and the reason is
// written beside that refusal: it can do whatever the operating system can,
// sockets included, so importing it would route around every other name on that
// list. That is still true. What it costs is that a program cannot name the
// termination signal, because the standard library exposes it nowhere else.
//
// # Why the exception is a name and not a package
//
// A file level exception would let anything from syscall into that file, and a
// year from now nobody would remember that the file is an exception. The list
// below is therefore read as a pair: which file, and which identifiers. Writing
// syscall.Socket in the allowed file is a finding exactly like writing syscall
// anywhere else. C is gated the same way, in CgoAllowed.
//
// # What it does not do
//
// It reads the tree as written. A name reached indirectly, through a variable
// holding a function value, is invisible here, as it is to the concurrency
// guard beside it. Both answer the question a reviewer would ask by reading,
// which is a smaller question than what the program can do.
// # Why the second entry names the same identifier as the first
//
// 🔴 It is the same promise on the other surface, not a second exception. The
// window writes the same temporary files and holds the same markers as the
// command line, so a system taking it down mid run leaves the same directory
// behind. Until 2026-09-01 it left exactly that, because watchSignals lived in
// internal/cli and internal/gui may not import it. The rule the two surfaces
// share is the sixth unbreakable one, and it was broken in one direction: one
// function, two behaviours, depending on which half of the product was asked.
//
// Naming one identifier twice rather than widening either entry is deliberate.
// The allowance is read as a pair of file and name, so two files each allowed
// one name is a smaller opening than one file allowed a package.
var SyscallAllowed = map[string][]string{
	// The termination signal, so that a run stopped by a system rather than by
	// a person still removes what it was in the middle of writing, and then
	// ends by that signal.
	"internal/cli/signals.go": {"SIGTERM"},
	// The same, for a run started from the window.
	"internal/gui/background/background.go": {"SIGTERM"},
	// Loading the software renderer from the opengl folder beside the window,
	// by the absolute path the window works out itself, on Windows when the
	// graphics driver gave no window (B-414, DEC-317). LoadDLL loads one file
	// and reads the system's reason in the same call when it fails, which a
	// call through cgo could not promise.
	"cmd/qadg-gui/software_windows.go": {"LoadDLL"},
}

// syscallPackage is the import being controlled.
const syscallPackage = "syscall"

// syscallGate holds the product to SyscallAllowed.
var syscallGate = namedGate{importPath: syscallPackage, list: "SyscallAllowed"}

// packageMayNameSyscall reports whether a package holds a file allowed to write
// syscall.
func packageMayNameSyscall(pkg string) bool {
	return holdsAnAllowedFile(SyscallAllowed, pkg)
}

// SyscallUse reports every place that writes syscall outside what the list
// allows, in both directions. namedGate.use says what each kind of finding
// means.
func SyscallUse(root string, allowed map[string][]string) ([]string, error) {
	return syscallGate.use(root, allowed)
}
