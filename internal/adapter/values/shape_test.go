package values_test

import (
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
)

// The shape of a field is the kind its values are made as and, for a number,
// the width of the longest one the field can hold (DEC-341).
//
// The widths are the ends of the range the draw uses, so a table typed from
// them holds every number the run writes: 999.99 is the largest decimal of a
// field with no bound, 100 the largest of a range up to it, and a list holds
// what is on it, its longest whole part and its most places.
func TestTheShapeOfAFieldIsWhatItsValuesAreMadeAs(t *testing.T) {
	source := values.New("en-US", "shape")
	for _, given := range []struct {
		name  string
		field recipe.Field
		want  generate.Shape
	}{
		{"a word list", recipe.Field{Generator: values.FirstName}, generate.Shape{Kind: generate.ShapeText}},
		{"an identifier", recipe.Field{Generator: values.SSN}, generate.Shape{Kind: generate.ShapeText}},
		{"a uuid by generator", recipe.Field{Generator: values.UUID}, generate.Shape{Kind: generate.ShapeUUID}},
		{"a uuid by type", recipe.Field{Type: values.TypeUUID}, generate.Shape{Kind: generate.ShapeUUID}},
		{"a date of birth", recipe.Field{Generator: values.BirthDate}, generate.Shape{Kind: generate.ShapeDate}},
		{"a date by type", recipe.Field{Type: values.TypeDate}, generate.Shape{Kind: generate.ShapeDate}},
		{"a list of words", recipe.Field{Values: []string{"free", "pro"}}, generate.Shape{Kind: generate.ShapeText}},
		{"an expression", recipe.Field{Expression: "${a}"}, generate.Shape{Kind: generate.ShapeText}},
		{"a whole number with no bound", recipe.Field{Type: values.TypeInteger},
			generate.Shape{Kind: generate.ShapeInteger, Digits: 6}},
		{"a whole number of four characters", recipe.Field{Type: values.TypeInteger, MaxLength: 4, HasMaxLength: true},
			generate.Shape{Kind: generate.ShapeInteger, Digits: 4}},
		{"a whole number from -5 to 100", recipe.Field{Type: values.TypeInteger, MinValue: "-5", MaxValue: "100"},
			generate.Shape{Kind: generate.ShapeInteger, Digits: 3}},
		{"a whole number from -12345", recipe.Field{Type: values.TypeInteger, MinValue: "-12345", MaxValue: "7"},
			generate.Shape{Kind: generate.ShapeInteger, Digits: 5}},
		{"a decimal with no bound", recipe.Field{Type: values.TypeDecimal},
			generate.Shape{Kind: generate.ShapeDecimal, Digits: 3, Places: 2}},
		{"a decimal up to 99999.99", recipe.Field{Type: values.TypeDecimal, MaxValue: "99999.99"},
			generate.Shape{Kind: generate.ShapeDecimal, Digits: 5, Places: 2}},
		{"a list of whole numbers", recipe.Field{Type: values.TypeInteger, Values: []string{"1", "-123"}},
			generate.Shape{Kind: generate.ShapeInteger, Digits: 3}},
		{"a list of decimals", recipe.Field{Type: values.TypeDecimal, Values: []string{"1.5", "22.125", "7"}},
			generate.Shape{Kind: generate.ShapeDecimal, Digits: 2, Places: 3}},
		{"a list of long whole numbers", recipe.Field{Type: values.TypeInteger,
			Values: []string{"1234567890123456789012"}}, generate.Shape{Kind: generate.ShapeInteger, Digits: 22}},
		{"a range the plan refuses", recipe.Field{Type: values.TypeInteger, MinValue: "9", MaxValue: "1"},
			generate.Shape{Kind: generate.ShapeInteger, Digits: 18}},
	} {
		if got := source.ShapeOf(generate.FieldRequest(given.field)); got != given.want {
			t.Errorf("%s: the shape is %+v, want %+v", given.name, got, given.want)
		}
	}
}
