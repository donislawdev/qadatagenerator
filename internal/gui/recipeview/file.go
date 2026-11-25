package recipeview

import (
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/adapter/recipefile"
	"github.com/donislawdev/qadatagenerator/internal/fault"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	"github.com/donislawdev/qadatagenerator/internal/gui/parts"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
)

// The recipe as a file: opened into the field, saved out of it, and not lost
// when the window closes (DEC-342, piece 8a).
//
// # Why this exists
//
// The screen says a recipe is a file kept beside the tests, and until
// 2026-11-25 it could neither open one nor write one. A recipe somebody built
// here reached a file only through the clipboard, and the cross took it away
// without a word. The window keeps nothing of a recipe between runs, because
// About promises that it keeps the size of the window and a folder and nothing
// else, so the answer is the file itself and a question before it is lost.
//
// # What a save may never do
//
// Leave the file half written. The bytes go to a temporary file beside it and
// reach the name by one rename (store.Replace), so a save that fails leaves the
// file as it was and the recipe still on the screen.
//
// 🔴 The save dialog of the toolkit empties the file it is pointed at before it
// hands the file over: dialog/file.go calls storage.Writer, which creates the
// file, and creating a file that is there truncates it. So a person who picks a
// file that already exists in Save as, agrees to replace it, and then meets a
// save that fails, is left with that file empty rather than as it was. The
// recipe is still on the screen in that case and nothing of theirs that was not
// going to be replaced is lost, but the gap is real and the toolkit closes it
// only with a dialog of this window's own, which piece 8f decides.

// buildFile makes the three buttons and the line naming the file, once, for the
// reason buildBand gives: a control rebuilt under somebody loses the keyboard.
//
// None of the three is ever disabled. A button that went grey once a file was
// saved would take the keyboard standing on it to the top of the screen, which
// is what the handover of a run exists to avoid, and the line beside them
// already says whether there is anything to save.
func (v *View) buildFile() {
	v.openFile = parts.NewButton(v.text.T(i18n.RecipeOpen), v.chooseToOpen)
	v.saveFile = parts.NewButton(v.text.T(i18n.RecipeSave), func() { v.Save(nil) })
	v.saveFileAs = parts.NewButton(v.text.T(i18n.RecipeSaveAs), func() { v.chooseToSave(nil) })
	// The name of the file is cut rather than folded, so a long one gives
	// up its end rather than widening the column it stands in.
	v.fileName = paint.Cut("")
	v.files = background.NewLatest(v.work.Read)
	if v.work.AskFirst != nil {
		v.work.AskFirst(v.keepFirst)
	}
}

// fileHeading is the heading of the first step with the three buttons at its
// far end, and under the words of the heading the name of the file the recipe
// came from and whether it is saved.
//
// # Why the name has a line of its own
//
// It stood between the heading and the buttons first, and the render of
// 2026-11-25 cut it to two letters at the size the window opens at: the three
// buttons and the heading take 360 of the 410 units the column has. A line
// under the heading costs the editor one line of height and lets the name have
// the whole width of the column.
//
// The line is there in every state, empty when there is nothing to name, so
// that the editor under it does not move when a recipe is first typed.
func (v *View) fileHeading(step fyne.CanvasObject) fyne.CanvasObject {
	actions := paint.Row(palette.NameStepRelated,
		v.openFile.OnScreen(), v.saveFile.OnScreen(), v.saveFileAs.OnScreen())
	words := paint.Middle(step)
	heading := container.New(layout.NewBorderLayout(nil, nil, words, actions), words, actions)
	// The words of a label stand one hairline in from its edge, and the words
	// of the heading one tight step past the disc, so the line is moved in by
	// the difference to start under them.
	inset := palette.StepDisc + palette.StepOf(palette.NameStepTight) - palette.StepOf(palette.NameStepHairline)
	name := paint.Lead(paint.Space(inset, 0), v.fileName)
	return container.New(layout.NewVBoxLayout(), heading, name)
}

// drawFile says which file the recipe on the screen is, and whether what is on
// the screen is what the file holds.
func (v *View) drawFile() {
	v.fileName.SetText(v.fileSays())
}

// fileSays is the line beside the heading. Nothing at all on an empty screen
// that came from no file, where there is nothing to name and nothing to lose.
func (v *View) fileSays() string {
	name := filepath.Base(v.kept.Path)
	switch {
	case v.opening != "":
		return v.text.T(i18n.RecipeFileOpening) + " " + filepath.Base(v.opening)
	case v.saving:
		return v.text.T(i18n.RecipeFileSaving)
	case v.kept.Path == "" && v.editor.Text == "":
		return ""
	case v.kept.Path == "":
		return v.text.T(i18n.RecipeFileNone)
	case v.unsaved():
		return name + " (" + v.text.T(i18n.RecipeFileChanged) + ")"
	}
	return name
}

// unsaved reports whether the recipe on the screen holds something no file
// does: text never saved, or changes made since the file was opened or saved.
// Text on its way to the disk in a save going on is not counted, because the
// save is already keeping it.
func (v *View) unsaved() bool {
	text := v.editor.Text
	if v.saving && text == v.savingText {
		return false
	}
	return text != v.kept.Text
}

// File is the path of the file the recipe on the screen came from or was last
// saved to, and empty for a recipe never saved, for a caller with nobody
// looking at the screen.
func (v *View) File() string { return v.kept.Path }

// Unsaved reports whether closing the window now would ask first, for the same
// callers.
func (v *View) Unsaved() bool { return v.unsaved() }

// Asked is the question the screen is waiting for an answer to, or nothing, so
// that a test can answer it the way a person would. A screen with no window
// behind it asks without showing anything, which is how its questions are
// answered in a test.
func (v *View) Asked() *parts.Question { return v.question }

// OpenFile reads a recipe file into the field, off the thread that draws.
//
// It does not ask about the recipe on the screen, which the button and a drop
// both do before calling this. Exported for the reason SetRecipe is: a test
// and a picture reach the state a person reaches by choosing a file.
//
// 🔴 The read goes to a place of work for the reason the check does: a file can
// be on a share that answers slowly or a pipe that never answers, and the
// window has to go on answering meanwhile.
func (v *View) OpenFile(path string) {
	v.opening = path
	v.drawFile()
	v.files.Ask(func(*background.Stopper) func() {
		opened, err := recipefile.Read(path)
		return func() {
			v.opening = ""
			if err != nil {
				v.drawFile()
				v.tell(i18n.RecipeNotOpened, fault.Says(err))
				return
			}
			v.Took(opened)
		}
	}, func(err error) func() {
		return func() {
			v.opening = ""
			v.drawFile()
			v.tell(i18n.RecipeNotOpened, fault.Says(err))
		}
	})
}

// Took puts a recipe read from a file on the screen and remembers the file.
//
// Exported for the stored pictures, which reach a screen holding a file
// without a disk, by handing over what a read would have answered. The file is
// remembered BEFORE the text goes in, because putting the text in judges it
// and draws the line beside the heading, which must then say saved rather
// than changed.
func (v *View) Took(opened recipefile.Opened) {
	v.kept = opened
	v.SetRecipe(opened.Text)
	v.drawFile()
	v.drawCommand()
	v.focusEditor()
}

// Dropped takes a file dropped onto the window while this screen is open.
//
// Only the first, as on the take screen, and only after asking about a recipe
// on the screen that no file holds. Any file is taken, whatever its name ends
// in, because the command line reads a recipe from any path and a file that
// is not a recipe is said so by the check, in the card, as for one typed.
func (v *View) Dropped(dropped []fyne.URI) {
	if len(dropped) == 0 || v.question != nil {
		return
	}
	path := dropped[0].Path()
	v.keepFirst(func() { v.OpenFile(path) }, func() {})
}

// chooseToOpen asks about the recipe on the screen, then for a file.
func (v *View) chooseToOpen() {
	if v.question != nil {
		return
	}
	v.keepFirst(func() {
		if v.window == nil {
			return
		}
		chooser := dialog.NewFileOpen(func(chosen fyne.URIReadCloser, err error) {
			if err != nil {
				v.tell(i18n.RecipeNotOpened, err.Error())
				return
			}
			if chosen == nil {
				// Dismissed, which is a person changing their mind.
				return
			}
			path := chosen.URI().Path()
			_ = chosen.Close()
			v.OpenFile(path)
		}, v.window)
		v.startIn(chooser)
		chooser.Show()
	}, func() {})
}

// Save writes the recipe on the screen into the file it came from, or asks
// where when it came from none. done hears whether it was saved, and may be
// nothing.
//
// A file somebody else changed on the disk since it was opened here is asked
// about before it is replaced, because the save would throw their change away.
func (v *View) Save(done func(saved bool)) {
	// A question answering with Save has already let go of itself, so a
	// question still standing here is somebody pressing the button behind it.
	if v.saving || v.question != nil {
		finish(done, false)
		return
	}
	if v.kept.Path == "" {
		v.chooseToSave(done)
		return
	}
	v.saveInto(v.kept.Path, false, done)
}

// SaveTo writes the recipe on the screen into this file and makes it the file
// of the screen, which is what choosing a name in Save as does. Exported for a
// test, which reaches the state without the dialog.
func (v *View) SaveTo(path string) { v.saveInto(path, true, nil) }

// chooseToSave asks for a file to save into, then saves.
func (v *View) chooseToSave(done func(saved bool)) {
	if v.window == nil || v.saving {
		finish(done, false)
		return
	}
	chooser := dialog.NewFileSave(func(into fyne.URIWriteCloser, err error) {
		if err != nil {
			v.tell(i18n.RecipeNotSaved, err.Error())
			finish(done, false)
			return
		}
		if into == nil {
			finish(done, false)
			return
		}
		// The toolkit has already made the file, empty, to hand it over.
		// Nothing is written through it: the save goes through a file of its
		// own and one rename, as every save here does.
		path := into.URI().Path()
		_ = into.Close()
		v.saveInto(path, true, done)
	}, v.window)
	name := v.text.T(i18n.RecipeCommandFile)
	if v.kept.Path != "" {
		name = filepath.Base(v.kept.Path)
	}
	chooser.SetFileName(name)
	v.startIn(chooser)
	chooser.Show()
}

// startIn opens a chooser in the folder of the file on the screen, when there
// is one, so that the next file is looked for beside the last.
func (v *View) startIn(chooser *dialog.FileDialog) {
	if v.kept.Path == "" {
		return
	}
	if folder, err := storage.ListerForURI(storage.NewFileURI(filepath.Dir(v.kept.Path))); err == nil {
		chooser.SetLocation(folder)
	}
}

// saveInto saves the recipe on the screen into a file, off the thread that draws,
// holding the window open until it has finished.
//
// chosen says the person picked this file in the save dialog just now, and
// agreed there to replace it if it was there, so it is not asked about again.
func (v *View) saveInto(path string, chosen bool, done func(saved bool)) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		absolute = path
	}
	text, form, stamp := v.editor.Text, v.kept.Form, v.kept.Stamp
	samePlace := absolute == v.kept.Path
	v.saving, v.savingText = true, text
	v.drawFile()
	letGo := v.hold()
	v.work.Read(func() func() {
		if !chosen && samePlace && recipefile.Changed(absolute, stamp) {
			return func() {
				v.saved(letGo)
				v.askToReplace(absolute, done)
			}
		}
		written, err := recipefile.Write(absolute, text, form)
		return func() {
			v.saved(letGo)
			if err != nil {
				v.tell(i18n.RecipeNotSaved, fault.Says(err))
				finish(done, false)
				return
			}
			v.kept = recipefile.Opened{Path: absolute, Text: text, Form: form, Stamp: written}
			v.drawFile()
			v.drawCommand()
			finish(done, true)
		}
	}, func(err error) func() {
		return func() {
			v.saved(letGo)
			v.tell(i18n.RecipeNotSaved, fault.Says(err))
			finish(done, false)
		}
	})
}

// saved ends a save on the screen and lets go of the window held for it.
func (v *View) saved(letGo func()) {
	v.saving, v.savingText = false, ""
	if letGo != nil {
		letGo()
	}
	v.drawFile()
}

// askToReplace asks whether a file changed on the disk behind the window may be
// replaced by the recipe on the screen.
func (v *View) askToReplace(path string, done func(saved bool)) {
	v.ask(i18n.RecipeChangedTitle, filepath.Base(path)+" "+v.text.T(i18n.RecipeChangedOnDisk),
		parts.Reply{Words: v.text.T(i18n.ToolkitCancel), Do: func() { finish(done, false) }},
		parts.Reply{Words: v.text.T(i18n.RecipeReplaceFile), Do: func() { v.saveInto(path, true, done) }})
}

// keepFirst goes on at once when nothing on the screen would be lost, and asks
// first when something would: save it, let it go, or stay. It is what the cross
// of the window asks through, and what opening another file asks before it
// takes the field.
//
// stay is called when the person keeps the recipe, by cancelling or because
// the save they chose did not happen, so the window knows not to go.
func (v *View) keepFirst(goOn, stay func()) {
	if !v.unsaved() {
		goOn()
		return
	}
	sentence := v.text.T(i18n.RecipeKeepUntitled)
	if v.kept.Path != "" {
		sentence = filepath.Base(v.kept.Path) + " " + v.text.T(i18n.RecipeKeepChanged)
	}
	v.ask(i18n.RecipeKeepTitle, sentence,
		parts.Reply{Words: v.text.T(i18n.ToolkitCancel), Do: stay},
		parts.Reply{Words: v.text.T(i18n.RecipeKeepDiscard), Do: goOn},
		parts.Reply{Words: v.text.T(i18n.RecipeSave), Do: func() {
			v.Save(func(saved bool) {
				if saved {
					goOn()
					return
				}
				stay()
			})
		}})
}

// tell says that something did not happen, and why, with one answer.
func (v *View) tell(title i18n.Key, why string) {
	v.ask(title, why, parts.Reply{Words: v.text.T(i18n.ToolkitOK)})
}

// ask puts a question over the window, or keeps it for a test to answer when
// there is no window. Each answer lets go of the question before it is carried
// out, because an answer may ask a question of its own.
func (v *View) ask(title i18n.Key, sentence string, replies ...parts.Reply) {
	for at, reply := range replies {
		do := reply.Do
		replies[at].Do = func() {
			v.question = nil
			if do != nil {
				do()
			}
		}
	}
	v.question = parts.NewQuestion(sentence, replies...)
	if v.window != nil {
		v.question.Show(v.window, v.text.T(title))
	}
}

// focusEditor puts the keyboard in the field, after starting from the example
// and after opening a file, because editing what has just arrived is what
// comes next. The toolkit clears focus on a tap of its own accord, so this is
// not taking the keyboard from somebody, it is giving it somewhere to be.
func (v *View) focusEditor() {
	// No application means nothing is running this screen, which happens to a
	// screen built to be read rather than shown. Asking the driver for a canvas
	// then would end the process inside a courtesy.
	app := fyne.CurrentApp()
	if app == nil {
		return
	}
	if canvas := app.Driver().CanvasForObject(v.editor); canvas != nil {
		canvas.Focus(v.editor)
	}
}

// finish tells whoever waits on a save how it ended, when anybody does.
func finish(done func(saved bool), saved bool) {
	if done != nil {
		done(saved)
	}
}
