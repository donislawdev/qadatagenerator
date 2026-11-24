package outward

import "net/url"

// Nowhere is a desktop that opens nothing.
//
// It is for a picture of the window and for a test, where a button that really
// opened a browser would change something outside the picture or the test. The
// window proper hands in the toolkit's application instead.
type Nowhere struct{}

// OpenURL opens nothing and reports nothing wrong.
func (Nowhere) OpenURL(*url.URL) error { return nil }

// ShowFolder shows nothing and reports nothing wrong, for the same callers and
// the same reason as OpenURL.
func (Nowhere) ShowFolder(string) error { return nil }
