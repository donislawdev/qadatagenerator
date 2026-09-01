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
	"github.com/donislawdev/qadatagenerator/internal/gui/about"
	"github.com/donislawdev/qadatagenerator/internal/gui/background"
	"github.com/donislawdev/qadatagenerator/internal/gui/fynetheme"
	"github.com/donislawdev/qadatagenerator/internal/gui/manifestview"
	"github.com/donislawdev/qadatagenerator/internal/gui/shell"
	"github.com/donislawdev/qadatagenerator/internal/gui/takeview"
	palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"
	"github.com/donislawdev/qadatagenerator/internal/license"
	"github.com/donislawdev/qadatagenerator/internal/take"
	"github.com/donislawdev/qadatagenerator/internal/version"
)

func main() {
	application := app.New()
	application.Settings().SetTheme(fynetheme.Dark{})

	text := i18n.English
	window := application.NewWindow(text.T(i18n.WindowTitle))

	// The readers are handed to the screen here rather than reached for inside
	// it, which is what lets every state of that screen be built in a test
	// against documents that never touch a disk.
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
	// The reader and the three places work happens are all handed in, which is
	// what lets every state of that screen be reached in a test without a disk
	// and without a goroutine.
	damage := takeview.New(text, window, described, take.Open, background.Work{
		Run:   background.InBackground,
		Read:  background.ReadInBackground,
		Watch: background.WatchForClosing,
	})

	frame := shell.New([]shell.Screen{
		{Name: text.T(i18n.ScreenTake), Content: damage.Content(), OnDropped: damage.Dropped},
		{Name: text.T(i18n.ScreenManifest), Content: browser.Content()},
		{Name: text.T(i18n.ScreenAbout), Content: about.Content(text, license.Current(), version.Current())},
	})

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
