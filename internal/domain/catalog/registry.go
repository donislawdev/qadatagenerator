package catalog

// The closure between the two halves of a mutator, as functions over the
// identifiers rather than over a list kept here.
//
// # Where the list of implemented identifiers lives, and why it moved
//
// It was here for a day. While there were no implementations, an identifier was
// the only certain thing about one, and a list in the domain was the honest
// shape. The implementations now exist in an adapter, so a list here would be a
// second copy of that registry's keys, and two lists part company. It moved,
// and nothing in this file had to change: these take the identifiers as an
// argument, which is what made them testable against a build with nothing
// implemented in the first place.

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
