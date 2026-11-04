package main

import "testing"

func TestDiskPathsAreSlashPathsUnderTheVolume(t *testing.T) {
	for _, c := range []struct {
		root string
		sep  byte
		os   string
		fs   string
	}{
		{"/", '/', "/", "."},
		{"/", '/', "/Users/heron/crypt", "Users/heron/crypt"},
		{"/", '/', "/a/b.tsj", "a/b.tsj"},
		{`C:\`, '\\', `C:\`, "."},
		{`C:\`, '\\', `C:\Users\heron\crypt`, "Users/heron/crypt"},
		{`C:\`, '\\', `C:\a\b.tsj`, "a/b.tsj"},
	} {
		if got := fsPathIn(c.root, c.sep, c.os); got != c.fs {
			t.Errorf("fsPathIn(%q, %q) = %q, want %q", c.root, c.os, got, c.fs)
		}
		if got := osPathIn(c.root, c.sep, c.fs); got != c.os {
			t.Errorf("osPathIn(%q, %q) = %q, want %q", c.root, c.fs, got, c.os)
		}
	}
}
