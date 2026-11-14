package emojinames_test

import (
	"testing"

	"github.com/lantervale/ember/internal/emojinames"
)

func TestShortcodesExpandAndUnknownOnesStay(t *testing.T) {
	for in, want := range map[string]string{
		"I :heart: it :fire:":          "I ❤️ it 🔥",
		":+1: and :thumbsup:":          "👍 and 👍",
		":cry:":                        "😢",
		"time 10:30, ratio 3:2":        "time 10:30, ratio 3:2",
		":nope_not_one: :fire:":        ":nope_not_one: 🔥",
		"::":                           "::",
		":wave::skin-tone-3:":          "👋🏼",
		":fire::skin-tone-3:":          "🔥:skin-tone-3:",
		"no colon at all":              "no colon at all",
		":fr: :jp:":                    "🇫🇷 🇯🇵",
		"a :smile:b:smile:":            "a 😄b😄",
		":family_man_woman_girl: home": "👨‍👩‍👧 home",
	} {
		if got := emojinames.Expand(in); got != want {
			t.Errorf("%q: %q, want %q", in, got, want)
		}
	}
}

func TestShortcodesMatchByPrefix(t *testing.T) {
	got := emojinames.Matching("fi", 6, nil)
	if len(got) != 6 || len(got[0].Name) > len(got[5].Name) {
		t.Fatalf("six names starting fi, shortest first: %v", got)
	}
	if fire := emojinames.Matching("fir", 1, nil); len(fire) != 1 || fire[0].Name != "fire" {
		t.Fatalf("fir brings fire first: %v", fire)
	}
	fire := func(e emojinames.Entry) bool { return e.Name == "fire" }
	if drawable := emojinames.Matching("fi", 6, fire); len(drawable) != 1 {
		t.Fatalf("keep filters: %v", drawable)
	}
	for _, e := range got {
		if e.Name[:2] != "fi" {
			t.Fatalf("%v", got)
		}
	}
	if len(emojinames.Matching("zzzz", 6, nil)) != 0 {
		t.Fatal("nothing starts zzzz")
	}
	if e, ok := emojinames.Lookup("thumbsup"); !ok || e != "👍" {
		t.Fatal("aliases")
	}
}
