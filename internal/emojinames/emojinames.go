// Package emojinames is the emoji shortcodes Slack and GitHub use, :fire:
// and :+1:, from GitHub's gemoji (MIT, Copyright 2019 GitHub, Inc., its
// text in LICENSE-gemoji), for twod's text and ui's inputs.
package emojinames

//go:generate go run ./gen -src https://raw.githubusercontent.com/github/gemoji/v4.1.0/db/emoji.json -out names_gen.go

import (
	"sort"
	"strings"
)

// Entry is one shortcode: its name without colons, its emoji, and whether
// the emoji takes a skin tone.
type Entry struct {
	Name  string
	Emoji string
	Tones bool
}

func find(name string) (Entry, bool) {
	i := sort.Search(len(names), func(i int) bool { return names[i].Name >= name })
	if i < len(names) && names[i].Name == name {
		return names[i], true
	}
	return Entry{}, false
}

// Lookup is the emoji of name, without colons.
func Lookup(name string) (string, bool) {
	e, ok := find(name)
	return e.Emoji, ok
}

// Matching lists up to most shortcodes starting with prefix that keep
// says to, shortest names first, so :fi brings :fire: before
// :field_hockey:. A nil keep keeps every one.
func Matching(prefix string, most int, keep func(Entry) bool) []Entry {
	i := sort.Search(len(names), func(i int) bool { return names[i].Name >= prefix })
	var out []Entry
	for ; i < len(names) && strings.HasPrefix(names[i].Name, prefix); i++ {
		if keep == nil || keep(names[i]) {
			out = append(out, names[i])
		}
	}
	sort.SliceStable(out, func(a, b int) bool { return len(out[a].Name) < len(out[b].Name) })
	if len(out) > most {
		out = out[:most]
	}
	return out
}

// tones are Slack's skin tones, skin-tone-2 the lightest to skin-tone-6.
var tones = map[string]rune{ //nolint:gochecknoglobals // a lookup table
	"skin-tone-2": 0x1f3fb,
	"skin-tone-3": 0x1f3fc,
	"skin-tone-4": 0x1f3fd,
	"skin-tone-5": 0x1f3fe,
	"skin-tone-6": 0x1f3ff,
}

// Tone is emoji with the skin tone name, such as skin-tone-3, when the
// emoji takes one.
func Tone(emoji, name string) (string, bool) {
	tone, ok := tones[name]
	if !ok || emoji == "" {
		return emoji, false
	}
	toned := false
	for _, e := range names {
		if e.Emoji == emoji {
			toned = e.Tones
			break
		}
	}
	if !toned {
		return emoji, false
	}
	base := []rune(strings.ReplaceAll(emoji, "️", ""))
	return string(base[0]) + string(tone) + string(base[1:]), true
}

// Expand replaces every known :name: in s with its emoji, a skin tone
// right after it (:wave::skin-tone-3:) applied, and leaves the rest as
// typed.
func Expand(s string) string {
	if !strings.Contains(s, ":") {
		return s
	}
	var b strings.Builder
	last := ""
	for len(s) > 0 {
		i := strings.IndexByte(s, ':')
		if i < 0 {
			b.WriteString(s)
			break
		}
		j := strings.IndexByte(s[i+1:], ':')
		if j < 0 {
			b.WriteString(s)
			break
		}
		name := s[i+1 : i+1+j]
		if toned, ok := Tone(last, name); ok && i == 0 {
			out := b.String()
			b.Reset()
			b.WriteString(strings.TrimSuffix(out, last) + toned)
			last = ""
			s = s[i+2+j:]
			continue
		}
		if e, ok := Lookup(name); ok && validName(name) {
			b.WriteString(s[:i])
			b.WriteString(e)
			last = e
			s = s[i+2+j:]
			continue
		}
		b.WriteString(s[:i+1])
		last = ""
		s = s[i+1:]
	}
	return b.String()
}

// validName reports whether name could be a shortcode: letters, digits
// and _+-, nothing else.
func validName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' && r != '+' && r != '-' {
			return false
		}
	}
	return true
}

// NameRune reports whether r can be part of a shortcode's name.
func NameRune(r rune) bool { return validName(string(r)) }
