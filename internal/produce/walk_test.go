package produce_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/app/corrupt"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/domain/seed"
)

// 🔴 On the catalog this build carries, three damaged rows in a row land in
// three layers, from every place the walk can start (DEC-291).
//
// The selector's own test holds this for layers of sizes written into it. This
// one reads the real catalog, so a layer that grows until it crowds the walk
// fails here instead of going unnoticed. One run over the whole walk and two
// rows more reads every window of three consecutive rows.
func TestOnTheRealCatalogThreeRowsTouchThreeLayers(t *testing.T) {
	described, err := catalogyaml.Load()
	if err != nil {
		t.Fatalf("the embedded catalog did not load: %v", err)
	}
	var layers []string
	for _, entry := range described.Entries() {
		if len(layers) == 0 || layers[len(layers)-1] != string(entry.Layer) {
			layers = append(layers, string(entry.Layer))
		}
	}
	if len(layers) < 3 {
		t.Fatalf("the catalog holds %d layers, too few for this to say anything", len(layers))
	}
	view := recipe.Recipe{Seed: "s", Locale: "pl-PL", Entities: []recipe.Entity{{
		Name: "t", Count: 1, HasCount: true, Fields: []recipe.Field{{Name: "text", Type: "string"}},
	}}}
	selector, err := corrupt.NewSelector(recipe.Policy{Layers: layers, Rows: "100%"},
		described, view, seed.New("the real catalog"))
	if err != nil {
		t.Fatalf("the selector refused every layer of the catalog: %v", err)
	}

	rows := described.Len() + 2
	var walked []string
	for row := 0; row < rows; row++ {
		orders := selector.Orders("t", row, rows)
		if len(orders) != 1 {
			t.Fatalf("row %d carries %d orders, want one", row, len(orders))
		}
		walked = append(walked, strings.SplitN(string(orders[0].Mutator), ".", 2)[0])
	}
	if problem := fewerLayersThanRows(walked, 3); problem != "" {
		t.Error(problem)
	}
	// The canary is the shape of the defect: the walk in identifier order.
	if fewerLayersThanRows([]string{"L0", "L0", "L0", "L1"}, 3) == "" {
		t.Fatal("the check does not notice rows that all land in one layer")
	}
}

// fewerLayersThanRows names the first run of consecutive rows that touched
// fewer layers than there are rows in it, and nothing when there is none.
func fewerLayersThanRows(walked []string, width int) string {
	for start := 0; start+width <= len(walked); start++ {
		distinct := map[string]bool{}
		for _, layer := range walked[start : start+width] {
			distinct[layer] = true
		}
		if len(distinct) < width {
			return fmt.Sprintf("rows %d to %d touched %d layers: %v",
				start+1, start+width, len(distinct), walked[start:start+width])
		}
	}
	return ""
}
