// Command qadg-gui is the window half of the product.
//
// It is a second binary rather than a verb of the first one, and the reason is
// mechanical rather than stylistic. The graphical toolkit needs the C
// toolchain, so a command line binary that could reach this code would stop
// cross compiling from one machine, which is how it is built for three systems
// today. A guard checks that edge transitively, because an import through
// three other packages pulls in exactly the same toolchain and is harder to
// see in a review.
//
// This file stays short for the same reason cmd/qadg does. Everything worth
// testing lives in internal/gui, where a screen can be rendered without a
// display. What is left here is the one thing a test cannot do: opening a real
// window, and saying which reader each screen gets.
package main

import (
	"fmt"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
	"github.com/donislawdev/qadatagenerator/internal/adapter/responsejson"
	"github.com/donislawdev/qadatagenerator/internal/fault"
	"github.com/donislawdev/qadatagenerator/internal/gui/about"
	"github.com/donislawdev/qadatagenerator/internal/gui/assets"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/gui/fynetheme"
	"github.com/donislawdev/qadatagenerator/internal/gui/manifestview"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	"github.com/donislawdev/qadatagenerator/internal/gui/quickview"
	"github.com/donislawdev/qadatagenerator/internal/gui/recipeview"
	"github.com/donislawdev/qadatagenerator/internal/gui/shell"
	"github.com/donislawdev/qadatagenerator/internal/gui/takeview"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
	"github.com/donislawdev/qadatagenerator/internal/license"
	"github.com/donislawdev/qadatagenerator/internal/take"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

func main() {
	// The last catch, for the one thread the goroutines cannot answer for.
	//
	// Everything this window starts elsewhere turns a panic into a failure a
	// screen draws, in internal/gui/background. This is the thread that draws,
	// where the toolkit calls back into our widgets, and a panic there arrives
	// here with the window already gone.
	//
	// What it buys is the two things a crash owes somebody. A line on the error
	// channel saying what happened, in the words the command line uses for the
	// same event, instead of a stack trace. And an exit code of 1 rather than
	// Go's own 2, which is the number this tool gives a wrong invocation, so
	// whatever started this window is not told it typed something wrong.
	defer func() {
		if recovered := recover(); recovered != nil {
			fmt.Fprintln(os.Stderr, fault.FromPanic(recovered))
			fmt.Fprintln(os.Stderr, fault.CrashedDo)
			os.Exit(1)
		}
	}()

	application := app.New()
	application.Settings().SetTheme(fynetheme.Current())

	text := i18n.English
	window := application.NewWindow(text.T(i18n.WindowTitle))

	frame := assemble(text, window, application.Clipboard())

	// Dropping is a property of the window rather than of a control, so the
	// frame is what receives it and hands it to whichever screen is open.
	window.SetOnDropped(frame.Dropped)

	// 🔴 THE ONE WAY IN THAT docs/08 NAMES, AND THE LIMIT IT ARRIVES WITH.
	// Scenario one describes a keyboard shortcut opening the palette, and
	// DEC-202 left it out because nobody had measured whether it collides with
	// the text field of the recipe editor. Measured on 2026-09-08, in the
	// toolkit's own driver at internal/driver/glfw/window.go:865: if the
	// control holding focus is a Shortcutable, the driver hands it the shortcut
	// and RETURNS, so canvas.TypedShortcut is never reached. An entry is a
	// Shortcutable and its handler silently ignores a name it does not know.
	//
	// So this works everywhere except while somebody is typing in a field, and
	// that is written here rather than discovered. Making it work there too
	// means overriding TypedShortcut on our own entry and forwarding the names
	// the toolkit does not handle, which needs a copy of the toolkit's own
	// register of shortcut names. A second copy of a register is the thing this
	// project refuses, so the limit stands until somebody decides that trade.
	window.Canvas().AddShortcut(
		&desktop.CustomShortcut{KeyName: fyne.KeyK, Modifier: fyne.KeyModifierControl},
		func(fyne.Shortcut) { frame.Reveal(text.T(i18n.ScreenQuick)) },
	)
	window.SetContent(frame.Content())
	// A window with no size of its own shrinks to what its content demands, and
	// a scrolling container demands almost nothing. Running the first build
	// showed one heading in a window 128 pixels tall while every test was
	// green, because the tests read the content rather than how much of it a
	// person can see.
	window.Resize(fyne.NewSize(palette.StartWidth, palette.StartHeight))
	window.ShowAndRun()
}

// saveReport is how the About screen puts the machine readable report on disk.
//
// It lives here rather than on that screen for the reason every reader in this
// file does: a chooser needs a window to appear over, and a screen that reached
// for one could not be rendered without one. The screen is handed this and
// knows nothing about dialogs.
//
// A failure is shown rather than swallowed. Somebody who asked for a file and
// got neither a file nor a sentence has no way to tell a refusal from a program
// that stopped answering, and that silence is the one this product refuses
// everywhere else.
func saveReport(window fyne.Window) about.Saver {
	return func(contents []byte) {
		dialog.ShowFileSave(func(into fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, window)
				return
			}
			if into == nil {
				return
			}
			defer into.Close()
			if _, err := into.Write(contents); err != nil {
				dialog.ShowError(err, window)
			}
		}, window)
	}
}

// assemble builds every screen and the frame around them.
//
// It is apart from main because main had grown past the limit on the length of
// a function, and the limit is bent for nothing. What came out is the half that
// answers a different question: main opens a window and shows it, this decides
// what is in it.
//
// Every screen is handed its readers and its places of work here rather than
// reaching for them, which is what lets each of them be built in a test with no
// disk and no goroutine.
func assemble(text i18n.Catalogue, window fyne.Window, clipboard quickview.Clipboard) *shell.Shell {
	browser := manifestview.New(text, window, manifestview.Readers{
		Manifest: manifestjson.Load,
		Response: responsejson.Load,
	}, background.ReadInBackground)
	// The catalog of damage is data compiled into this binary, so a failure to
	// read it means the binary itself is not what it was built as. There is
	// nothing a person could do about it from inside a window and nothing this
	// window could honestly show, so it says so and stops.
	described, err := catalogyaml.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "this build cannot read its own catalog of damage:", err)
		os.Exit(1)
	}
	damage := takeview.New(text, window, described, take.Open, background.Work{
		Run:   background.InBackground,
		Read:  background.ReadInBackground,
		Watch: background.WatchForClosing,
	}, clipboard)
	// Two places of work rather than one. A recipe is allowed to be 256 KB and
	// parsing one inside that limit can cost hundreds of megabytes, so judging
	// what is typed belongs anywhere except the thread that draws. Writing the
	// dataset belongs there even less: it is the one thing on this screen that
	// can take minutes.
	editor := recipeview.New(text, recipeview.Checker(described), recipeview.Writer(),
		background.Work{
			Read:    background.ReadInBackground,
			Produce: background.ProduceInBackground,
		}, window)

	// The one screen that reaches outside this tool's own files and channels,
	// and the way out is handed in rather than reached for, like every other
	// screen here. A screen built to be read rather than shown has no
	// application behind it, and the clipboard of the toolkit asks for one.
	value := quickview.New(text, clipboard)

	// # Why this order, which changed on 2026-09-08
	//
	// The editor was first until then, and the reason was recorded here: it was
	// the only screen that could generate anything, and the screen a window
	// opens on is the one it is judged by. That reason expired when quick was
	// built on 2026-09-06, and nobody moved the list.
	//
	// The order now follows the numbering in docs/08, which is not decoration:
	// that document calls the first scenario "the path that builds adoption"
	// and the second "the shortest way to the first wow". Both are the manual
	// tester, who is the person this window is hardest for. They stood fifth
	// and second, so somebody opening this window for the first time landed on
	// raw YAML. Reading a screen is easier than writing one, so the screens
	// that read come after the screens that make, and About goes last because
	// it answers a question nobody has while working.
	making, checking := text.T(i18n.SidebarGroupMake), text.T(i18n.SidebarGroupCheck)
	// 🔴 The icons are here because they were NOT, from the day the sidebar
	// gained them until 2026-09-09, and nothing said so.
	//
	// This list exists twice. The other copy is in internal/gui/screens, which
	// is what the probe that renders every screen state builds from, and it has
	// carried an icon per screen since the day they were drawn. This one had
	// none. So every render anybody looked at showed a column of five names
	// with an icon each, and the program showed five names with a gap where the
	// icon would be - which is worse than no gap at all, because the space is
	// reserved by the same layout either way.
	//
	// It survived because the fault is an ABSENCE on a surface nobody
	// photographs: the tests read the probe's copy, the renders come from the
	// probe's copy, and the only way to see this one is to run the binary and
	// look. The owner did, and asked why the program looks worse than the
	// drawing. This was most of the answer.
	//
	// TestEverySidebarScreenCarriesAnIcon reads both copies out of the source
	// and refuses either one going bare, so the register is closed in the only
	// way two lists in two packages can be.
	return shell.New([]shell.Screen{
		{Name: text.T(i18n.ScreenQuick), Group: making, Icon: paint.IconBolt,
			Content: value.Content()},
		{Name: text.T(i18n.ScreenTake), Group: making, Icon: paint.IconBreak,
			Content: damage.Content(), OnDropped: damage.Dropped},
		{Name: text.T(i18n.ScreenRecipe), Group: making, Icon: paint.IconRecipe,
			Content: editor.Content()},
		{Name: text.T(i18n.ScreenManifest), Group: checking, Icon: paint.IconChecklist,
			Content: browser.Content()},
		{Name: text.T(i18n.ScreenAbout), Icon: paint.IconInfo,
			Content: about.Content(text, license.Current(), version.Current(),
				saveReport(window), assets.Notice)},
	}, shell.Brand{
		// The same name the window carries in its title bar, from the same key,
		// so the two can never say different things.
		Name: text.T(i18n.WindowTitle),
		Mark: paint.IconBolt,
		// The tool's own version and not the catalog's or the manifest's. This
		// window has four version numbers behind it and the About screen names
		// them all, which is the right place for that: here there is room for
		// one, and the one somebody means by "which version is this" is the
		// program's.
		Version: "v" + version.Current().Tool,
		Promise: text.T(i18n.SidebarOffline),
	})
}
