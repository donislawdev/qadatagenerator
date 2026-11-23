//go:build !darwin

package parts

import palette "github.com/donislawdev/qadatagenerator/internal/gui/theme"

// wheelDistance is the distance a wheel turned by dy should move an area
// showing view units: the notches dy stands for, each the system's distance. A
// touchpad's fraction of a notch stays a fraction. The toolkit's step is
// palette.ToolkitWheelStep, which says where it comes from.
func wheelDistance(dy, view float32) float32 {
	return dy / palette.ToolkitWheelStep * notchDistance(view)
}
