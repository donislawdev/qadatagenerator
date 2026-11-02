package intl

import "testing"

// 🔴 The rule DEC-268 admitted the numbers under, held on every one of them.
//
// A number joins the list only when at least two of the four providers whose
// test documentation was read publish it as a test number. One provider is a
// number out of one table, and a list that grew one such number at a time would
// become a copy of that table without anybody deciding it should.
func TestEveryPublishedNumberHasTwoProvidersBehindIt(t *testing.T) {
	read := map[string]bool{cardStripe: true, cardBraintree: true, cardAdyen: true, cardCheckout: true}
	seen := map[string]bool{}
	for _, published := range cardPublished {
		if seen[published.number] {
			t.Errorf("%s is on the list twice, so it is drawn twice as often as the others", published.number)
		}
		seen[published.number] = true

		providers := map[string]bool{}
		for _, by := range published.publishedBy {
			if !read[by] {
				t.Errorf("%s names %q, whose documentation was not among the four read", published.number, by)
			}
			providers[by] = true
		}
		if len(providers) < 2 {
			t.Errorf("%s has %d provider behind it, and a number joins only with two", published.number, len(providers))
		}
	}
}

// The brand written beside a number agrees with its form. American Express is
// the brand of the fifteen digit form, and every other brand here is sixteen.
func TestEveryPublishedNumberHasTheFormOfItsBrand(t *testing.T) {
	for _, published := range cardPublished {
		amex := published.brand == "American Express"
		switch {
		case amex && len(published.number) != cardAmexLength:
			t.Errorf("%s is written beside American Express and is %d digits", published.number, len(published.number))
		case !amex && len(published.number) != cardLength:
			t.Errorf("%s is written beside %s and is %d digits", published.number, published.brand, len(published.number))
		case !cardChecked(published.number):
			t.Errorf("%s is not a form this build checks", published.number)
		}
	}
}
