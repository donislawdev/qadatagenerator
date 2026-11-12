package values

import (
	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/us"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
)

// Where a city and a region come from, and how a region follows its city.
//
// A file of its own since 2026-11-12 (DEC-278), when en-US began drawing its
// places from the postal register this build already carried and a region
// began reading the city of its row. The registry names these functions and
// says nothing else about places.

// placesFromRegister reports whether the places of a locale come from its postal
// register rather than from word lists.
//
// en-US alone today. Its register says which state every place lies in, which
// is what lets a region follow its city. The word lists of pl-PL say nothing of
// the kind, so there a city and a voivodeship are drawn apart.
func placesFromRegister(locale string) bool { return locale == us.PlacesLocale }

// cities names the places a city is drawn from: one per postal code where the
// locale has a register, so that a place with more codes is drawn more often
// (DEC-278), and the word list otherwise.
func (s Source) cities(generate.Request) (listing, error) {
	if placesFromRegister(s.locale) {
		return listing{entries: us.PlaceNames(), list: "cities", key: "cities"}, nil
	}
	return s.entriesOf("cities")
}

// regions names the regions a region is drawn from, and in a locale with a
// register only the ones the bound city lies in, weighted by its codes there.
//
// # A city the register does not hold
//
// It draws from every region, the way a first name with an unreadable gender
// draws from both lists. Nothing in a recipe reaches that today: a city binds
// only through the city generator, which draws from the same register, and a
// list under values plays no role.
func (s Source) regions(req generate.Request) (listing, error) {
	if !placesFromRegister(s.locale) {
		return s.entriesOf("regions")
	}
	if city, bound := req.Agrees.Value(provider.RoleCity); bound {
		if within := us.RegionsOf(city); len(within) > 0 {
			// Keyed by the city, because a unique column of regions that
			// follows a city walks the regions of that city alone, and a list
			// under a limit on length is remembered by its key.
			return listing{entries: within, list: "regions", key: "regions of " + city}, nil
		}
	}
	return listing{entries: us.Regions(), list: "regions", key: "regions"}, nil
}

// placesOr answers whether a place can be drawn in a locale: always where its
// postal register stands, and where the word list is carried otherwise.
func placesOr(list string) func(string) (bool, string) {
	carried := fromLists(list)
	return func(locale string) (bool, string) {
		if placesFromRegister(locale) {
			return true, ""
		}
		return carried(locale)
	}
}

// whereRegionsFollowCities is the city, in the locales whose register says which
// region a place lies in.
func whereRegionsFollowCities(locale string) []string {
	if placesFromRegister(locale) {
		return []string{provider.RoleCity}
	}
	return nil
}

// wherePostalCodesFollowPlaces is the city and the region, in the locales whose
// register says which place each code belongs to (DEC-282). How the code
// narrows to them, and what it does with a pair the register does not hold, is
// written at us.Postal.Generate.
func wherePostalCodesFollowPlaces(locale string) []string {
	if placesFromRegister(locale) {
		return []string{provider.RoleCity, provider.RoleRegion}
	}
	return nil
}
