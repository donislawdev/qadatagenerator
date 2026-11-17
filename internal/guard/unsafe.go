package guard

// UnsafeAllowed names the files that may import unsafe, together with the names
// each of them may take from it.
//
// # Why unsafe is gated like syscall
//
// unsafe lets a file read and write any memory of the process and make a
// function value of any address, so it can call a function of the system that
// no import names. It is also what the compiler asks a file to import before it
// honours go:linkname. So the network rule refuses it in our files except here,
// and an allowance is a file and the names it may use, read as a pair, exactly
// as in SyscallAllowed.
var UnsafeAllowed = map[string][]string{
	// Handing text and a buffer to the Windows calls in CgoAllowed, which take
	// C pointers (DEC-315, DEC-316).
	"cmd/qadg-gui/console_windows.go": {"Pointer", "Sizeof"},
}

// unsafePackage is the import being controlled.
const unsafePackage = "unsafe"

// unsafeGate holds the product to UnsafeAllowed.
var unsafeGate = namedGate{importPath: unsafePackage, list: "UnsafeAllowed"}

// packageMayUseUnsafe reports whether a package holds a file allowed to import
// unsafe.
func packageMayUseUnsafe(pkg string) bool {
	return holdsAnAllowedFile(UnsafeAllowed, pkg)
}

// UnsafeUse reports every place that imports unsafe outside what the list
// allows, in both directions.
func UnsafeUse(root string, allowed map[string][]string) ([]string, error) {
	return unsafeGate.use(root, allowed)
}
