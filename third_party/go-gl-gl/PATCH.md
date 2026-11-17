# github.com/go-gl/gl, carried with one change

This directory is the module `github.com/go-gl/gl` at version
`v0.0.0-20260331235117-4566fea9a276`, whose module sum is
`h1:IO5P06Pcj9K04d+l4nrf3c2U56+dAotIFG6u4P1wAHI=`. That is the version
`fyne.io/fyne/v2` v2.8.2 requires. The `go.mod` at the root of this repository
points the module path here with a `replace` directive, so every build of the
window takes the binding from this directory. The command line binary does not
link it at all.

Only what the toolkit imports is kept: the module's `go.mod` and `LICENSE`,
the package `v2.1/gl`, which every desktop build of the window links, and the
package `v3.1/gles2`, which the toolkit imports on arm64 and which `go mod tidy`
looks for on every platform. `v3.1/gles2` is unchanged.

## What changed

In `v2.1/gl` two files lose the line that links `opengl32.dll` on Windows:

```
#cgo !gles2,windows       LDFLAGS: -lopengl32
```

`procaddr.go` carries it in its cgo block and the generated `package.go`
carries it as a comment line of its own. In `procaddr.go` the Windows branch of
`GlowGetProcAddress` then finds `wglGetProcAddress` with `GetProcAddress` after
`LoadLibraryA("opengl32.dll")` instead of calling it as a linked function. The
order of the lookups is the published one. A null from the loader is answered
with a null rather than passed to `GetProcAddress`.

`PATCH.diff` beside this file is the same change in the unified format, the
one `patch -p1` applies inside the published module.

Nothing else differs from the published module. `TestTheOpenGLBindingIsThePublishedOnePlusOneChange`
in `internal/guard` holds the copy to that: it reads the published version
from the module cache, checks the sum above against its bytes, applies
`PATCH.diff` to it and compares every file here with the result, byte for
byte.

## Why

Linked, the binding makes the window name `opengl32.dll` in its import table,
so Windows maps the system's copy before any code of the program runs, and
then hands that copy to every later request for the same name. On a computer
whose graphics driver gives no OpenGL window, the window starts again and
loads the software renderer from the `opengl` folder beside it before the
toolkit asks for anything. With the import in place that renderer could never
be the `opengl32.dll` the toolkit draws with.

Looked up instead, the first request for `opengl32.dll` is the one the
windowing library makes by name when it creates its drawing context, and by
then the window has had its chance to load the renderer. On a computer with a
working driver that request finds the system's copy, as before.

The lookup by name follows the same search order the windowing library already
uses for the same file, so it opens nothing that was not open before.

## Licence

MIT, as published. `LICENSE` beside this file is the module's own, unchanged.
