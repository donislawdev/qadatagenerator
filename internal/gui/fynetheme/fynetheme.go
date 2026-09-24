// Package fynetheme dresses the toolkit in the ladders and tokens this project
// measured.
//
// # Why this is a separate package from the values
//
// The values compile without the graphical toolkit, which is what let the
// bridge to docs/17 close before the toolkit was ever added. Putting the
// adapter beside the values would have undone that: every check of a colour
// would then need the C toolchain to build. So the values stay where anything
// can read them, and this package is the only place that knows a toolkit
// exists at all.
//
// # What is in it
//
// Visual, the one theme the window wears, and Current, the one place naming
// it. Until DEC-253 a second theme stood here, Dark, which wore the palette of
// the first window. The window stopped using it at DEC-232 and it went once its
// own tests were the only thing still reading it. The questions those tests
// asked moved to Visual rather than going with it.
package fynetheme
