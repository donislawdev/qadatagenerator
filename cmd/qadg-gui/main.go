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
	"github.com/donislawdev/qadatagenerator/internal/gui/catalogue"
	"github.com/donislawdev/qadatagenerator/internal/gui/fynetheme"
	"github.com/donislawdev/qadatagenerator/internal/gui/manifestview"
	"github.com/donislawdev/qadatagenerator/internal/gui/outward"
	"github.com/donislawdev/qadatagenerator/internal/gui/paint"
	"github.com/donislawdev/qadatagenerator/internal/gui/quickview"
	"github.com/donislawdev/qadatagenerator/internal/gui/recipeview"
	"github.com/donislawdev/qadatagenerator/internal/gui/screens"
	"github.com/donislawdev/qadatagenerator/internal/gui/shell"
	"github.com/donislawdev/qadatagenerator/internal/gui/takeview"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
	"github.com/donislawdev/qadatagenerator/internal/license"
	"github.com/donislawdev/qadatagenerator/internal/take"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

func main() {
	defer lastCatch()

	// Asked before anything of the toolkit exists, so a refusal opens no
	// window at all rather than one that closes again.
	asked, err := arguments(os.Args[1:], os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(wrongInvocation)
	}
	if asked.help {
		return
	}

	application := app.New()
	application.Settings().SetTheme(fynetheme.Current())

	text := i18n.English
	// The toolkit's own words, in the language of this window rather than of
	// the desktop (B-290). A refusal leaves them in the desktop's language,
	// which is worth a line on the error channel and not worth no window.
	if err := fynetheme.Speak(text); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	window := application.NewWindow(text.T(i18n.WindowTitle))

	frame := frameFor(asked, text, window, application)

	// Dropping is a property of the window rather than of a control, so the
	// frame is what receives it and hands it to whichever screen is open.
	window.SetOnDropped(frame.Dropped)
	revealOnShortcut(window, frame, text)
	window.SetContent(frame.Content())
	// A window with no size of its own shrinks to what its content demands, and
	// a scrolling container demands almost nothing. Running the first build
	// showed one heading in a window 128 pixels tall while every test was
	// green, because the tests read the content rather than how much of it a
	// person can see.
	window.Resize(fyne.NewSize(palette.StartWidth, palette.StartHeight))
	// What the window no longer uses goes back once it is left alone, and
	// internal/gui/background/tidy.go says when and why. Stopped as the window
	// closes, while the toolkit still runs, and again once it has stopped, in
	// case it stopped some other way.
	stopTidying := background.TidyWhenLeftAlone(window.Canvas())
	window.SetOnClosed(stopTidying)
	window.ShowAndRun()
	stopTidying()
}

// lastCatch is the last catch, for the one thread the goroutines cannot answer
// for. It is deferred by main and calls recover itself, which is the only way
// recover stops a panic.
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
func lastCatch() {
	if recovered := recover(); recovered != nil {
		fmt.Fprintln(os.Stderr, fault.FromPanic(recovered))
		fmt.Fprintln(os.Stderr, fault.CrashedDo)
		os.Exit(1)
	}
}

// frameFor is the frame the command line asked for: the catalogue of controls
// with -catalogue, the window proper without it.
//
// Only one of the two is built. The window proper starts places of work and a
// watch on the signal that closes it, and none of that belongs behind a view
// of controls that does no work. The catalogue carries no links to pages
// outside this program either: it is a view of controls, and the control those
// links are drawn as is already on its page of buttons.
func frameFor(asked launch, text i18n.Catalogue, window fyne.Window, application fyne.App) *shell.Shell {
	if asked.catalogue {
		// The samples and the brand come from the package the stored pictures
		// take them from, so the window and the pictures of it say the same.
		return catalogue.Frame(text, screens.Brand(text), screens.CatalogueSamples())
	}
	return assemble(text, window, application.Clipboard(), application)
}

// revealOnShortcut opens the quick screen on Ctrl+K.
//
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
//
// In the catalogue of controls the frame has no screen of that name, and
// Reveal of a name it does not have does nothing.
func revealOnShortcut(window fyne.Window, frame *shell.Shell, text i18n.Catalogue) {
	window.Canvas().AddShortcut(
		&desktop.CustomShortcut{KeyName: fyne.KeyK, Modifier: fyne.KeyModifierControl},
		func(fyne.Shortcut) { frame.Reveal(text.T(i18n.ScreenQuick)) },
	)
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
			// The close is asked about as well as the write, because a file
			// that fails to close may not be on the disk at all, and a report
			// somebody saved and cannot find is the same silence.
			if _, err := into.Write(contents); err != nil {
				_ = into.Close()
				dialog.ShowError(err, window)
				return
			}
			if err := into.Close(); err != nil {
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
func assemble(text i18n.Catalogue, window fyne.Window, clipboard quickview.Clipboard,
	desktop outward.Desktop) *shell.Shell {
	// 🔴 One place of work for the whole window, handed to every screen as
	// the same value. Until 2026-10-27 each screen was given a literal of its
	// own, and the one for the recipe screen was written without the watch
	// for a signal, so a container stopped while it wrote a dataset left the
	// temporary files and the marker behind. A port one screen has and the
	// other does not can no longer be written here.
	//
	work := placesOfWork(window)
	browser := manifestview.New(text, window, manifestview.Readers{
		Manifest: manifestjson.Load,
		Response: responsejson.Load,
	}, work.Read, clipboard)
	// The catalog of damage is data compiled into this binary, so a failure to
	// read it means the binary itself is not what it was built as. There is
	// nothing a person could do about it from inside a window and nothing this
	// window could honestly show, so it says so and stops.
	described, err := catalogyaml.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "this build cannot read its own catalog of damage:", err)
		os.Exit(1)
	}
	damage := takeview.New(text, window, described, take.Open, work, clipboard)
	// The recipe screen uses two of the places of work. A recipe is allowed to
	// be 256 KB and parsing one inside that limit can cost hundreds of
	// megabytes, so judging what is typed belongs anywhere except the thread
	// that draws. Writing the dataset belongs there even less: it is the one
	// thing on this screen that can take minutes.
	editor := recipeview.New(text, recipeview.Checker(described), recipeview.Writer(),
		work, window)

	// The one screen that reaches outside this tool's own files and channels,
	// and the way out is handed in rather than reached for, like every other
	// screen here. A screen built to be read rather than shown has no
	// application behind it, and the clipboard of the toolkit asks for one.
	value := quickview.New(text, clipboard)

	// The two pages outside this program open through the application, off
	// the thread that draws, and a page the system did not open is said in a
	// dialog with its address, so a button that did nothing is never silent.
	refused := func(title, message string) { dialog.ShowInformation(title, message, window) }
	return shell.New(sidebar(text, window, value, damage, editor, browser),
		screens.WindowBrand(text, desktop, work.Read, refused))
}

// placesOfWork is where every screen sends what must not happen on the thread
// that draws, together with the one closing of the window.
//
// The closing is one for the window because both screens that write run on
// this window, and a run on one ending must not give the close button back
// while the other is still writing. It is apart from assemble so that a test
// can ask whether every port is there without building a screen.
func placesOfWork(window background.Window) background.Work {
	closing := background.NewClosing(window, background.WatchForClosing)
	return background.Work{
		Run:     background.InBackground,
		Produce: background.ProduceInBackground,
		Read:    background.ReadInBackground,
		Hold:    closing.Hold,
	}
}

// sidebar is the column of screens the frame shows, each with its name, its
// group and its icon.
//
// Apart from assemble because that function passed the limit on the length of
// a function. assemble decides which reader and which place of work each
// screen gets, and this decides where each one stands.
func sidebar(text i18n.Catalogue, window fyne.Window, value *quickview.View, damage *takeview.View,
	editor *recipeview.View, browser *manifestview.View) []shell.Screen {
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
	return []shell.Screen{
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
	}
}
