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

	"github.com/donislawdev/qadatagenerator/internal/adapter/catalogyaml"
	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/adapter/manifestjson"
	"github.com/donislawdev/qadatagenerator/internal/adapter/responsejson"
	"github.com/donislawdev/qadatagenerator/internal/fault"
	"github.com/donislawdev/qadatagenerator/internal/gui/about"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/gui/fynetheme"
	"github.com/donislawdev/qadatagenerator/internal/gui/manifestview"
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
	application.Settings().SetTheme(fynetheme.Dark{})

	text := i18n.English
	window := application.NewWindow(text.T(i18n.WindowTitle))

	frame := assemble(text, window)

	// Dropping is a property of the window rather than of a control, so the
	// frame is what receives it and hands it to whichever screen is open.
	window.SetOnDropped(frame.Dropped)
	window.SetContent(frame.Content())
	// A window with no size of its own shrinks to what its content demands, and
	// a scrolling container demands almost nothing. Running the first build
	// showed one heading in a window 128 pixels tall while every test was
	// green, because the tests read the content rather than how much of it a
	// person can see.
	window.Resize(fyne.NewSize(palette.StartWidth, palette.StartHeight))
	window.ShowAndRun()
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
func assemble(text i18n.Catalogue, window fyne.Window) *shell.Shell {
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
	})
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

	// The editor is first, and that is the answer to why it was built: somebody
	// who opened this window before it could generate nothing with it, and the
	// screen a window opens on is the one it is judged by.
	return shell.New([]shell.Screen{
		{Name: text.T(i18n.ScreenRecipe), Content: editor.Content()},
		{Name: text.T(i18n.ScreenTake), Content: damage.Content(), OnDropped: damage.Dropped},
		{Name: text.T(i18n.ScreenManifest), Content: browser.Content()},
		{Name: text.T(i18n.ScreenAbout), Content: about.Content(text, license.Current(), version.Current())},
	})
}
