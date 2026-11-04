package main

import (
	"strings"

	"github.com/merlindorin/ember/color"
	"github.com/merlindorin/ember/twod"
)

// Chat shows the messages sent, the last few, in both faces.
type Chat struct {
	lines        []string
	mono, colour *twod.Text
}

func newChat(mono, colour *twod.Face) *Chat {
	c := &Chat{lines: greeting()}
	c.mono = twod.NewText(
		"",
		twod.Typeface(mono),
		twod.FontSize(22),
		twod.Color(color.Hex("#f0b44c")),
		twod.At(20, 44),
		twod.Pivot(0, 0),
		twod.ExpandShortcodes(),
	)
	c.colour = twod.NewText(
		"",
		twod.Typeface(colour),
		twod.FontSize(22),
		twod.At(20, 214),
		twod.Pivot(0, 0),
		twod.ExpandShortcodes(),
	)
	c.show()
	return c
}

// Send adds a message, keeping the last five; a blank one is dropped.
func (c *Chat) Send(text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	c.lines = append(c.lines, text)
	if len(c.lines) > 5 {
		c.lines = c.lines[len(c.lines)-5:]
	}
	c.show()
}

func (c *Chat) show() {
	text := strings.Join(c.lines, "\n")
	c.mono.SetText(text)
	c.colour.SetText(text)
}
