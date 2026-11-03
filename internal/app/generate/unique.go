package generate

import (
	"fmt"
	"math"

	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// uniqueSet remembers what each unique field has already produced.
//
// A claim stays pending until the row it belongs to is kept, because a row
// built again for its key has to give back what it took the first time. Kept
// claims are never given back.
type uniqueSet struct {
	seen    map[string]map[string]bool
	pending []claimed
	walkers map[string]*walker
}

type claimed struct{ field, value string }

func newUniqueSet(entity recipe.Entity) *uniqueSet {
	set := &uniqueSet{seen: make(map[string]map[string]bool), walkers: make(map[string]*walker)}
	for _, field := range entity.Fields {
		if field.Unique {
			set.seen[field.Name] = make(map[string]bool)
		}
	}
	return set
}

// claim records a value and reports whether it was free.
func (s *uniqueSet) claim(field, value string) bool {
	seen, tracked := s.seen[field]
	if !tracked {
		return true
	}
	if seen[value] {
		return false
	}
	seen[value] = true
	s.pending = append(s.pending, claimed{field, value})
	return true
}

// free reports whether no row has taken a value yet.
func (s *uniqueSet) free(field, value string) bool {
	return !s.seen[field][value]
}

// keep makes what the row claimed permanent.
func (s *uniqueSet) keep() {
	s.pending = s.pending[:0]
}

// release gives back what the row claimed.
func (s *uniqueSet) release() {
	for _, taken := range s.pending {
		delete(s.seen[taken.field], taken.value)
	}
	s.pending = s.pending[:0]
}

// walkerFor returns the walker of one domain of one field.
//
// The size is compared as well as the name, so that two domains a source
// happened to name alike cannot share what was learned about either of them.
func (s *uniqueSet) walkerFor(field string, domain Domain) *walker {
	name := field + "\x00" + domain.Key
	if known, found := s.walkers[name]; found && known.size == domain.Size {
		return known
	}
	made := newWalker(domain.Size)
	s.walkers[name] = made
	return made
}

// askedDomain is one answer of the source about a domain, kept with whether
// the source knew it.
type askedDomain struct {
	domain Domain
	known  bool
}

// domainOf returns the domain a unique field walks in a row, asking the source
// once for a field whose domain is the same in every row.
//
// # Why once, and for which fields
//
// 🔴 Until 2026-11-03 every row that walked asked again, and the source counts
// the entries of a list each time it is asked, so a unique list of 28000
// entries under values took 2.8 s where the same table without unique took
// 0.077 s (B-348). A request differs from row to row only in its address and in
// what the row agrees with, and a domain does not depend on the address, which
// the plan and the walk of a key rely on as well. So the answer is kept for a
// field no rule binds, and for a field drawing from a list under values, which
// draws from the list alone whatever it agrees with. Any other field a rule
// binds asks in every row, because a first name of a woman and one of a man
// are two domains, and keeping an answer for each agreement would hold a list
// for every value a row can agree with.
func (t *tableRun) domainOf(field recipe.Field, request Request) (Domain, bool) {
	if t.bound[field.Name] && len(field.Values) == 0 {
		return t.values.DomainOf(request)
	}
	if kept, found := t.domains[field.Name]; found {
		return kept.domain, kept.known
	}
	domain, known := t.values.DomainOf(request)
	t.domains[field.Name] = askedDomain{domain: domain, known: known}
	return domain, known
}

// keyParts returns the parts of the primary key the run holds, and whether it
// holds the key at all.
//
// A part this build leaves empty in every row is the same in every row, so it
// says nothing about which keys repeat and is left out. The key is not held
// when nothing is left, or when a part is unique, because a key with a unique
// part cannot repeat and holding it would cost memory for nothing.
func keyParts(entity recipe.Entity, values Values) ([]recipe.Field, bool) {
	var parts []recipe.Field
	for _, name := range entity.PrimaryKey {
		field, known := entity.Field(name)
		if !known || leftBlank(field, values) {
			continue
		}
		if field.Unique {
			return nil, false
		}
		parts = append(parts, field)
	}
	return parts, len(parts) > 0
}

// keySet remembers the keys a table has already used.
//
// # Why the key is held whole
//
// 🔴 Since 2026-10-26. A primary key was a name the recipe gave and nothing
// behind it, so a key of pesel produced two rows with the same PESEL whenever
// two draws met, and a dataset loaded behind the key was refused by the
// database for something the recipe itself declared. A unique flag on every
// part would not do: a key of an order and a line number repeats each of them
// on purpose and only the pair has to differ.
type keySet struct {
	parts []recipe.Field
	names map[string]bool
	taken map[string]bool
	// The domain of the whole key, worked out the first time a walk needs it.
	counted bool
	domains []Domain
	size    int64
	walker  *walker
}

func newKeySet(parts []recipe.Field) *keySet {
	names := make(map[string]bool, len(parts))
	for _, part := range parts {
		names[part.Name] = true
	}
	return &keySet{parts: parts, names: names, taken: make(map[string]bool)}
}

// holds reports whether a field is a part of a key the run holds. It answers
// on a table whose key is not held, too, because the answer is then no.
func (k *keySet) holds(field string) bool {
	return k != nil && k.names[field]
}

// claim records the key of a row and reports whether it was free.
func (k *keySet) claim(built map[string]string) bool {
	values := make([]string, len(k.parts))
	for index, part := range k.parts {
		values[index] = built[part.Name]
	}
	key := keyOf(values)
	if k.taken[key] {
		return false
	}
	k.taken[key] = true
	return true
}

// walk chooses a free key once building again has run out of luck, and
// returns the values its parts take.
func (k *keySet) walk(t *tableRun, row int) (map[string]string, error) {
	where := "entities." + t.entity.Name + ".primary_key"
	if !k.countable(t) {
		return nil, fault.New(fault.Data, fmt.Sprintf(
			"%s: %d attempts in a row built a key an earlier row already has, at row %d of %d. "+
				"This build does not count how many keys the fields can make, so it cannot tell "+
				"whether any are left", where, uniqueTries, row+1, t.rows)).
			DoInstead("lower the count, or add a field to the key")
	}
	start := t.drawnPlace(seed.Coordinate{Entity: t.entity.Name, Row: int64(row)}, "value:key#walk", k.size)
	place, found := k.walker.next(start, func(at int64) bool {
		return !k.taken[keyOf(k.valuesAt(at))]
	})
	if !found {
		return nil, fault.New(fault.Data, fmt.Sprintf(
			"%s: every one of the %d keys the fields can make is already used, at row %d of %d",
			where, k.size, row+1, t.rows)).
			DoInstead("lower the count, or add a field to the key")
	}
	forced := make(map[string]string, len(k.parts))
	for index, value := range k.valuesAt(place) {
		forced[k.parts[index].Name] = value
	}
	return forced, nil
}

// countable works out the domain of the whole key, and reports whether there
// is one to walk. There is when every part is free of any rule, so that its
// domain is the same in every row, and when this build knows every one of them.
func (k *keySet) countable(t *tableRun) bool {
	if k.counted {
		return k.walker != nil
	}
	k.counted = true
	size := int64(1)
	for _, part := range k.parts {
		if t.bound[part.Name] {
			return false
		}
		domain, known := t.values.DomainOf(requestFor(t.entity.Name, part, 0, "value"))
		if !known || domain.Size == 0 || size > math.MaxInt64/domain.Size {
			return false
		}
		size *= domain.Size
		k.domains = append(k.domains, domain)
	}
	k.size = size
	k.walker = newWalker(size)
	return true
}

// valuesAt returns the values of the parts at one place in the domain of the
// whole key, the last part changing fastest.
func (k *keySet) valuesAt(place int64) []string {
	values := make([]string, len(k.domains))
	for index := len(k.domains) - 1; index >= 0; index-- {
		size := k.domains[index].Size
		values[index] = k.domains[index].At(place % size)
		place /= size
	}
	return values
}
