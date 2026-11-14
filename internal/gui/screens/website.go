package screens

import "fyne.io/fyne/v2"

// Website is the window as the project website shows it, one state for each
// screen a person works on, drawn from what the README shows.
//
// # Why these are not on the list of All
//
// Every state of All is drawn from the sample, which carries Polish data and a
// manifest built for the screen rather than written by a run, because those
// states are there to be compared, not shown. The site shows the window to
// somebody deciding whether to use it, so its pictures come from the README
// instead: the recipe it saves as shop.yaml, a file of customers made from
// that recipe, the manifest a run over that file writes, and the record of what
// an application did that the README prints. Only a test can produce that data,
// because producing it means running the program, so the data arrives as Data
// and the recipe arrives here as text.
//
// # What holds the pictures to the window
//
// TestTheWebsiteShowsTheWindowAsItIs draws each state and compares it with the
// picture the site carries, its texts on every system and its pixels where the
// pictures are drawn. A change to a screen or to what the program produces turns
// it red, and -update draws the pictures again together with the stored trees.
func Website(recipe string) []State {
	return []State{
		{
			Name:    "quick-website",
			Purpose: "ten vehicle identification numbers from the seed the README draws its examples with",
			Screen:  screenQuick,
			Quick:   Quick{Generator: "vin", Seed: websiteSeed},
		},
		{
			Name:    "recipe-website",
			Purpose: "the recipe the README saves as shop.yaml, with its policy, judged and previewed",
			Screen:  screenRecipe,
			Recipe:  recipe,
		},
		{
			Name: "take-website",
			Purpose: "a file of customers made from that recipe, read, with the kinds of damage " +
				"the README orders for a file of your own chosen",
			Screen:      screenTake,
			NeedsSource: true,
			Layers:      websiteLayers(),
		},
		{
			Name: "browser-website",
			Purpose: "the manifest a run over that file wrote, beside the record the README prints, " +
				"with the entry the README talks about read out in full",
			Screen:        screenManifest,
			NeedsManifest: true,
			NeedsResponse: true,
			// Into the table the way browser-focus-cell goes, then down to the
			// fourth entry, which is row 135: the card number with two digits
			// swapped that the application took.
			Keyboard: Keyboard{
				Tabs: 8,
				Keys: []fyne.KeyName{fyne.KeyDown, fyne.KeyDown, fyne.KeyDown},
			},
		},
	}
}

// websiteSeed is the seed every example of the README is drawn with.
const websiteSeed = "readme"

// websiteLayers are the kinds of damage the README orders for a file of your
// own, in the order its command names them. The test that makes the manifest
// for these pictures orders the same ones, so the screen and the run agree.
func websiteLayers() []string { return []string{"L2", "L3", "UNI", "REV"} }
