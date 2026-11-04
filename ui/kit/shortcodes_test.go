package kit_test

import (
	"bytes"
	"image"
	"image/png"
	"testing"
	"testing/fstest"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/twod"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// showWith is show for a Document of kit's theme and opts.
func showWith(t *testing.T, opts []ui.DocumentOption, nodes ...*ui.Node) *shown {
	t.Helper()
	s := &shown{doc: ui.NewDocument(append([]ui.DocumentOption{kit.Theme()}, opts...)...)}
	s.doc.Body().Append(nodes...)
	win := platform.NewWindow(platform.Size(400, 300))
	win.Add(s.doc)
	s.auto = automation.New(automation.Recording())
	s.w = ember.New(ember.Automation(s.auto)).AddEntities(win)
	s.in = win.Input()
	s.frames(2)
	require.Empty(t, s.doc.Warnings())
	return s
}

func fireAndFish(t *testing.T) *twod.EmojiSet {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewNRGBA(image.Rect(0, 0, 4, 4))))
	set, err := twod.LoadEmoji(fstest.MapFS{"emoji/1f525.png": {Data: b.Bytes()}, "emoji/1f41f.png": {Data: b.Bytes()}},
		"emoji")
	require.NoError(t, err)
	return set
}

func TestAShortcodeFieldSuggestsUnderItsInput(t *testing.T) {
	field := kit.ShortcodeField(kit.TextInput("", ui.Placeholder("Say")))
	in, list := field.Children()[0], kit.SuggestionList(field)
	s := showWith(t, []ui.DocumentOption{ui.Emoji(fireAndFish(t))}, field)
	s.click(in)
	for _, r := range "hi :fi" {
		s.in.AddText(r)
	}
	s.frames(2)
	require.Len(t, list.Children(), 2)
	assert.GreaterOrEqual(t, list.Box().Y, in.Box().Y+in.Box().H, "under the input")
	assert.Equal(t, automation.RoleSelect, s.element(t, "🔥 :fire:").Role)
	s.key(input.KeyEnter)
	assert.Equal(t, "hi 🔥", s.doc.Value(in))
	assert.Zero(t, list.Box().H, "closed once chosen")
}

func TestAShortcodeListsIDAndItsOptionsAreTheSameAtEveryBuild(t *testing.T) {
	ids := func() []string {
		field := kit.ShortcodeField(kit.TextInput("", ui.Placeholder("Say")))
		s := showWith(t, []ui.DocumentOption{ui.Emoji(fireAndFish(t))}, field)
		s.click(field.Children()[0])
		for _, r := range ":fi" {
			s.in.AddText(r)
		}
		s.frames(2)
		var out []string
		_, all := s.auto.Snapshot("", "")
		for _, e := range all {
			out = append(out, e.ID)
		}
		assert.Equal(t, "say-suggestions", kit.SuggestionList(field).ID(), "named after its field")
		assert.True(t, kit.SuggestionList(field).Children()[0].HasClass("ember-suggestion"),
			"the engine's own class, apart from a game's .suggestion")
		return out
	}
	first := ids()
	assert.NotEmpty(t, first)
	assert.Equal(t, first, ids(), "two builds, the same automation IDs")
}

func TestAShortcodeListOpensAboveItsFieldAtTheWindowsFoot(t *testing.T) {
	for _, c := range []struct {
		top   string
		above bool
	}{{"10px", false}, {"260px", true}} {
		field := kit.ShortcodeField(kit.TextInput("", ui.Placeholder("Say")))
		field.SetStyle("position: absolute; left: 10px; top: " + c.top)
		in, list := field.Children()[0], kit.SuggestionList(field)
		s := showWith(t, []ui.DocumentOption{ui.Emoji(fireAndFish(t))}, field)
		s.click(in)
		for _, r := range ":fi" {
			s.in.AddText(r)
		}
		s.frames(2)
		l, f := list.Box(), in.Box()
		require.NotZero(t, l.H, "open at %s", c.top)
		if c.above {
			assert.LessOrEqual(t, l.Y+l.H, f.Y, "at the foot, above the field")
		} else {
			assert.GreaterOrEqual(t, l.Y, f.Y+f.H, "at the top, under it")
		}
		assert.LessOrEqual(t, l.Y+l.H, float32(300), "inside the window")
	}
}
