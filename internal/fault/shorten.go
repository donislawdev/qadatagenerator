package fault

// Shortest is how much of any one piece of untrusted text survives in a
// message, in characters.
//
// Characters rather than bytes, and the unit is named because for the text this
// tool handles the two are different numbers. It is set well above every
// message this project writes, so it only ever cuts something that came from
// outside.
const Shortest = 200

// Shorten trims a piece of untrusted text to something a report can carry.
//
// # Why this exists, and why it lives here rather than beside one of its users
//
// Input from outside has a size limit or is a file somebody else chose the size
// of. The report about it has none unless something puts one there. Measured on
// 2026-08-02: a recipe carrying one two hundred kilobyte value produced a two
// hundred kilobyte error message, so hostile input came back amplified into
// somebody's build log, and with the machine readable flag onto the data
// channel where a pipeline parses it.
//
// 🔴 It lives in this package because it turned out to have more than one user,
// and the note left with the first fix asked exactly the right question: how
// many roads lead to the channel. The answer was three. The recipe reader was
// the first, the errors of the numbering schemes are the second and are still
// open, and the third arrived six days later with the reader of somebody else's
// file, where a column name is untrusted text of unbounded length. Measured on
// 2026-08-11 before that was bounded: a column named with sixty four kilobytes
// produced a refusal of sixty five thousand nine hundred and twenty seven
// bytes.
//
// A copy of this rule beside each user is the guard somebody eventually does
// not write, which is why the third road had none.
//
// # Where to apply it
//
// Where a message is recorded rather than at the dozen places that build one,
// wherever the recording is a single place. Names of entities, fields, keys and
// columns all come from outside and all end up inside a sentence, so bounding
// the sentence bounds every one of them, including the ones somebody writes
// next year.
//
// The cut lands on a character boundary. Cutting on a byte would split a multi
// byte character and put invalid UTF-8 into that same machine readable report.
func Shorten(text string) string {
	count := 0
	for index := range text {
		count++
		if count > Shortest {
			return text[:index] + "... (shortened)"
		}
	}
	return text
}
