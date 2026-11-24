package values_test

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/recipe"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// bounded is the request for a number with the bounds a recipe wrote.
func bounded(kind, low, high string, row int64) generate.Request {
	request := byType(kind, "n", row)
	request.MinValue, request.MaxValue = low, high
	return request
}

// shapes are how a number of each type is written in a column.
var shapes = map[string]*regexp.Regexp{
	values.TypeInteger: regexp.MustCompile(`^-?(0|[1-9][0-9]*)$`),
	values.TypeDecimal: regexp.MustCompile(`^-?(0|[1-9][0-9]*)\.[0-9]{2}$`),
}

// strayed returns the first value that is not written as its type writes a
// number or lies outside low and high, and nothing when every value is inside.
//
// The ends are compared in hundredths, which holds both types exactly.
func strayed(kind string, column []string, low, high string) string {
	bottom, top := hundredthsOf(low), hundredthsOf(high)
	for _, value := range column {
		if !shapes[kind].MatchString(value) {
			return value
		}
		if amount := hundredthsOf(value); amount < bottom || amount > top {
			return value
		}
	}
	return ""
}

// hundredthsOf reads a number written with at most two places as hundredths.
func hundredthsOf(text string) int64 {
	whole, fraction, _ := strings.Cut(strings.TrimPrefix(text, "-"), ".")
	amount, err := strconv.ParseInt(whole+(fraction + "00")[:2], 10, 64)
	if err != nil {
		panic(fmt.Sprintf("%q is not a number this test reads", text))
	}
	if strings.HasPrefix(text, "-") {
		return -amount
	}
	return amount
}

// Every number a bounded field draws lies between its bounds, both included,
// is written the way its type writes a number, and the two ends are values the
// field can take. An end the recipe leaves out is the one the field has
// without bounds: nought below, six digits above.
func TestANumberStaysInsideItsBounds(t *testing.T) {
	// The canary: the check below has to see a value past an end and a value
	// written with the wrong number of places, or a green run proves nothing.
	if strayed(values.TypeInteger, []string{"5", "11"}, "1", "10") != "11" {
		t.Fatal("the check passed a value above the largest bound")
	}
	if strayed(values.TypeDecimal, []string{"1.5"}, "0", "9.99") != "1.5" {
		t.Fatal("the check passed a decimal written with one place")
	}

	source := values.New("en-US", "bounds")
	for _, given := range []struct {
		name, kind, low, high string
		// first and last are the ends as the column writes them, and size is
		// how many numbers lie between them.
		first, last string
		size        int64
	}{
		{"a quantity", values.TypeInteger, "1", "10", "1", "10", 10},
		{"across nought", values.TypeInteger, "-50", "50", "-50", "50", 101},
		{"one number", values.TypeInteger, "7", "7", "7", "7", 1},
		{"only a smallest", values.TypeInteger, "999990", "", "999990", "999999", 10},
		{"only a largest", values.TypeInteger, "", "3", "0", "3", 4},
		{"a price", values.TypeDecimal, "0.5", "99.99", "0.50", "99.99", 9950},
		{"a refund", values.TypeDecimal, "-5", "0", "-5.00", "0.00", 501},
		{"a decimal with only a largest", values.TypeDecimal, "", "0.1", "0.00", "0.10", 11},
	} {
		t.Run(given.name, func(t *testing.T) {
			domain, known := source.DomainOf(bounded(given.kind, given.low, given.high, 0))
			if !known || domain.Size != given.size {
				t.Fatalf("the field can take %d numbers (known %v), and should take %d", domain.Size, known, given.size)
			}
			if first, last := domain.At(0), domain.At(domain.Size-1); first != given.first || last != given.last {
				t.Errorf("the numbers run from %q to %q, and should run from %q to %q", first, last, given.first, given.last)
			}
			column := make([]string, 0, 400)
			for row := int64(0); row < 400; row++ {
				value, err := source.Produce(bounded(given.kind, given.low, given.high, row))
				if err != nil {
					t.Fatalf("row %d: %v", row, err)
				}
				column = append(column, value)
			}
			if value := strayed(given.kind, column, given.first, given.last); value != "" {
				t.Errorf("the field drew %q, outside %s to %s or not written as a %s", value, given.first, given.last, given.kind)
			}
		})
	}
}

// A range the field cannot hold is refused before a row is written, saying
// what is wrong with it and what to do instead, and the plan names the field.
//
// Each of these would otherwise be settled quietly: a bound rounded to the
// places of its type, two ends swapped, a number longer than its column, a
// bound on a type that draws no number at all.
func TestARangeTheFieldCannotHoldIsRefusedWithWhatToDo(t *testing.T) {
	for _, given := range []struct {
		name  string
		field recipe.Field
		says  string
	}{
		{"ends the wrong way round", recipe.Field{Type: values.TypeInteger, MinValue: "10", MaxValue: "1"},
			"min_value 10 is above max_value 1"},
		{"a fraction for an integer", recipe.Field{Type: values.TypeInteger, MinValue: "2.5"},
			"is not a whole number"},
		{"three places for a decimal", recipe.Field{Type: values.TypeDecimal, MaxValue: "0.125"},
			"has 3 decimal places"},
		{"a smallest above the default largest", recipe.Field{Type: values.TypeInteger, MinValue: "1000000"},
			"the largest number the field draws without max_value"},
		{"a largest below nought", recipe.Field{Type: values.TypeInteger, MaxValue: "-1"},
			"the smallest number the field draws without min_value"},
		{"a largest longer than the column", recipe.Field{Type: values.TypeInteger, MaxValue: "1000",
			MaxLength: 3, HasMaxLength: true}, "1000, the largest number the field draws, is 4 characters long"},
		{"a sign longer than the column", recipe.Field{Type: values.TypeDecimal, MinValue: "-5",
			MaxLength: 4, HasMaxLength: true}, "-5.00, the smallest number the field draws, is 5 characters long"},
		{"more digits than a draw holds", recipe.Field{Type: values.TypeInteger, MaxValue: "1000000000000000000"},
			"has more digits than this build draws"},
		{"a bound on a type that is no number", recipe.Field{Type: "money", MaxValue: "5"},
			"declared as money"},
		{"a bound that is no number", recipe.Field{Type: values.TypeInteger, MaxValue: "ten"},
			"is not a number"},
	} {
		t.Run(given.name, func(t *testing.T) {
			given.field.Name = "n"
			rec := person(3, []recipe.Field{given.field})
			_, err := generate.Prepare(rec, values.New(rec.Locale, rec.Seed), generate.NeverStops{})
			if err == nil {
				t.Fatal("the plan accepted a range the field cannot hold")
			}
			if fault.KindOf(err) != fault.Config || !strings.Contains(err.Error(), "entities.person.fields.n: ") ||
				!strings.Contains(err.Error(), given.says) {
				t.Errorf("refused as %v with %q, and it should name the field and say %q", fault.KindOf(err), err, given.says)
			}
			if fault.InsteadOf(err) == "" {
				t.Errorf("%q says nothing about what to do instead", err)
			}
		})
	}
}
