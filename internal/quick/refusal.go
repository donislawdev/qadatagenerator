package quick

import (
	"sort"
	"strconv"
	"time"

	"github.com/donislawdev/qadatagenerator/internal/adapter/values"
	"github.com/donislawdev/qadatagenerator/internal/domain/provider"
)

// RefusalKind is a request this mode will not carry out, named rather than
// worded.
//
// # Why a kind and not the sentence itself
//
// The same reason the mode next door gives: the command line is English always,
// by an unbreakable rule, while the window shows whatever it was translated
// into and takes its words from a catalogue of keys. A shared sentence would
// force one of the two to be wrong, and the one it would break is the window,
// quietly, by putting an untranslated string on a translated screen.
//
// What both surfaces share is which requests are refused and why, which is what
// a kind carries. Each surface then owns its own four parts.
type RefusalKind int

const (
	// NotRefused is the zero value, so that a refusal nobody filled in cannot
	// be mistaken for a real one.
	NotRefused RefusalKind = iota
	// UnknownGenerator names a value this build cannot produce.
	UnknownGenerator
	// RoleNotRead binds a value to something the chosen generator takes no
	// account of, such as a gender beside a city.
	RoleNotRead
	// UnreadableGender is a gender outside the spellings this mode takes.
	UnreadableGender
	// UnreadableDate is a date of birth that is not written as YYYY-MM-DD.
	UnreadableDate
	// CountNotPositive asks for fewer than one value.
	//
	// 🔴 A recipe ACCEPTS a count of nought and this refuses it, which is not
	// an inconsistency for the same reason maxLength refuses a nought that
	// count allows: no rows of an entity is a smaller dataset, and somebody
	// may mean it. No values here is a run that produces nothing at all, and
	// on a channel carrying only values it is indistinguishable from a failure
	// that printed nothing. This verb exists to answer "give me a value".
	CountNotPositive
	// CountNotWhole is text asked for as a number of values that never became
	// one.
	//
	// 🔴 It lives here rather than on each surface, and that is the whole
	// reason it exists. The mode takes an int, so until 2026-09-09 the
	// question "is this text a number" could only be answered by whoever held
	// the text - the command line answered it in wholeCount and the window had
	// no field to ask about. The moment the window offers a field somebody
	// types into (DEC-235), a second answer would be written, and two answers
	// to one question are how two surfaces come to disagree about the same
	// value.
	CountNotWhole
	// Unavailable is a kind this build cannot produce in the locale asked
	// for, because the data it is drawn from is not carried there: the postal
	// codes of pl-PL, or any word list in a locale this build has never heard
	// of. Allowed names the locales the kind is produced in.
	//
	// 🔴 Refused here rather than left to the run, because the run knows why
	// and not what to do about it in this mode. The run is shared with
	// recipes, and the next move for a recipe - another locale for the whole
	// dataset, or the field left out - names two things this mode does not
	// have. What both surfaces of this mode can point at is another locale.
	Unavailable

	// kindsEnd is not a refusal. It stands after the last one so that Kinds
	// counts up to it rather than keeping a second list of the kinds above,
	// which a new one could be left out of.
	kindsEnd
)

// Kinds is every refusal this mode can produce, so that a surface can be held
// to answering for each of them.
//
// 🔴 Counted rather than listed. Each surface turns a kind into words through
// a register, and a guard beside each register holds it against this answer in
// both directions. A list written out here would be one more place to forget a
// new kind in, and the guards would then agree with the list instead of with
// the constants.
func Kinds() []RefusalKind {
	kinds := make([]RefusalKind, 0, int(kindsEnd-NotRefused-1))
	for kind := NotRefused + 1; kind < kindsEnd; kind++ {
		kinds = append(kinds, kind)
	}
	return kinds
}

// MostShownAtOnce is the largest number of values a SCREEN draws at once.
//
// 🔴 It is NOT a limit of this mode and Check does not apply it. Run will
// produce a million values if asked, and the command line asks for them - the
// number here exists because a window has to build a row for each one on the
// thread that draws, and that is a cost the command line does not have.
//
// It lives in the mode for the same reason DateShape does: a fact a surface
// needs, kept where both surfaces can see it, so that the window is not the
// only place it is written down. The guard beside this file holds the other
// half - that asking Check for this many values is not a refusal.
//
// # 🔴 Why a thousand, lowered from twenty thousand on 2026-10-21 (DEC-262)
//
// The owner typed a million into the real window. The field took its first five
// digits, so ten thousand values were drawn without a word about the rest, and
// the whole window lagged under the pointer from then on. The measurement below
// says why the number had been twenty thousand, and its own last paragraph says
// what it left out, which is exactly what the owner ran into: a window that has
// built the card still repaints and walks it on every move of the pointer.
//
// A thousand is the owner's choice, made on what this screen is for: values to
// paste into a form or a test. A set larger than that is a file, and the line at
// the foot of the screen carries no limit at all. Not measured against painting
// either, and the margin is now a factor of twenty under the size that lagged.
//
// What follows is the measurement that set the number until then, kept because
// the building half of it is still true.
//
// Measured on the card that draws them, three runs of ten at each size, median:
//
//	   500 values     0.52 ms     0.57 MB     11 761 allocations
//	 5 000 values     5.7  ms     5.7  MB    119 762 allocations
//	20 000 values    27.4  ms    22.8  MB    479 765 allocations
//	50 000 values    99    ms    57.0  MB  1 199 768 allocations
//
// Building is linear and cheap - about 1.1 microseconds and 1.14 kB a value -
// so the first number tried, five hundred, was picked to sound safe rather than
// measured, which is the thing the list of four counts was replaced for. Twenty
// thousand rather than fifty: at fifty the spread opens to 77 against 133 ms,
// so the measurement has stopped being stable, and the memory is the card alone,
// rebuilt whole on every draw.
//
// 🔴 What it is NOT measured against: painting. The benchmark builds the card
// and asks its size, which is what blocks the drawing thread. How long the
// toolkit then spends painting half a million objects a frame is not measured,
// so this number carries the margin of that ignorance. Verified by hand at the
// size the window is kept: twenty thousand values are on screen and it answers.
// (That sentence was about the window answering ONCE, and it did not survive
// the owner using it, see the top of this comment.)
const MostShownAtOnce = 1000

// CountFrom reads how many values were asked for out of the text somebody
// typed or passed on a flag.
//
// 🔴 strconv.Atoi is base ten whatever the digits look like, and that matters
// here for a reason this project has already paid for: a count of 010 in a
// recipe once produced EIGHT rows, because the reader it went through read a
// leading nought as octal (DEC-170). Atoi reads 010 as ten.
//
// It answers the SHAPE only. Whether the number is a number of values - at
// least one - is Refused's business, so that the two questions keep their two
// answers and neither surface has to remember both.
//
// 🔴 It does NOT trim, and that is deliberate rather than an omission. A flag
// carries what somebody passed and a padded number is a mistake worth naming,
// while a field carries what somebody pasted and a seed copied out of a log
// brings its spaces with it. So the window trims before asking and the command
// line does not - loosening this here would have quietly widened what an
// already published flag accepts.
func CountFrom(text string) (int, Refusal) {
	number, err := strconv.Atoi(text)
	if err != nil {
		return 0, Refusal{Kind: CountNotWhole, Value: text}
	}
	return number, Refusal{}
}

// Refusal is why a value will not be produced, with what it needs to be said
// properly.
//
// Value and Allowed are here for the reason every refusal in this project
// carries them: a message has to name the offending value and what would have
// been taken instead, and working that out again on each surface is how the two
// come to disagree.
type Refusal struct {
	// Kind is what was wrong.
	Kind RefusalKind
	// Value is the offending value.
	Value string
	// Role is which role was bound, for a refusal about one.
	Role string
	// Locale is named only where it is what makes the answer no: a role read
	// in one locale and not in another, or a kind produced in one and not in
	// another.
	Locale string
	// Allowed is what this build would have taken: the generators it has, the
	// spellings of a gender, the generators that do read the role, or the
	// locales the kind is produced in.
	Allowed []string
}

// Refused reports whether this is a refusal at all.
func (r Refusal) Refused() bool { return r.Kind != NotRefused }

// Check says whether this request will be carried out, before anything is
// drawn.
//
// # Why a bound role nothing reads is refused rather than ignored
//
// A recipe binds a rule across a whole table and some columns of that table act
// on it, so a column that does not is ordinary. Here there is one value, so a
// bound role the generator ignores has no other column to be for. Accepting it
// would answer a question nobody asked and say nothing about it, which is the
// silence this product refuses everywhere else.
//
// # Why the gender is checked here at all
//
// The vocabulary is the domain's and every reader of a gender shares it, so
// this is not a second opinion about what the words mean. What it adds is WHEN
// the answer comes: a word this product has no meaning for is refused before
// anything is drawn, with the accepted spellings named, rather than reaching a
// scheme that refuses it in its own words or a word list that quietly draws
// from both.
//
// 🔴 This mode took two spellings until 2026-09-06, and the reason was that a
// first name compared the bound value against the word female itself while a
// PESEL took all twelve, so a narrower verb was one that did not inherit the
// disagreement. DEC-214 removed the disagreement, so the narrowness lost its
// reason and went with it.
func Check(req Request) Refusal {
	source := values.New(req.Locale, req.Seed)
	if !source.Known(req.Generator) {
		return Refusal{Kind: UnknownGenerator, Value: req.Generator, Allowed: source.Names()}
	}
	// Before the generator is asked anything, because how many were wanted is
	// a question about the request rather than about the value, and a run of
	// nought values would otherwise be refused by nothing at all.
	if refused := CheckCount(req.Count); refused.Refused() {
		return refused
	}
	// Before the roles, because a kind that cannot be produced here makes
	// every question about what is bound to it beside the point.
	if fine, _ := source.CanProduce(req.Generator, req.Locale); !fine {
		return Refusal{
			Kind: Unavailable, Value: req.Generator, Locale: req.Locale,
			Allowed: localesOf(source, req.Generator),
		}
	}
	for _, role := range sortedRoles(req.Agrees) {
		if !source.ReadsRole(req.Generator, role) {
			return Refusal{
				Kind: RoleNotRead, Value: req.Generator, Role: role,
				Locale: req.Locale, Allowed: readersOf(source, role),
			}
		}
	}
	if bound, ok := req.Agrees[provider.RoleGender]; ok {
		if _, known := provider.ParseGender(bound); !known {
			return Refusal{Kind: UnreadableGender, Value: bound, Role: provider.RoleGender,
				Allowed: AcceptedGenders()}
		}
	}
	if bound, ok := req.Agrees[provider.RoleBirthDate]; ok {
		if _, err := time.Parse(DateLayout, bound); err != nil {
			// No Allowed here, and that is the rule of this file working: the
			// layout a person reads is YYYY-MM-DD, which is a word for a
			// person, and DateLayout is how Go spells the same thing. Each
			// surface writes the readable one in its own words.
			return Refusal{Kind: UnreadableDate, Value: bound, Role: provider.RoleBirthDate}
		}
	}
	return Refusal{}
}

// CheckCount says whether a number is a number of values this mode produces,
// which is at least one.
//
// Check asks it, and a surface asks it on its own while somebody is still typing
// the number (B-363), so the answer given before the press and the one given by
// it are one rule rather than two.
func CheckCount(count int) Refusal {
	if count < 1 {
		return Refusal{Kind: CountNotPositive, Value: strconv.Itoa(count)}
	}
	return Refusal{}
}

// readersOf names the generators that would act on that role, so that a refusal
// points somewhere rather than only saying no.
func readersOf(source values.Source, role string) []string {
	var readers []string
	for _, generator := range source.Names() {
		if source.ReadsRole(generator, role) {
			readers = append(readers, generator)
		}
	}
	return readers
}

// localesOf names the locales a kind is produced in, so that a refusal for the
// one asked about points at the ones that would do. It can be empty, and each
// surface words its next move so that it still reads true when it is.
func localesOf(source values.Source, generator string) []string {
	var producing []string
	for _, locale := range Locales() {
		if fine, _ := source.CanProduce(generator, locale); fine {
			producing = append(producing, locale)
		}
	}
	return producing
}

func sortedRoles(agrees map[string]string) []string {
	roles := make([]string, 0, len(agrees))
	for role := range agrees {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	return roles
}
