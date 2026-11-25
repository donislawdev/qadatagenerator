package values

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// A password in plain text, for a test that has to log in or sign up
// (DEC-340, point 6 of the owner's plan).
//
// # What one looks like, and who chose it
//
// Twelve to sixteen characters, and always at least one lowercase letter, one
// uppercase letter, one digit and one symbol, which is what the common password
// rules ask for, so a value from here is one such a rule accepts. The owner chose
// the symbols on 2026-11-25: none of them is a quote, a backslash, a space or an
// angle bracket. So a password needs no escaping in CSV or in JSON, and in a
// shell it can stand between single quotes as it is, because the set has no
// apostrophe. It does need those quotes there: a dollar or an exclamation mark
// means something to a shell. A password a form should refuse is the business
// of damage, not of this generator.
//
// # Why it never begins with = + - or @
//
// A spreadsheet opening a CSV takes a cell beginning with one of those four for
// a formula, so a password beginning with one would turn into something else
// the moment somebody looked at the file. Those four may stand anywhere else.
//
// # What it is not
//
// Secret. It comes from the seed like every value here, so anybody holding the
// seed makes the same passwords. They are for tests and never for an account
// somebody uses.

// The classes a password draws from, one character of each at least.
const (
	passwordLower   = "abcdefghijklmnopqrstuvwxyz"
	passwordUpper   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	passwordDigits  = "0123456789"
	passwordSymbols = "!#$%&*+-=?@^_"
)

// passwordClasses are the four classes in the order a password takes its first
// character of each.
var passwordClasses = [...]string{passwordLower, passwordUpper, passwordDigits, passwordSymbols}

// passwordAlphabet is every character a password may hold.
var passwordAlphabet = strings.Join(passwordClasses[:], "")

// The length of a password when max_length does not cut it shorter.
const (
	passwordShortest = 12
	passwordLongest  = 16
)

// formulaStarts are the characters a spreadsheet reads a cell beginning with as
// a formula.
const formulaStarts = "=+-@"

// passwordFloor is the shortest a password can be and still hold a character of
// every class.
const passwordFloor = len(passwordClasses)

// password draws one password.
//
// The length is drawn first, then one character of each class, then the rest
// from every class together, and then the whole is shuffled, so a class does not
// keep the place it was drawn in. Each draw has an address of its own, so a
// value does not depend on the order the rows are asked for.
func (s Source) password(req generate.Request) (string, error) {
	longest := passwordLongest
	if req.HasMaxLength && req.MaxLength < longest {
		longest = req.MaxLength
	}
	if longest < passwordFloor {
		return "", fault.New(fault.Config, fmt.Sprintf(
			"a password holds a lowercase letter, an uppercase letter, a digit and a symbol, so it needs at least %d characters, and %s accepts %d",
			passwordFloor, req.At.Field, longest)).
			DoInstead("raise max_length to " + strconv.Itoa(passwordFloor) + " or more")
	}
	shortest := min(passwordShortest, longest)
	length := shortest + int(s.seed.IntN(coordinate(req, ":length"), int64(longest-shortest+1)))

	drawn := make([]byte, 0, length)
	for index, class := range passwordClasses {
		drawn = append(drawn, pick(s, req, ":class:"+strconv.Itoa(index), class))
	}
	for index := len(drawn); index < length; index++ {
		drawn = append(drawn, pick(s, req, ":char:"+strconv.Itoa(index), passwordAlphabet))
	}
	for index := len(drawn) - 1; index > 0; index-- {
		other := s.seed.IntN(coordinate(req, ":shuffle:"+strconv.Itoa(index)), int64(index+1))
		drawn[index], drawn[other] = drawn[other], drawn[index]
	}
	if strings.IndexByte(formulaStarts, drawn[0]) >= 0 {
		// A letter or a digit is always there, one of each was drawn.
		for index := 1; index < len(drawn); index++ {
			if strings.IndexByte(passwordSymbols, drawn[index]) < 0 {
				drawn[0], drawn[index] = drawn[index], drawn[0]
				break
			}
		}
	}
	return string(drawn), nil
}

// pick draws one character of an alphabet at one part of a value.
func pick(s Source, req generate.Request, part, alphabet string) byte {
	return alphabet[s.seed.IntN(coordinate(req, part), int64(len(alphabet)))]
}

// passwordDomain says a field limited below the floor of a password has no
// value to give, so the plan refuses it before a row is written, with the
// sentence the draw would refuse it with. Every other limit leaves the domain
// unknown: it is far larger than any table.
func passwordDomain(req generate.Request) (generate.Domain, bool) {
	if req.HasMaxLength && req.MaxLength < passwordFloor {
		return generate.Domain{Key: Password}, true
	}
	return generate.Domain{}, false
}
