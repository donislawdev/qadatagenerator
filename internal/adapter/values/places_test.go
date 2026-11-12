package values_test

import (
	"slices"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/provider/us"
	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// TestACityAndARegionInEnglishComeFromThePostalRegister holds DEC-278 for one
// column at a time: in en-US a city is a place of the Census register this
// build carries, and a region is the code of a state that register names.
//
// The weighting is held as well, on a fixed seed so that it is the same answer
// on every run rather than a likelihood. The register gives Texas 1734 of its
// 28 658 codes and Rhode Island 54, so a draw weighted by codes gives Texas
// about three times its flat share of a fifty second and Rhode Island well
// under it. A flat draw over the codes, which is what DEC-278 refused, gives
// each about the flat share.
func TestACityAndARegionInEnglishComeFromThePostalRegister(t *testing.T) {
	source := values.New("en-US", "places")
	places, regions := setOf(us.PlaceNames()), setOf(us.Regions())
	const rows = 5200
	counted := map[string]int{}
	for row := int64(0); row < rows; row++ {
		city, cityErr := source.Produce(ask(values.City, "city", row))
		region, regionErr := source.Produce(ask(values.Region, "region", row))
		if cityErr != nil || regionErr != nil {
			t.Fatalf("en-US refused a place at row %d: %v %v", row, cityErr, regionErr)
		}
		if !places[city] {
			t.Errorf("the city %q is not a place of the postal register", city)
		}
		if !regions[region] || len(region) != 2 {
			t.Errorf("the region %q is not the two letter code of a state of the register", region)
		}
		counted[region]++
	}
	flat := rows / len(regions)
	if counted["TX"] < 2*flat || counted["RI"] > flat {
		t.Errorf("Texas was drawn %d times and Rhode Island %d in %d rows, where a draw weighted "+
			"by codes gives Texas more than %d and Rhode Island fewer than %d",
			counted["TX"], counted["RI"], rows, 2*flat, flat)
	}
}

// TestARegionLiesWhereItsCityIs holds the pairing of DEC-278: a region bound to
// a city is a state that city lies in, weighted by its codes there.
//
// Three cities, chosen for what each can show. Chicago lies in one state, so
// its region is always the same. Houston lies in seven, with 107 of its 113
// codes in Texas. Springfield lies in sixteen, so a region that ignored the
// city would still land in one of them now and then, and only many rows tell
// the two apart.
func TestARegionLiesWhereItsCityIs(t *testing.T) {
	source := values.New("en-US", "paired")
	for _, city := range []string{"Chicago", "Houston", "Springfield"} {
		within := setOf(us.RegionsOf(city))
		if len(within) == 0 {
			t.Fatalf("the register holds no place called %s, so this test proves nothing", city)
		}
		for row := int64(0); row < 200; row++ {
			request := ask(values.Region, "region", row)
			request.Agrees = provider.NewAgreement(map[string]string{provider.RoleCity: city})
			region, err := source.Produce(request)
			if err != nil {
				t.Fatalf("a region bound to %s was refused: %v", city, err)
			}
			if !within[region] {
				t.Errorf("a region bound to %s is %s, where %s lies only in %v", city, region, city,
					sortedKeys(within))
			}
		}
	}
}

// TestARecipeInEnglishPairsTheRegionWithTheCity holds the half a person sees:
// a recipe with a city and a region is told that the region follows the city,
// although no rule says so, and every row keeps it. A Polish recipe is told
// nothing and keeps nothing, because its lists do not say which city lies in
// which voivodeship.
func TestARecipeInEnglishPairsTheRegionWithTheCity(t *testing.T) {
	fields := []recipe.Field{field("city", "city"), field("region", "region")}
	english := person(300, fields)
	english.Locale = "en-US"
	plan, rows := produced(t, english)
	for i, row := range rows {
		if !slices.Contains(us.RegionsOf(row["city"]), row["region"]) {
			t.Errorf("row %d puts %s in %s, where it does not lie", i, row["city"], row["region"])
		}
	}
	if len(plan.Implied) != 1 || plan.Implied[0].Rule.Field != "region" ||
		!slices.Equal(plan.Implied[0].Rule.With, []string{"city"}) {
		t.Errorf("the plan says %+v about what follows what, and should say region follows city",
			plan.Implied)
	}

	polish, _ := produced(t, person(10, fields))
	if len(polish.Implied) != 0 {
		t.Errorf("a Polish recipe is told %+v, where nothing says which city lies in which "+
			"voivodeship", polish.Implied)
	}
}

// TestARecipeInEnglishPairsThePostalCodeWithItsPlace holds DEC-282 for the three
// shapes a recipe can give a postal code a place in, and the chain of DEC-283
// with it: with a city and a region, the region follows the city and the code
// follows both, so the code is drawn after a value that is itself bound.
//
// The postal code stands first among the fields on purpose. Its implied rule
// then comes before the rule of the region it reads, and only the order the
// run puts the rules in makes the code agree with the region the row ends with.
func TestARecipeInEnglishPairsThePostalCodeWithItsPlace(t *testing.T) {
	inEnglish := func(fields ...recipe.Field) recipe.Recipe {
		rec := person(300, fields)
		rec.Locale = "en-US"
		return rec
	}
	placeOf := func(code string) provider.Place {
		place, found := us.Postal{}.PlaceFor(code)
		if !found {
			t.Fatalf("the code %s is not in the register it was drawn from", code)
		}
		return place
	}

	plan, rows := produced(t, inEnglish(
		field("zip", "postal_code"), field("city", "city"), field("region", "region")))
	for i, row := range rows {
		if got := placeOf(row["zip"]); got != (provider.Place{Name: row["city"], Region: row["region"]}) {
			t.Errorf("row %d says %s, %s and its code %s belongs to %s, %s",
				i, row["city"], row["region"], row["zip"], got.Name, got.Region)
		}
	}
	follows := map[string][]string{}
	for _, implied := range plan.Implied {
		follows[implied.Rule.Field] = implied.Rule.With
	}
	if !slices.Equal(follows["zip"], []string{"city", "region"}) || !slices.Equal(follows["region"], []string{"city"}) {
		t.Errorf("the plan says %v about what follows what, and should say the code follows the "+
			"city and the region, and the region the city", follows)
	}

	_, rows = produced(t, inEnglish(field("zip", "postal_code"), field("city", "city")))
	for i, row := range rows {
		if got := placeOf(row["zip"]); got.Name != row["city"] {
			t.Errorf("row %d says %s and its code %s belongs to %s", i, row["city"], row["zip"], got.Name)
		}
	}

	_, rows = produced(t, inEnglish(field("zip", "postal_code"), field("region", "region")))
	for i, row := range rows {
		if got := placeOf(row["zip"]); got.Region != row["region"] {
			t.Errorf("row %d says %s and its code %s lies in %s", i, row["region"], row["zip"], got.Region)
		}
	}
}

func setOf(entries []string) map[string]bool {
	set := make(map[string]bool, len(entries))
	for _, entry := range entries {
		set[entry] = true
	}
	return set
}

func sortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
