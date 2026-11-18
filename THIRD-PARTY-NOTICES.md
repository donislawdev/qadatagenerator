# Third party notices

`qadg`, the command line of QA Data Generator, is Copyright (C) 2026 DonislawDev
and is licensed under the GNU General Public License, version 3 only. The full
text is in [LICENSE](LICENSE). The specification and the corruption catalog
entries built into `qadg` are licensed under Creative Commons Attribution 4.0
International, in [LICENSE-spec](LICENSE-spec).

Every `qadg` binary also contains work that other people made: the Go runtime
and standard library, two Go modules, and the lists of names and places that
generated values are drawn from. Their licences ask that their notices travel
with every copy, a binary one included, so this file is part of every release
archive that carries `qadg`, and it is compiled into `qadg` itself, which prints
it at the end of `qadg --license`. Each of these licences is compatible with
GPL-3.0, and the list of places is in the public domain.

The same modules are in all five builds: Windows and Linux on amd64 and arm64,
and macOS on arm64. One notice below belongs to a file of the Go runtime that
only the amd64 builds link, and it says so.

What this file lists is checked rather than asserted. A guard in the source asks
the build which packages it links, for each of the five, and fails when a module
is missing here, when a version here is not the one linked, when a licence file
beside linked code or beside the lists of names is not reproduced in full under
its own heading, or when a line of linked source that states a copyright is not
quoted under it.

Each section opens with the version every build carries and an SPDX licence
expression that names every text quoted in that section, in the order they
appear. Two texts in the standard library match no entry on the SPDX License
List, and they are named `LicenseRef-Sun-fdlibm-2004` and `LicenseRef-Cephes`
here.

`go.mod` names more modules than these. The others are linked only into the
window, `qadg-gui`, whose notices are in
[THIRD-PARTY-NOTICES-GUI.md](THIRD-PARTY-NOTICES-GUI.md), or are named by a
dependency's own `go.mod` for its tests and tools. None of them is in `qadg`.

---

## Go standard library and runtime

Version `go1.27.4`. BSD-3-Clause AND SunPro AND LicenseRef-Sun-fdlibm-2004 AND
LicenseRef-Cephes AND MIT. Every Go program links the runtime and the parts of
the standard library it uses, so this applies to every build. The first
identifier is the Go project's own licence, and the rest are the notices inside
the standard library quoted further down. The text below is the `LICENSE` file
at the root of that Go distribution, and the copies of `golang.org/x/crypto`,
`golang.org/x/net`, `golang.org/x/sys` and `golang.org/x/text` that the
standard library carries inside itself are under the same text.

```
Copyright 2009 The Go Authors.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google LLC nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

The distribution also carries a `PATENTS` file. It grants patent rights to
anybody who uses this implementation of Go and attaches no condition to passing
it on, so it is not reproduced here.

Source: <https://go.googlesource.com/go>

### Notices inside the standard library

Some files of the standard library are derived from code published under a
notice of its own, and the Go project keeps that notice in the source. These
are the ones in the packages `qadg` links, as the source states them.

#### Sun's fdlibm, by way of FreeBSD, in package `math`

Several functions in `math` are derived from FreeBSD's `msun`, which came from
Sun's fdlibm. The notice appears in two forms. The first is `SunPro` on the
SPDX License List. The second lacks the line naming SunPro, matches no entry
there, and is `LicenseRef-Sun-fdlibm-2004` here.

```
Copyright (C) 1993 by Sun Microsystems, Inc. All rights reserved.

Developed at SunPro, a Sun Microsystems, Inc. business.
Permission to use, copy, modify, and distribute this
software is freely granted, provided that this notice
is preserved.
```

```
Copyright (C) 2004 by Sun Microsystems, Inc. All rights reserved.

Permission to use, copy, modify, and distribute this
software is freely granted, provided that this notice
is preserved.
```

#### The Cephes Math Library, in package `math`

No entry on the SPDX License List matches this text, so it is
`LicenseRef-Cephes` here.

```
Cephes Math Library Release 2.8:  June, 2000
Copyright 1984, 1987, 1989, 1992, 2000 by Stephen L. Moshier

The readme file at http://netlib.sandia.gov/cephes/ says:
   Some software in this archive may be from the book _Methods and
Programs for Mathematical Functions_ (Prentice-Hall or Simon & Schuster
International, 1989) or from the Cephes Mathematical Library, a
commercial product. In either event, it is copyrighted by the author.
What you see here may be used freely but it comes with no support or
guarantee.
```

#### Inferno, in the amd64 runtime

`runtime/memmove_amd64.s` is derived from Inferno's `libkern/memmove-386.s`, and
only the amd64 builds link it, which are the builds for Windows and Linux on
amd64.

```
Derived from Inferno's libkern/memmove-386.s (adapted for amd64)
https://bitbucket.org/inferno-os/inferno-os/src/master/libkern/memmove-386.s

        Copyright © 1994-1999 Lucent Technologies Inc. All rights reserved.
        Revisions Copyright © 2000-2007 Vita Nuova Holdings Limited (www.vitanuova.com).  All rights reserved.
        Portions Copyright 2009 The Go Authors. All rights reserved.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.  IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
```

---

## github.com/goccy/go-yaml

Version `v1.19.2`. MIT. It reads recipes and the
corruption catalog, which are written in YAML.

```
MIT License

Copyright (c) 2019 Masaaki Goshima

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

Source: <https://github.com/goccy/go-yaml>

---

## golang.org/x/text

Version `v0.43.0`. BSD-3-Clause. It puts text into a
Unicode normalization form, for the damage that writes text in a different form
than it was read in. The module also carries a `PATENTS` file, the same grant
as Go's, so it is not reproduced here.

```
Copyright 2009 The Go Authors.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

   * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
   * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
   * Neither the name of Google LLC nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```

Source: <https://cs.opensource.google/go/x/text>

---

## joke2k/faker

Version `v40.36.0`. MIT. The lists of first names and last names for `en-US`
and `pl-PL`, and the lists of Polish cities and voivodeships, were read from
Faker, the fake data library for Python, at this release. Each list names the
file it was read from in its own first lines. The text below is the
`LICENSE.txt` of Faker at that release.

```
Copyright (c) 2012 Daniele Faraglia

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
```

Source: <https://github.com/joke2k/faker>

---

## U.S. Census Bureau

Version `2020 vintage`. Public domain. The list of ZIP Code Tabulation Areas and
the places they lie in, for `en-US`, was built from the 2020 ZCTA to Place
relationship file and the 2020 Places gazetteer of the United States Census
Bureau. A work of the United States Government has no copyright, under 17
U.S.C. 105, so there is no licence to reproduce. The source is named here so
that nobody has to wonder where the list came from.

Source: <https://www.census.gov/geographies/reference-files/time-series/geo/relationship-files.html>
