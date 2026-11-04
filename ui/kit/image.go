package kit

import (
	"fmt"

	"github.com/merlindorin/ember/ui"
)

// Image is an img of class "kit-image" showing src, an image
// of the Document's stylesheets' files, as large as the picture unless its
// layout stretches it. Its class picks the old ember/ui's stretch mode: none (as
// "scale") fills its box; "kit-keep" and "kit-keep-centered" keep the picture's
// size at the top left or the centre; "kit-keep-aspect" and
// "kit-keep-aspect-centered" fit it inside, keeping its shape; and
// "kit-keep-aspect-covered" covers the box with it, cut to it. Tiling is
// TileImage.
func Image(src string, args ...ui.Arg) *ui.Node {
	return ui.Img(append([]ui.Arg{ui.Class("kit-image"), ui.Attr("src", src)}, args...)...)
}

// TileImage is the old ember/ui's Image in its tile mode: a div of class
// "kit-tile-image" repeating src across its box. Give it a size: tiles have
// none of their own.
func TileImage(src string, args ...ui.Arg) *ui.Node {
	return ui.Div(append([]ui.Arg{ui.Class("kit-tile-image"), ui.Style(fmt.Sprintf("background-image: url(%q)", src))},
		args...)...)
}
