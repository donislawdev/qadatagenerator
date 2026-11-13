package mutators

// The Unicode corpus this build carries, which is what the capability
// corpus.unicode stands for in a run.
//
// # Why a handful of values rather than a list on disk
//
// The other corpora are word lists, chosen by locale and too long to write
// out. This one is a few characters picked for what each of them breaks, and
// every one of them is the same in every locale. A file beside it would be a
// second place to look for something that fits on a screen.
//
// # Why every character is written as an escape
//
// Two of them cannot be seen, and a source file holding them as they are would
// look like it holds nothing. The eight digit escape is used throughout so that
// one form reads the same for all of them.
//
// # Why these characters and not others
//
// Each is chosen for the failure it reaches, not for how it looks.
const (
	// smilingFace is outside the basic plane, which is the part that matters.
	// It takes four bytes in UTF-8 and two units in UTF-16, so it reaches the
	// database character set that stores three bytes at most and the length
	// check that counts UTF-16 units. An emoji from the basic plane would pass
	// both and test nothing a Polish letter does not.
	smilingFace = "\U0001F642"

	// family is one character to a reader and seven code points underneath:
	// four people joined by three zero width joiners. A length limit counted
	// in code points sees seven, one counted in UTF-8 bytes sees twenty five,
	// and a cut made anywhere inside it leaves people a reader never typed.
	family = "\U0001F469\U0000200D\U0001F469\U0000200D\U0001F467\U0000200D\U0001F466"

	// zeroWidthSpaceText is invisible and is a character all the same. Two
	// values that differ by it look identical on every screen, which is what
	// makes it a way to register a second account under somebody else's name.
	zeroWidthSpaceText = "\U0000200B"
)

// capitalModifiers maps a Latin letter to the modifier letter that looks like
// its capital, raised.
//
// # Why the capital forms, and why that is the whole trick
//
// Compatibility normalisation turns each of these into the plain capital, and
// lower casing leaves them alone, because a modifier letter has no case. A
// pipeline that lower cases first and normalises second therefore turns BIGBIRD
// written in these into the capital BIGBIRD on the first pass and into bigbird
// on the second, which is the account takeover the catalog entry is about. The
// small modifier letters normalise to small letters already and would leave
// that pipeline stable, so they would supply nothing.
//
// Four letters have no raised capital in Unicode: S, X, Y and Z. They stay as
// they are, and one letter that does change is enough for the value to change
// between the two passes. A test holds every other one against the
// normalisation tables of the Unicode library, so a wrong code point here
// fails rather than quietly supplying a value that is stable.
//
// Both cases of a letter map to the same character, because the modifier is
// what the value is about and its case is the thing the pipeline gets wrong.
var capitalModifiers = map[rune]rune{
	'A': '\U00001D2C', 'B': '\U00001D2E', 'C': '\U0000A7F2', 'D': '\U00001D30',
	'E': '\U00001D31', 'F': '\U0000A7F3', 'G': '\U00001D33', 'H': '\U00001D34',
	'I': '\U00001D35', 'J': '\U00001D36', 'K': '\U00001D37', 'L': '\U00001D38',
	'M': '\U00001D39', 'N': '\U00001D3A', 'O': '\U00001D3C', 'P': '\U00001D3E',
	'Q': '\U0000A7F4', 'R': '\U00001D3F', 'T': '\U00001D40', 'U': '\U00001D41',
	'V': '\U00002C7D', 'W': '\U00001D42',
}
