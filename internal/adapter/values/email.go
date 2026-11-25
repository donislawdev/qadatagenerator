package values

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/text/unicode/norm"

	"github.com/donislawdev/qadatagenerator/internal/app/generate"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
	"github.com/donislawdev/qadatagenerator/internal/fault"
)

// An email address made of a person's name at a domain nobody receives mail at
// (DEC-340, point 6 of the owner's plan).
//
// # Whose name
//
// The name of the row. A row with a first name and a last name gets an address
// made of them, anna.nowak at a domain, the way a first name follows the gender
// column: the email reads the given name and the surname those columns play
// (provider.RoleGivenName and RoleSurname), and a recipe needs no rule for it. A
// row without them gets a name drawn for the address alone, from the lists of
// the locale, so an address is still a person's name rather than a string of
// letters.
//
// # Which domains, and who chose them
//
// example.com, example.net and example.org, chosen by the owner on 2026-11-25.
// RFC 2606 reserves all three for documentation, and each answered a lookup on
// that day with the null MX of RFC 7505, which says the domain takes no mail.
// So an address from here reaches nobody, and it still passes a form that
// checks whether the ending of a domain exists, which .test would not.
//
// # What the local part holds
//
// Lowercase letters and digits from the name, joined by a dot, an underscore
// or nothing, sometimes with two digits after it, which is what addresses look
// like in the wild. Letters with marks lose them, ł becomes l, and anything
// else, an apostrophe or a hyphen in a name, is left out, because many forms
// refuse them and an address a form should accept is the business of this
// generator, while one it may refuse is the business of damage. A local part
// never begins or ends with a dot and never holds two in a row, so it is a
// dot-atom of RFC 5322 section 3.2.3.

// emailDomains are the domains an address is drawn at.
var emailDomains = [...]string{"example.com", "example.net", "example.org"}

// emailShortest is the shortest address this generator writes: one letter, the
// at sign and the shortest domain.
var emailShortest = len("a@") + len(emailDomains[0])

// The shapes of a local part, from the given name g and the surname s.
const (
	shapeDotted    = iota // g.s
	shapeInitial          // gs with the initial of g alone
	shapeJoined           // gs
	shapeUnderline        // g_s
	shapeReversed         // s.g
	shapeCount
)

// email draws one address.
func (s Source) email(req generate.Request) (string, error) {
	given, err := s.nameFor(req, provider.RoleGivenName, Source.firstNames, ":given")
	if err != nil {
		return "", err
	}
	surname, err := s.nameFor(req, provider.RoleSurname, Source.lastNames, ":surname")
	if err != nil {
		return "", err
	}
	local := localPart(asciiName(given), asciiName(surname),
		int(s.seed.IntN(coordinate(req, ":shape"), shapeCount)))
	if s.seed.IntN(coordinate(req, ":numbered"), 2) == 1 {
		local += fmt.Sprintf("%02d", s.seed.IntN(coordinate(req, ":number"), 100))
	}
	domain := emailDomains[s.seed.IntN(coordinate(req, ":domain"), int64(len(emailDomains)))]
	return fitAddress(local, domain, req)
}

// nameFor is the name the row binds to a role, or one drawn for the address
// when the row binds none.
func (s Source) nameFor(req generate.Request, role string, entries listed, part string) (string, error) {
	if bound, given := req.Agrees.Value(role); given {
		return bound, nil
	}
	found, err := entries(s, generate.Request{At: req.At})
	if err != nil {
		return "", err
	}
	return found.entries[s.seed.IntN(coordinate(req, part), int64(len(found.entries)))], nil
}

// localPart joins the two names in one of the shapes. A name with nothing left
// of it once written in plain letters drops out of the shape, and a local part
// with nothing at all is user, so the address is still an address.
func localPart(given, surname string, shape int) string {
	switch {
	case given == "" && surname == "":
		return "user"
	case given == "":
		return surname
	case surname == "":
		return given
	}
	switch shape {
	case shapeInitial:
		return given[:1] + surname
	case shapeJoined:
		return given + surname
	case shapeUnderline:
		return given + "_" + surname
	case shapeReversed:
		return surname + "." + given
	}
	return given + "." + surname
}

// fitAddress writes the address, shortening the local part to what max_length
// leaves it, or refuses a limit no address fits.
//
// A shortened local part is cut, not chosen from, because there is nothing to
// choose from: it is made rather than drawn. A dot left at its end is cut as
// well, since a dot-atom cannot end with one.
func fitAddress(local, domain string, req generate.Request) (string, error) {
	if !req.HasMaxLength {
		return local + "@" + domain, nil
	}
	if req.MaxLength < emailShortest {
		return "", fault.New(fault.Config, fmt.Sprintf(
			"an email address here is at least %d characters, one letter, the at sign and a domain such as %s, and %s accepts %d",
			emailShortest, emailDomains[0], req.At.Field, req.MaxLength)).
			DoInstead("raise max_length to " + strconv.Itoa(emailShortest) + " or more")
	}
	room := req.MaxLength - len("@"+domain)
	if len(local) > room {
		local = strings.TrimRight(local[:room], "._")
	}
	return local + "@" + domain, nil
}

// asciiName writes a name in the lowercase letters and digits a local part
// holds: marks taken off, a few letters with no decomposition spelled out, and
// everything else left out.
func asciiName(name string) string {
	var written strings.Builder
	for _, r := range norm.NFD.String(strings.ToLower(name)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			written.WriteRune(r)
		default:
			written.WriteString(plainLetters[r])
		}
	}
	return written.String()
}

// plainLetters are the letters NFD leaves whole, written the way an address
// writes them.
var plainLetters = map[rune]string{
	'ł': "l", 'đ': "d", 'ø': "o", 'ı': "i", 'ß': "ss", 'æ': "ae", 'œ': "oe", 'þ': "th",
}

// emailDomain says a field limited below the shortest address has no value to
// give, so the plan refuses it before a row is written. Every other limit leaves
// the domain unknown.
func emailDomain(req generate.Request) (generate.Domain, bool) {
	if req.HasMaxLength && req.MaxLength < emailShortest {
		return generate.Domain{Key: Email}, true
	}
	return generate.Domain{}, false
}
