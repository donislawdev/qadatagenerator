package main

import "github.com/donislawdev/qadatagenerator/internal/license"

// licenseListVersion is the release of the SPDX License List every identifier
// below was read on, and the document records it.
const licenseListVersion = "3.29"

// ourLicence is what QA Data Generator is under, from the register the
// command line prints with --license.
const ourLicence = license.Code

// listedLicences are the identifiers from the SPDX License List a document may
// carry. Each was read on 2026-11-18 on the list itself, at release 3.29.0 of
// 2026-09-16 (json/licenses.json at the tag v3.29.0 of
// https://github.com/spdx/license-list-data, then the newest release), and
// none is deprecated there.
//
// Which text is which identifier was matched the same day against the list's
// own templates, every quoted text of both notices files and of the software
// renderer's licence files, not by eye (tools/noty/spdx, DEC-328). Where no
// template matched the whole text, the difference was read word by word: the
// Unlicense of go-text carries a title line, the SIL Open Font License of
// Inter says "and" where the template says "&", the ISC text of the MinGW-w64
// runtime is OpenBSD's older one with "and distribute" and a line of
// sponsorship, its first notice is the Zope Public License 2.1 without the
// title and the sentence that a notice accompanies it, and the GNU Lesser
// General Public License 2.1 is the text gnu.org gives today, with the address
// of the Free Software Foundation replaced by its web page. CC-BY-4.0 and
// OPUBL-1.0 are licences a quoted statement names, with no text of their own in
// the notices. NCSA is the older licence LLVM's own licence file ends with,
// which LLVM says still covers a small part of its code alone.
//
// An identifier the notices name and this list does not is refused rather than
// passed through. A misspelt one makes the whole document invalid, and a valid
// one nobody read is a licence nobody checked.
var listedLicences = map[string]bool{
	"Apache-2.0":        true,
	"Bitstream-Vera":    true,
	"BSD-1-Clause":      true,
	"BSD-2-Clause":      true,
	"BSD-3-Clause":      true,
	"BSL-1.0":           true,
	"CC-BY-4.0":         true,
	"GPL-3.0-only":      true,
	"HPND-sell-variant": true,
	"ISC":               true,
	"LGPL-2.1-or-later": true,
	"MIT":               true,
	"NCSA":              true,
	"OFL-1.1":           true,
	"OPUBL-1.0":         true,
	"SMLNJ":             true,
	"SunPro":            true,
	"Unlicense":         true,
	"Zlib":              true,
	"ZPL-2.1":           true,
}

// listedExceptions are the identifiers from the list of exceptions the same
// release publishes beside the licences (json/exceptions.json at v3.29.0), the
// only words that may follow WITH. Read on 2026-11-18 and not deprecated there.
// LLVM's own exception to Apache-2.0 is in the software renderer that travels
// beside the window on Windows. The two GCC exceptions are the permission to
// link the start files of the GNU C Library into any program, with and without
// the paragraph that says modified versions need not keep it, each matched
// that day against its template.
var listedExceptions = map[string]bool{
	"GCC-exception-2.0":      true,
	"GCC-exception-2.0-note": true,
	"LLVM-exception":         true,
}

// refPrefix is how SPDX names a text that is not on its list.
const refPrefix = "LicenseRef-"

// An unlistedLicence is a text the SPDX License List has no identifier for. A
// document that names one has to carry its words, and the words are the quoted
// text in the notices that opens with the line given here, so they exist
// once, in the notices, and not a second time in Go.
type unlistedLicence struct {
	id    string
	name  string
	opens string
}

// unlisted are those texts. Each matched no template of release 3.29.0 on
// 2026-11-18 (tools/noty/spdx).
//
// The first two are in the standard library. The 2004 form of the fdlibm
// notice is the SunPro text without its "Developed at SunPro" line, which that
// template does not let go of. The next four are in the window: the Arev
// changes to the DejaVu font of the toolkit, the public domain statement of a
// MinGW-w64 header in GLFW, the SGI Free Software License B without a version,
// which the OpenGL reference pages name, and MinGW-w64's own note that the
// Cephes files it carries came without a licence, which is not the Cephes
// notice of the standard library. The last is no licence at all: the law that
// says a work of the United States Government has no copyright, for the list
// of places drawn from the Census Bureau's files.
var unlisted = []unlistedLicence{
	{
		id:    "LicenseRef-Sun-fdlibm-2004",
		name:  "Sun Microsystems fdlibm notice, 2004 form",
		opens: "Copyright (C) 2004 by Sun Microsystems, Inc. All rights reserved.",
	},
	{
		id:    "LicenseRef-Cephes",
		name:  "Cephes Math Library notice",
		opens: "Cephes Math Library Release 2.8:  June, 2000",
	},
	{
		id:    "LicenseRef-Arev",
		name:  "Arev Fonts notice, as DejaVu Sans Mono for Powerline carries it",
		opens: "Fonts are (c) Bitstream (see below). DejaVu changes are in public domain.",
	},
	{
		id:    "LicenseRef-Public-Domain",
		name:  "MinGW-w64 public domain notice",
		opens: "This file has no copyright assigned and is placed in the Public Domain.",
	},
	{
		id:    "LicenseRef-SGI-FreeB",
		name:  "SGI Free Software License B, with no version named",
		opens: "Copyright (c) 2010 Khronos Group.",
	},
	{
		id:    "LicenseRef-MinGW-w64-Cephes",
		name:  "MinGW-w64 note on the Cephes math library",
		opens: "FIXME: Cephes math lib",
	},
	{
		id:    "LicenseRef-US-Government-Work",
		name:  "17 U.S.C. 105(a), no copyright in a work of the United States Government",
		opens: "Copyright protection under this title is not available for any work of the",
	},
}

// unlistedNamed is the entry for one LicenseRef identifier, or nil.
func unlistedNamed(id string) *unlistedLicence {
	for i := range unlisted {
		if unlisted[i].id == id {
			return &unlisted[i]
		}
	}
	return nil
}
