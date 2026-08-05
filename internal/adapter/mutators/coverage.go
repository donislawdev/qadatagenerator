package mutators

import (
	"sort"

	"github.com/donislawdev/qadatagenerator/internal/domain/catalog"
)

// Coverage answers what this build can do about a layer, from the two halves
// that know: the catalog, which describes damage, and the registry above, which
// implements it.
//
// It exists so that validate can answer what corrupt would. The two halves live
// in different packages and neither can answer alone, and a use case that could
// see only one of them would tell somebody their recipe is fine while the verb
// that runs it does nothing. That is the defect the generator registry caused
// before it became a port, found by running the binary with every test green.
type Coverage struct {
	// Described is the catalog this build ships.
	Described catalog.Catalog
}

// Layers returns every layer the catalog carries damage in, sorted.
//
// A layer with no entries is not in the list. From the point of view of somebody
// writing a policy, naming it and naming one that does not exist come to the
// same thing: no damage happens either way, and the list beside the refusal
// shows what will.
func (c Coverage) Layers() []string {
	seen := map[string]bool{}
	for _, entry := range c.Described.Entries() {
		seen[string(entry.Layer)] = true
	}
	layers := make([]string, 0, len(seen))
	for layer := range seen {
		layers = append(layers, layer)
	}
	sort.Strings(layers)
	return layers
}

// Unimplemented is how much of a layer's damage this build describes and cannot
// yet do.
//
// It is a count rather than a refusal on purpose. The catalog grows faster than
// the engine and that is by design, so a layer waiting for its implementations
// is the ordinary state of this project rather than a fault in somebody's
// recipe. What it must not be is a surprise, which is why validate says it.
func (c Coverage) Unimplemented(layer string) int {
	implemented := make(map[catalog.ID]bool, len(registry))
	for id := range registry {
		implemented[id] = true
	}
	waiting := 0
	for _, entry := range c.Described.InLayer(catalog.Layer(layer)) {
		if !implemented[entry.ID] {
			waiting++
		}
	}
	return waiting
}
