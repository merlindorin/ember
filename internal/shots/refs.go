package shots

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// markdownImage is an image in a page: its target, relative to the page.
	markdownImage = regexp.MustCompile(`!\[[^\]]*\]\(([^)\s]+)`)
	// code is a page's code: fenced blocks and inline spans, whose images
	// are examples of Markdown, not pictures.
	code = regexp.MustCompile("(?s)```.*?```|`[^`\n]*`")
	// shortcodeImage is a picture the site's own pages name, under docs.
	shortcodeImage = regexp.MustCompile(`\bimage2?="([^"]+)"`)
)

// Ref is a picture the guide shows: its path under docs/, and the page
// that shows it.
type Ref struct{ Picture, Page string }

// Refs are the pictures root's guide shows: the images of every page
// under docs/, relative to their page, and those the site's own pages
// (.hugo/content) name, relative to docs/. Links leaving the guide are
// left out.
func Refs(root string) ([]Ref, error) {
	var out []Ref
	docs := filepath.Join(root, "docs")
	err := filepath.WalkDir(docs, func(file string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(file, ".md") {
			return err
		}
		b, err := os.ReadFile(file) //nolint:gosec // the repository's own pages, walked from its root
		if err != nil {
			return err
		}
		page, _ := filepath.Rel(docs, file)
		page = filepath.ToSlash(page)
		for _, m := range markdownImage.FindAllStringSubmatch(code.ReplaceAllString(string(b), ""), -1) {
			if strings.Contains(m[1], "://") {
				continue
			}
			out = append(out, Ref{path.Clean(path.Join(path.Dir(page), m[1])), page})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	site, _ := filepath.Glob(filepath.Join(root, ".hugo", "content", "*.md"))
	for _, file := range site {
		b, readErr := os.ReadFile(file)
		if readErr != nil {
			return nil, readErr
		}
		for _, m := range shortcodeImage.FindAllStringSubmatch(string(b), -1) {
			out = append(out, Ref{m[1], filepath.ToSlash(file)})
		}
	}
	return out, nil
}

// Made reports whether a picture is one task docs:shots makes rather than
// a file kept in git: every .png and .gif under docs/. A page may also show
// a committed file from outside docs/, such as an example's sprite sheet.
func Made(picture string) bool {
	ext := path.Ext(picture)
	return !strings.HasPrefix(picture, "../") && (ext == ".png" || ext == ".gif")
}
