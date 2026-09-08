package recipeview

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/commandline"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
)

// buildCommand makes the line that would do this in a terminal, once.
//
// # Why this screen was the last of the three to get one
//
// The other two produce from what is ON them: the quick screen has a generator
// and a locale, the take screen has a file it was handed. This one produces
// from a FILE, and the file is the thing it does not have. What a person types
// here is text in a field, and the command line takes a path.
//
// The owner settled it on 2026-09-08 with the shape below: the line is always
// there, with the name of the file left as a filled in example, and the copy
// beside it cannot be pressed. Turned down was hiding the line until there is a
// path, which takes the bridge away from exactly the person who is learning
// what a recipe is, and turned down was writing a path this window guessed,
// which hands somebody a command naming a file that may not exist.
//
// 🔴 Worth knowing before changing this: nothing in this build writes a recipe
// file, so there is no state this screen can reach where the copy becomes
// pressable. It is disabled in all of them. That is a true statement of what
// the window can do rather than a control waiting on a value, and the sentence
// under it says so in words. The day this screen learns to save a recipe is the
// day the button has something to become.
func (v *View) buildCommand() {
	v.command = widget.NewLabelWithStyle("", fyne.TextAlignLeading,
		fyne.TextStyle{Monospace: true})
	// Cut rather than wrapped, for the reason the screen next door measured on
	// 2026-09-08: a label reports the width of its whole text as the least it
	// can be, so an uncut command drags the pane holding it wider than the
	// window. Nothing here asks anybody to read the line to its end.
	v.command.Wrapping = fyne.TextWrapOff
	v.command.Truncation = fyne.TextTruncateEllipsis
	// No clipboard is taken for this screen, because a control that cannot be
	// pressed cannot copy. Adding the port now would be a wire to nowhere, and
	// the screens that do copy show exactly what to add when it is needed.
	v.copyCommand = parts.NewButton(v.text.T(i18n.CommandCopy), func() {})
	v.copyCommand.Disable()
}

// commandContent is the line, the button, and the sentence saying why the
// button is the way it is.
//
// The sentence is not an apology. A command carrying an example file name is a
// command that fails when pasted, and this product refuses to hand somebody an
// answer it knows to be wrong. Saying which part is the example is what turns a
// line nobody can run into a line somebody can finish.
func (v *View) commandContent() fyne.CanvasObject {
	// The Panel stays inside the Card, which is a box inside a box and is meant
	// to be. The card says that this group is one thing and names it, and the
	// panel says that the line inside it is there to be TAKEN rather than read,
	// which is the same thing a block of code in a page of text says by having
	// a ground of its own. The two are not saying the same thing twice.
	return parts.Card(v.text.T(i18n.CommandHeading), nil, container.NewVBox(
		parts.Panel(v.command),
		container.NewHBox(v.copyCommand.OnScreen(), layout.NewSpacer()),
		parts.Detail(v.text.T(i18n.RecipeCommandExample)),
	))
}

// commandLine writes out what a terminal would be asked for.
//
// The settings that ARE real here are written as they stand: the directory
// somebody typed and whether they ticked replace. Only the file is an example,
// which is what keeps the rest of the line worth reading.
func (v *View) commandLine() string {
	return commandline.New(commandline.Generate).
		Flag(commandline.Out, v.out.Text).
		Switch(commandline.Overwrite, v.replace.Checked).
		Argument(v.text.T(i18n.RecipeCommandFile)).
		String()
}
