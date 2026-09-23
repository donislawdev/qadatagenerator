package license

// Source is one project the reference data was read from, with every list read
// from it.
//
// # Why the register keeps a row per list and this groups them
//
// The register keeps a row per list because that is what a gate can hold to
// the files it embeds. A person reading the terms wants the project named once
// with its lists under it, and so does every surface that prints them.
type Source struct {
	Project string
	Version string
	License string
	Source  string
	// Checked is the day the terms of this project at this version were read.
	// Every row of one group carries the same day, which a test holds, so the
	// day of the first row is the day of the group rather than one of several.
	Checked string
	// Lists are the lists read from it, in the order of the register.
	Lists []string
}

// BySource groups the reference data by the project and the version it was read
// from, keeping the order of the register.
//
// # Why this lives beside the register rather than in a surface
//
// It was written in the command line first, and the About screen of the window
// needed the same grouping on 2026-09-23. Two copies of one rule drift apart the
// first time either changes, and a project named twice on one surface and once
// on the other is two answers to one question about licenses.
func BySource(data []DataSource) []Source {
	var grouped []Source
	at := map[string]int{}
	for _, row := range data {
		key := row.Project + " " + row.Version
		if index, seen := at[key]; seen {
			grouped[index].Lists = append(grouped[index].Lists, row.Corpus)
			continue
		}
		at[key] = len(grouped)
		grouped = append(grouped, Source{
			Project: row.Project,
			Version: row.Version,
			License: row.License,
			Source:  row.Source,
			Checked: row.Checked,
			Lists:   []string{row.Corpus},
		})
	}
	return grouped
}
