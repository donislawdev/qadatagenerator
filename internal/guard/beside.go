package guard

import (
	"debug/pe"
	"fmt"
	"go/build"
	"path/filepath"
	"sort"
	"strings"
)

// # Why the files beside Go are read
//
// The guards read Go. The go command also compiles assembly, C, C++,
// Objective-C and Fortran files it finds in a package, and links any .syso
// object as it lies, with no import anywhere to show for it. Assembly can make a
// system call in a package that never names syscall. A package holding any of
// these files is also compiled without the check that every function has a body
// (the -complete flag in cmd/go/internal/work/gc.go), so a function declared in
// Go can be written in one of them instead. So the only such files our
// directories may hold are .syso objects carrying nothing but Windows
// resources, which is what the icon and the version of each binary are.
//
// Which files the go command compiles is asked of go/build rather than listed
// here, so a kind it learns to compile is refused on the day the toolchain
// learns it. cmd/go keeps its own copy of that list in
// cmd/go/internal/modindex, the same as the one in go/build in Go 1.27.4.

// compiledByGo answers which files the go command compiles. UseAllFiles leaves
// build constraints out, so a file meant for another system is seen too.
var compiledByGo = func() build.Context {
	context := build.Default
	context.UseAllFiles = true
	return context
}()

// CompiledBesideGo lists every file of ours, in the directories the guards
// read, that the go command compiles or links into a package beside its Go
// source.
func CompiledBesideGo(root string) ([]string, error) {
	var files []string
	err := eachProductEntry(root, ".", func(path, name string) error {
		if strings.HasSuffix(name, ".go") {
			return nil
		}
		compiled, err := compiledByGo.MatchFile(filepath.Dir(path), filepath.Base(path))
		if err != nil {
			return fmt.Errorf("cannot tell whether the go command compiles %s: %w", name, err)
		}
		if compiled {
			files = append(files, name)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

// BesideGo reports every file of files, as CompiledBesideGo lists them, that the
// guards cannot vouch for: anything but a .syso, and a .syso carrying more than
// Windows resources.
func BesideGo(root string, files []string) []string {
	var findings []string
	for _, name := range files {
		if filepath.Ext(name) != ".syso" {
			findings = append(findings, name+" is compiled into its package beside the Go files, "+
				"and the guards read only Go, so what it does is checked by nothing")
			continue
		}
		findings = append(findings, resourcesOnly(filepath.Join(root, filepath.FromSlash(name)), name)...)
	}
	return findings
}

// resourcesOnly reports what a .syso carries beyond Windows resources.
//
// The linker reads a .syso as a COFF object through debug/pe, as this does
// (cmd/link/internal/loadpe), and gathers the sections named .rsrc, or .rsrc$
// with a suffix, into the resource table of the executable. A section that is
// neither code nor executable cannot be run, and a symbol the object names
// without defining it would be resolved against code elsewhere, so those are
// the things asked.
func resourcesOnly(path, name string) []string {
	object, err := pe.Open(path)
	if err != nil {
		return []string{fmt.Sprintf("%s cannot be read as an object file, so what it links into the binary "+
			"is unknown: %v", name, err)}
	}
	defer func() { _ = object.Close() }()
	if len(object.Sections) == 0 {
		return []string{name + " carries no section, and a .syso here is there to carry Windows resources"}
	}
	var findings []string
	for _, section := range object.Sections {
		if section.Name != ".rsrc" && !strings.HasPrefix(section.Name, ".rsrc$") {
			findings = append(findings, fmt.Sprintf("%s carries the section %q, and a .syso may carry only "+
				"Windows resources (.rsrc)", name, section.Name))
		}
		if section.Characteristics&(pe.IMAGE_SCN_CNT_CODE|pe.IMAGE_SCN_MEM_EXECUTE) != 0 {
			findings = append(findings, fmt.Sprintf("%s marks its section %q as code", name, section.Name))
		}
	}
	for _, symbol := range object.Symbols {
		if symbol.SectionNumber < 1 || int(symbol.SectionNumber) > len(object.Sections) {
			findings = append(findings, fmt.Sprintf("%s names the symbol %q outside its own sections, which "+
				"the linker would resolve against code elsewhere", name, symbol.Name))
		}
	}
	return findings
}
