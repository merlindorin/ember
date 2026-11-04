package main

import (
	"io/fs"
	"slices"
	"testing"
	"testing/fstest"
)

func TestDiskPathsAreSlashPathsFromTheRoot(t *testing.T) {
	for _, c := range []struct {
		windows bool
		os, fs  string
	}{
		{false, "/", "."},
		{false, "/Users/heron/crypt", "Users/heron/crypt"},
		{false, "/a/b.tsj", "a/b.tsj"},
		{true, `C:\`, "C:"},
		{true, `C:\Users\heron\crypt`, "C:/Users/heron/crypt"},
		{true, `D:\a\b.tsj`, "D:/a/b.tsj"},
	} {
		if got := fsPathFor(c.windows, c.os); got != c.fs {
			t.Errorf("fsPathFor(%v, %q) = %q, want %q", c.windows, c.os, got, c.fs)
		}
		if got := osPathFor(c.windows, c.fs); got != c.os {
			t.Errorf("osPathFor(%v, %q) = %q, want %q", c.windows, c.fs, got, c.os)
		}
	}
}

func TestOnWindowsTheDialogsRootListsEveryDrive(t *testing.T) {
	drives := map[string]fs.FS{
		"C:": fstest.MapFS{"Users/heron/walls.tsj": {Data: []byte("{}")}},
		"D:": fstest.MapFS{"a/ember/props.tsj": {Data: []byte("{}")}},
	}
	v := volumes{
		list: func() []string { return []string{"C:", "D:"} },
		open: func(name string) fs.FS { return drives[name] },
	}
	root, err := fs.ReadDir(v, ".")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range root {
		if !e.IsDir() {
			t.Errorf("%s: a drive is a folder", e.Name())
		}
		names = append(names, e.Name())
	}
	if !slices.Equal(names, []string{"C:", "D:"}) {
		t.Fatalf("the root lists the drives: %v", names)
	}
	in, err := fs.ReadDir(v, "D:/a/ember")
	if err != nil || len(in) != 1 || in[0].Name() != "props.tsj" {
		t.Fatalf("a folder on the second drive lists as on its own: %v %v", in, err)
	}
	info, err := fs.Stat(v, "C:/Users/heron/walls.tsj")
	if err != nil || info.IsDir() {
		t.Fatalf("a file stats through its drive: %v %v", info, err)
	}
	if _, err = fs.ReadDir(v, "E:/x"); err == nil {
		t.Fatal("a drive that isn't there lists nothing")
	}
}
