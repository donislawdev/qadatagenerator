package values

import (
	"strings"

	"github.com/donislawdev/qadatagenerator/internal/app/generate"
)

// ShapeOf says what the values of a field are, for a table made to hold them
// (DEC-341).
//
// # Where each answer comes from
//
// The kind is the one the values are made as: the type a generator gives, or
// the type the recipe wrote, with a UUID told apart from the other identifiers
// because a database has a type of its own for it. A number is as wide as the
// longest one the field can hold: for a list, the longest on the list, and
// otherwise the longer end of the range the run draws from, which is the span
// the draw itself uses, so the table and the values cannot disagree about it.
//
// A field the plan refuses never reaches a table, and the answer for it is the
// widest number this build draws rather than a failure, so that this question
// never has to be asked in a particular order with the others.
func (s Source) ShapeOf(req generate.Request) generate.Shape {
	kind := req.Type
	if req.Generator != "" {
		kind = types[req.Generator]
	}
	switch {
	case req.Generator == UUID || req.Type == TypeUUID:
		return generate.Shape{Kind: generate.ShapeUUID}
	case kind == TypeDate:
		return generate.Shape{Kind: generate.ShapeDate}
	case kind == TypeInteger, kind == TypeDecimal:
		return numberShape(req, kind)
	}
	return generate.Shape{Kind: generate.ShapeText}
}

// numberShape is the shape of a field of whole numbers or decimals.
func numberShape(req generate.Request, kind string) generate.Shape {
	shape := generate.Shape{Kind: generate.ShapeInteger}
	if kind == TypeDecimal {
		shape.Kind, shape.Places = generate.ShapeDecimal, decimalPlaces
	}
	if req.Type != kind {
		// No generator gives numbers today, and a range belongs to a field
		// described by type alone, so a generator that does is answered with
		// the widest number rather than with the range of a field it is not.
		shape.Digits = widestDrawn - shape.Places
		return shape
	}
	if len(req.Values) > 0 {
		shape.Places = 0
		// A list is checked against its type before a run (Accepts), so every
		// entry is digits with at most one point.
		for _, value := range req.Values {
			whole, fraction, _ := strings.Cut(strings.TrimPrefix(value, "-"), ".")
			shape.Digits = max(shape.Digits, len(whole))
			if shape.Kind == generate.ShapeDecimal {
				shape.Places = max(shape.Places, len(fraction))
			}
		}
		return shape
	}
	drawn, err := spanOf(req)
	if err != nil {
		shape.Digits = widestDrawn - shape.Places
		return shape
	}
	for _, end := range []int64{drawn.low, drawn.high} {
		whole, _, _ := strings.Cut(strings.TrimPrefix(drawn.text(end), "-"), ".")
		shape.Digits = max(shape.Digits, len(whole))
	}
	return shape
}
