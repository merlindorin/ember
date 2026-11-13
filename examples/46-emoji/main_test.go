package main

import (
	"strings"
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/ui/align"
)

func TestBothFacesShowWhatIsSent(t *testing.T) {
	mono, colour, err := faces()
	if err != nil {
		t.Fatal(err)
	}
	if colour.Emoji().Len() != 108 || len(mono.Fallbacks()) != 1 || mono.Emoji() != nil {
		t.Fatalf("Noto behind both, 108 Twemoji pictures in colour only: %d", colour.Emoji().Len())
	}
	for _, line := range greeting() {
		for _, e := range []string{"👨‍👩‍👧", "🇫🇷", "👍🏽", "1️⃣"} {
			if strings.Contains(line, e) && !colour.Emoji().Has(e) {
				t.Fatalf("the greeting's %s has a picture", e)
			}
		}
	}
	chat := newChat(mono, colour)
	doc := inputPage(colour.Emoji(), chat)
	win := platform.NewWindow(platform.Size(640, 420))
	win.Add(chat.mono, chat.colour, doc)
	auto := automation.New(automation.Recording())
	w := ember.New(ember.Automation(auto)).AddEntities(win)
	w.RunFrames(2)
	auto.Queue("message", automation.ActSet, "gg :tada:🇯🇵")
	auto.Queue("message", automation.ActSubmit, nil)
	w.RunFrames(3)
	if !strings.HasSuffix(chat.colour.Text(), "\ngg :tada:🇯🇵") || chat.mono.Text() != chat.colour.Text() ||
		doc.Value(doc.ByID("message")) != "" {
		t.Fatalf(
			"Enter sends to both, and empties the input: %q, left %q",
			chat.colour.Text(),
			doc.Value(doc.ByID("message")),
		)
	}
	if faults := align.Check(doc); len(faults) > 0 || len(doc.Warnings()) > 0 {
		t.Errorf("the input lines up and its sheet reads cleanly: %v %v", faults, doc.Warnings())
	}
	if colour.Measure("🎉", 22).X != 22 {
		t.Fatal("a colour emoji is a square at the text's size")
	}
}
