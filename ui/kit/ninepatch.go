package kit

import (
	"fmt"

	"github.com/merlindorin/ember/ui"
)

// NinePatch is a div of class "kit-nine-patch" drawing
// the image src stretched, its corners and edges kept at the margins
// left, top, right and bottom (in image pixels, one per point), its
// children inside them. It is CSS's border-image, the margins its border
// widths. src is an image of the Document's stylesheets' files.
//
// Its style attribute holds the patch, so style it further with classes.
func NinePatch(src string, left, top, right, bottom float32, args ...ui.Arg) *ui.Node {
	style := fmt.Sprintf("border-style: solid; border-color: transparent; border-width: %gpx %gpx %gpx %gpx; "+
		"border-image: url(%q) %g %g %g %g fill",
		top, right, bottom, left, src, top, right, bottom, left)
	return ui.Div(append([]ui.Arg{ui.Class("kit-nine-patch"), ui.Style(style)}, args...)...)
}
