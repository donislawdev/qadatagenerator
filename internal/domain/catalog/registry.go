package catalog

// implemented lists every mutator whose behaviour exists in this build.
//
// # Why it is empty, and why that is written down rather than left to be found
//
// The corruption engine does not exist yet. Until it does there is no behaviour
// to register, so this list is empty and the catalog is metadata with nothing
// behind it. That is the honest state of this build and the surfaces say so
// rather than implying a mutator can be run.
//
// # Why the list is identifiers and not implementations
//
// A mutator interface written before the engine would be a guess about what the
// engine needs. This project has already paid for one of those: an interface
// for reserved ranges was designed a day ahead of its first use, and the first
// use showed it had only a reading side, so the mutator that needed it could
// neither build a value nor tell whether it applied. The behaviour half of this
// registry lands with the engine that calls it, and until then this holds the
// one thing that is certain, which is which identifiers are covered.
var implemented = []ID{}

// Implemented returns the identifiers that have behaviour in this build,
// sorted.
func Implemented() []ID {
	ids := make([]ID, len(implemented))
	copy(ids, implemented)
	sortIDs(ids)
	return ids
}

// Unregistered returns the catalog entries that no implementation covers,
// sorted.
//
// Today that is every entry, and it shrinks as the engine grows. It is reported
// rather than treated as a failure, because a catalog entry describes damage
// that is worth doing whether or not this build can do it yet, and the catalog
// is meant to grow faster than the engine.
func Unregistered(c Catalog, ids []ID) []ID {
	covered := make(map[ID]bool, len(ids))
	for _, id := range ids {
		covered[id] = true
	}
	var missing []ID
	for _, id := range c.ordered {
		if !covered[id] {
			missing = append(missing, id)
		}
	}
	sortIDs(missing)
	return missing
}

// Undescribed returns the implementations that no catalog entry describes,
// sorted.
//
// This is the half of the closure that is a defect rather than a state of
// progress. A mutator that runs without an entry has no declared expectation,
// so a manifest naming it tells a test nothing about what should happen next,
// and rule four of the catalog says such a mutator does not exist. It also
// means the published specification and the behaviour have parted company,
// which is the drift the whole arrangement is meant to make impossible.
//
// # The other direction is not closed yet, and pretending otherwise would cost
//
// The architecture asks for equality between the two sets. Equality cannot hold
// while the engine is missing, and a test that asserted it with the whole
// catalog listed as an exception would pass against a build with no mechanism
// at all. Three guards written that way were found standing green during the
// last piece of work, so the direction that cannot fail today is deliberately
// left out and lands with the first mutator.
func Undescribed(c Catalog, ids []ID) []ID {
	var unknown []ID
	for _, id := range ids {
		if _, ok := c.Lookup(id); !ok {
			unknown = append(unknown, id)
		}
	}
	sortIDs(unknown)
	return unknown
}
