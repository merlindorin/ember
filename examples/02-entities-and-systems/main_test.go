package main

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestTheFlamesGoOutOneByOne(t *testing.T) {
	want, err := os.ReadFile("output.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(stdoutOf(t, main), "\n") + "\n"; got != string(want) {
		t.Fatalf("main prints\n%s\nwant output.txt\n%s", got, want)
	}
}

func stdoutOf(t *testing.T, run func()) []string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = w
	run()
	os.Stdout = stdout
	w.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n")
}
