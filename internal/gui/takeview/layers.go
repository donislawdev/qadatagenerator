package takeview

import (
	"fmt"

	"github.com/donislawdev/qadatagenerator/internal/adapter/i18n"
	"github.com/donislawdev/qadatagenerator/internal/take"
)

// nameOfLayer is what each kind of damage is called on screen.
//
// # Why this exists at all
//
// Until 2026-09-08 the tick boxes were labelled L0, L1, L2 and so on, which is
// how the catalogue files them and tells a person opening this window nothing.
// The handbook says outright that an internal term never reaches a label, and
// docs/08 describes this very screen as a person choosing "field format" and
// "value meaning". The code and the document disagreed, and the code was the
// one that was wrong.
//
// # Why the code stays beside the name
//
// It is what --layers takes in a terminal and what the manifest reports back,
// so dropping it would cut the window off from both. A person who reads the
// manifest afterwards has to be able to match what they ticked to what it
// says.
//
// 🔴 This map is CLOSED against the catalogue by a test. A kind of damage with
// no name here would otherwise fall back to its bare code and look, on screen,
// exactly like the thing this map was written to remove.
var nameOfLayer = map[string]i18n.Key{
	"L0":  i18n.TakeLayerBytes,
	"L1":  i18n.TakeLayerStructure,
	"L2":  i18n.TakeLayerFormat,
	"L3":  i18n.TakeLayerMeaning,
	"L4":  i18n.TakeLayerBetween,
	"L5":  i18n.TakeLayerAcross,
	"UNI": i18n.TakeLayerUnicode,
	"REV": i18n.TakeLayerUnusual,
}

// nameOfPerRow is how many defects one record may carry, said in words.
//
// Same reason as the map above and the same closing test: the engine takes
// exactly_one and any, and a list offering those two spellings asks somebody
// to read a settings file in order to use a window.
var nameOfPerRow = map[string]i18n.Key{
	"exactly_one": i18n.TakePerRowExactlyOne,
	"any":         i18n.TakePerRowAny,
}

// labelOfPerRow is one such setting named, or the bare value if nothing names
// it, for the reason given at labelOfLayer.
func (v *View) labelOfPerRow(value string) string {
	key, named := nameOfPerRow[value]
	if !named {
		return value
	}
	return v.text.T(key)
}

// perRowLabels is every such setting said the way a person reads it.
func (v *View) perRowLabels() []string {
	values := take.PerRowChoices()
	labels := make([]string, 0, len(values))
	for _, value := range values {
		labels = append(labels, v.labelOfPerRow(value))
	}
	return labels
}

// valueOfPerRowLabel turns what is picked back into what the engine is asked
// for.
func (v *View) valueOfPerRowLabel(label string) string {
	for _, value := range take.PerRowChoices() {
		if v.labelOfPerRow(value) == label {
			return value
		}
	}
	return label
}

// layerCodes is every kind of damage this build can apply to a file, in the
// order the catalogue gives them.
func (v *View) layerCodes() []string { return take.LayersIn(v.described) }

// layerLabels is the same list said the way a person reads it.
func (v *View) layerLabels() []string {
	codes := v.layerCodes()
	labels := make([]string, 0, len(codes))
	for _, code := range codes {
		labels = append(labels, v.labelOfLayer(code))
	}
	return labels
}

// labelOfLayer is one kind of damage named, with its code beside it.
//
// A code with no name falls back to the bare code rather than to an empty tick
// box. It is the wrong label and it is still a label, which is the better of
// the two failures a person can be handed.
func (v *View) labelOfLayer(code string) string {
	key, named := nameOfLayer[code]
	if !named {
		return code
	}
	return fmt.Sprintf(v.text.T(i18n.TakeLayerCode), v.text.T(key), code)
}

// codesOfLabels turns what is ticked back into what the engine is asked for.
func (v *View) codesOfLabels(labels []string) []string {
	byLabel := make(map[string]string, len(nameOfLayer))
	for _, code := range v.layerCodes() {
		byLabel[v.labelOfLayer(code)] = code
	}
	codes := make([]string, 0, len(labels))
	for _, label := range labels {
		if code, known := byLabel[label]; known {
			codes = append(codes, code)
		}
	}
	return codes
}

// labelsOfCodes is the other direction, for something outside this package
// asking for kinds by the name the catalogue uses.
func (v *View) labelsOfCodes(codes []string) []string {
	labels := make([]string, 0, len(codes))
	for _, code := range codes {
		labels = append(labels, v.labelOfLayer(code))
	}
	return labels
}
