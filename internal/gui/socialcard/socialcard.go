// Package socialcard draws the picture GitHub shows when a link to this
// repository is shared, and holds it to what the program says about itself.
//
// # Where it is and who puts it on GitHub
//
// The picture is .github/social-preview/social-preview.png, 1280 by 640 pixels,
// the size GitHub asks for. Beside it, says.txt lists every text drawn on it,
// one to a line. GitHub does not read the picture from the repository. It keeps
// a copy of its own in the repository settings (Settings, General, Social
// preview, Edit), so a new picture here changes nothing there until somebody
// uploads it, and GitHub shows it only while the repository is public.
//
// # What it is drawn from
//
// The parts of the program rather than artwork: the colours by the names the
// window uses (internal/gui/theme), the IBM Plex faces the window ships, the
// 64 pixel picture of the program icon from the icon file both executables
// carry, the identifiers this build generates, the number of kinds of damage it
// implements, three values its generators made, and three rows of a real run of
// the damage over a file of book numbers and surnames, read back from the
// manifest that run wrote.
//
// # When the program changes
//
// A new identifier, a new kind of damage, a new colour or a new word moves what
// the card would say or how it looks, and the test in this package fails until
// the picture is drawn again:
//
//	go test ./internal/gui/socialcard -run TestTheSocialPreviewSaysWhatTheProgramDoes -update
//
// Then look at the picture, commit it with says.txt, and upload it to GitHub
// again. The drawing is in the test file because it is a build step for a
// committed file, the way lentils.ico is, and the program itself never draws it.
package socialcard
