package css //nolint:testpackage // uses the package's unexported test helpers (el, page, sheet, styleOf)

import (
	"testing"

	"github.com/merlindorin/ember/ui/internal/dom"
	"github.com/stretchr/testify/assert"
)

func TestFontSmoothingNoneMakesPixelTextInheritedAndAutoUndoesIt(t *testing.T) {
	inner, plain := el(dom.Span, "inner"), el(dom.Span, "plain")
	outer := with(el(dom.Div, "outer"), inner, plain)
	Compute(page(outer),
		[]*Sheet{sheet(t, "#outer { -webkit-font-smoothing: none } #plain { -webkit-font-smoothing: auto }")})
	assert.True(t, styleOf(outer).PixelText)
	assert.True(t, styleOf(inner).PixelText, "inherited")
	assert.False(t, styleOf(plain).PixelText, "auto smooths again")
	_, warns := Parse("t.css", "p { -webkit-font-smoothing: blurry }")
	assert.NotEmpty(t, warns, "only its keywords")
}
