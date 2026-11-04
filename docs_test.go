package ember_test

import (
	"bytes"
	"html"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/merlindorin/ember/internal/shots"
)

var (
	exampleLink = regexp.MustCompile(`(?m)^\[([\w-]+)\]\(((?:\.\./)+examples/([\w-]+))/?\)$`)
	unchanged   = regexp.MustCompile(
		`(?m)^Unchanged from \[[^\]]+\]\((?:\.\./)+examples/([\w-]+)/?\): ([\w.-]+(?:, [\w.-]+)*)\.$`,
	)
	fileLink = regexp.MustCompile(`(?m)^\[[^\]]+\]\((?:\.\./)+(examples/[\w-]+/[\w.-]+)\)$`)
	mdLink   = regexp.MustCompile(`\]\(([^)\s#]+)`)
	code     = regexp.MustCompile("(?s)```.*?```|`[^`\n]*`")
	siteLink = regexp.MustCompile(`(?:href|src)="/ember/([^"#?]*)`)
	numbered = regexp.MustCompile(`^\d{1,2}-`)
	tag      = regexp.MustCompile(`<[^>]*>`)
	decision = regexp.MustCompile(`\A# (\d{4})\. .+\n\n((?:- \w+: .+\n)+)`)
	field    = regexp.MustCompile(`(?m)^- (\w+): (.+)$`)
	status   = regexp.MustCompile(`^(?:Accepted|Superseded by \[(\d{4})\]\(([^)]+)\))`)
)

func TestTheGuideShowsTheExamplesAsTheyAre(t *testing.T) {
	public := buildGuide(t)

	err := filepath.WalkDir("docs", func(source string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(source, ".md") {
			return err
		}
		if notADecision(source) {
			return nil
		}
		return checkGuidePage(t, public, source)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// checkGuidePage checks the page of the guide made from source: its links,
// its title and the examples it shows as they are.
func checkGuidePage(t *testing.T, public, source string) error {
	md, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	for _, m := range mdLink.FindAllStringSubmatch(code.ReplaceAllString(string(md), ""), -1) {
		if strings.Contains(m[1], "://") || shotMade(source, m[1]) {
			continue
		}
		if _, err = os.Stat(filepath.Join(filepath.Dir(source), m[1])); err != nil {
			t.Errorf("%s: link %s is broken on GitHub", source, m[1])
		}
	}
	if strings.HasPrefix(source, filepath.Join("docs", "design")) {
		return nil
	}

	slug := pageOf(source)
	built, err := os.ReadFile(filepath.Join(public, slug, "index.html"))
	if err != nil {
		t.Errorf("%s is not published at /%s/", source, slug)
		return nil //nolint:nilerr // reported, and the walk goes on to the next page
	}
	page := guidePage{source: source, slug: slug, text: html.UnescapeString(tag.ReplaceAllString(string(built), ""))}

	if !strings.HasPrefix(string(md), "# ") {
		t.Errorf("%s does not start with its # title", source)
	}
	page.linked = map[string]bool{}
	for _, m := range fileLink.FindAllStringSubmatch(string(md), -1) {
		page.linked[filepath.FromSlash(m[1])] = true
	}
	page.same = map[string]string{}
	for _, m := range unchanged.FindAllStringSubmatch(string(md), -1) {
		for name := range strings.SplitSeq(m[2], ", ") {
			page.same[name] = m[1]
		}
	}
	uses := exampleLink.FindAllStringSubmatch(string(md), -1)
	if len(uses) == 0 && filepath.Base(source) != "README.md" && !strings.HasPrefix(slug, "docs/decisions/") &&
		!reference(source) && !moved(md) {
		t.Errorf("%s shows no example", source)
	}
	for _, m := range uses {
		if err = page.checkExample(t, m); err != nil {
			return err
		}
	}
	return nil
}

// guidePage is a published page of the guide: the files its source links
// to, and those it lists as unchanged, by name, from the example they were
// shown in.
type guidePage struct {
	source, slug, text string
	linked             map[string]bool
	same               map[string]string
}

// checkExample checks the page shows the example the link m names, each
// file as it is, unless it is set aside or listed as unchanged.
func (p guidePage) checkExample(t *testing.T, m []string) error {
	if m[1] != m[3] {
		t.Errorf("%s: link text %s names another example than %s", p.source, m[1], m[2])
	}
	dir := filepath.Join("examples", m[3])
	files, _ := filepath.Glob(filepath.Join(dir, "*.go"))
	if len(files) == 0 {
		t.Errorf("%s: %s has no Go files", p.source, dir)
	}
	if _, err := os.Stat(filepath.Join(dir, "output.txt")); err == nil {
		files = append(files, filepath.Join(dir, "output.txt"))
	}
	for _, f := range files {
		want, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		shown := strings.Contains(p.text, string(want))
		if from, ok := p.same[filepath.Base(f)]; ok {
			before, beforeErr := os.ReadFile(filepath.Join("examples", from, filepath.Base(f)))
			if beforeErr != nil || string(before) != string(want) {
				t.Errorf("%s: %s is listed as unchanged from %s, but it changed", p.slug, f, from)
			}
			if shown {
				t.Errorf("%s: %s is listed as unchanged but shown too", p.slug, f)
			}
			continue
		}
		aside := strings.HasSuffix(f, "_test.go") || strings.HasPrefix(string(want), "//go:build ignore")
		if wanted := !aside || p.linked[f]; shown != wanted {
			t.Errorf("%s: %s shown %v", p.slug, f, shown)
		}
	}
	if !strings.Contains(p.text, "go run ./examples/"+m[3]) {
		t.Errorf("%s: no go run line for %s", p.slug, m[3])
	}
	return nil
}

// TestEveryGuidePictureIsCommittedOrMade checks, without a GPU or the
// pictures, that each picture the guide shows is either made by task
// docs:shots (every .png and .gif, from internal/shots's list) or kept in
// git (diagrams and the site's own art).
func TestEveryGuidePictureIsCommittedOrMade(t *testing.T) {
	refs, err := shots.Refs(".")
	if err != nil {
		t.Fatal(err)
	}
	made := map[string]bool{}
	for _, s := range shots.Shots {
		made[s.Out] = true
	}
	for _, r := range refs {
		if shots.Made(r.Picture) {
			if !made[r.Picture] && !siteArt(r.Picture) {
				t.Errorf("%s shows %s, which no shot in internal/shots makes", r.Page, r.Picture)
			}
			continue
		}
		if _, missing := os.Stat(filepath.Join("docs", filepath.FromSlash(r.Picture))); missing != nil {
			t.Errorf("%s shows %s, which isn't there", r.Page, r.Picture)
		}
	}
}

// shotMade reports whether link, from the page source, is a picture task
// docs:shots makes, there only once it has run (checked by
// TestEveryGuidePictureIsCommittedOrMade).
func shotMade(source, link string) bool {
	rel, err := filepath.Rel("docs", filepath.Join(filepath.Dir(source), link))
	if err != nil || !shots.Made(filepath.ToSlash(rel)) {
		return false
	}
	for _, s := range shots.Shots {
		if s.Out == filepath.ToSlash(rel) {
			return true
		}
	}
	return false
}

// siteArt reports whether picture is the site's own, kept in
// .hugo/assets/images rather than made.
func siteArt(picture string) bool {
	_, err := os.Stat(filepath.Join(".hugo", "assets", "images", filepath.FromSlash(picture)))
	return err == nil
}

// TestTheGuideHasNoDeadLinks also checks the live examples' links once task
// web:guide has built them into dist/live, which the site mounts as live/.
func TestTheGuideHasNoDeadLinks(t *testing.T) {
	public := buildGuide(t)
	_, err := os.Stat(filepath.Join("dist", "live", "index.html"))
	liveBuilt := err == nil

	err = filepath.WalkDir(public, func(file string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(file, ".html") {
			return err
		}
		built, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		page, _ := filepath.Rel(public, file)
		for _, m := range siteLink.FindAllStringSubmatch(string(built), -1) {
			if !liveBuilt && strings.HasPrefix(m[1], "live/") {
				continue
			}
			target := filepath.Join(public, filepath.FromSlash(m[1]))
			info, statErr := os.Stat(target)
			if statErr == nil && info.IsDir() {
				_, statErr = os.Stat(filepath.Join(target, "index.html"))
			}
			if statErr != nil {
				t.Errorf("%s links to /ember/%s, which is not published", page, m[1])
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestEveryEngineDecisionHasItsMetadata(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join("docs", "99-decisions", "[0-9][0-9][0-9][0-9]-*.md"))
	if len(files) == 0 {
		t.Fatal("no decisions")
	}
	for _, file := range files {
		md, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		m := decision.FindSubmatch(md)
		if m == nil {
			t.Errorf("%s: no # NNNN. title followed by the Status, Date, Area and Owner list", file)
			continue
		}
		if !strings.HasPrefix(filepath.Base(file), string(m[1])+"-") {
			t.Errorf("%s: titled %s", file, m[1])
		}
		fields := map[string]string{}
		for _, f := range field.FindAllStringSubmatch(string(m[2]), -1) {
			fields[f[1]] = f[2]
		}
		for _, name := range []string{"Status", "Date", "Area", "Owner"} {
			if fields[name] == "" {
				t.Errorf("%s: no %s", file, name)
			}
		}
		if _, err = time.Parse(time.DateOnly, fields["Date"]); err != nil {
			t.Errorf("%s: date %q is not YYYY-MM-DD", file, fields["Date"])
		}
		s := status.FindStringSubmatch(fields["Status"])
		if s == nil {
			t.Errorf("%s: status %q is neither Accepted nor Superseded by [NNNN](…)", file, fields["Status"])
			continue
		}
		if s[1] != "" && !strings.HasPrefix(path.Base(s[2]), s[1]+"-") {
			t.Errorf("%s: superseded by %s, but the link goes to %s", file, s[1], s[2])
		}
		if s[2] != "" {
			if _, err = os.Stat(filepath.Join(filepath.Dir(file), s[2])); err != nil {
				t.Errorf("%s: superseded by %s, which does not exist", file, s[2])
			}
		}
	}
}

func buildGuide(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("hugo"); err != nil {
		t.Skip("hugo is not installed")
	}
	public := t.TempDir()
	build := exec.Command("hugo", "--panicOnWarning", "--quiet", "--destination", public)
	// The pictures task docs:shots makes may not be there, on a fresh clone:
	// TestEveryGuidePictureIsCommittedOrMade checks them instead.
	build.Env = append(os.Environ(), "HUGO_PARAMS_SHOTSOPTIONAL=true")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("hugo: %v\n%s", err, out)
	}
	return public
}

// reference reports whether source is a reference page, which shows its
// subject in pictures and snippets rather than an example: a page of the
// components reference, whose section page shows the gallery example they
// come from, a page of the ui section, whose pictures come from the kit's
// own kit pages, Tile properties, or the Light pages on lit layers and on
// debugging.
func reference(source string) bool {
	return strings.HasPrefix(source, filepath.Join("docs", "05-ui")+string(filepath.Separator)) ||
		slices.Contains([]string{
			filepath.Join("docs", "06-2d", "12-tile-properties.md"),
			filepath.Join("docs", "06-2d", "19-lights", "01-lit-layers.md"),
			filepath.Join("docs", "06-2d", "19-lights", "08-debugging.md"),
		}, source)
}

// moved reports whether a page only points to where its example went, as
// The old ember/ui pages do once their example is on ui.
func moved(md []byte) bool {
	return bytes.Contains(md, []byte("\nMoved to uiv2: "))
}

// notADecision is a file such as TEMPLATE.md beside the decisions, which the guide does not publish.
func notADecision(source string) bool {
	dir, file := filepath.Split(source)
	return filepath.Clean(dir) == filepath.Join("docs", "99-decisions") && file != "README.md" &&
		!regexp.MustCompile(`^\d{4}-`).MatchString(file)
}

// pageOf maps docs/02-world/01-entities-and-systems.md to docs/world/entities-and-systems,
// and a README.md to its folder.
func pageOf(source string) string {
	parts := []string{"docs"}
	for _, p := range strings.Split(filepath.ToSlash(strings.TrimPrefix(source, "docs"+string(filepath.Separator))), "/") {
		p = strings.TrimSuffix(p, ".md")
		if p != "README" && p != "" {
			parts = append(parts, numbered.ReplaceAllString(p, ""))
		}
	}
	return path.Join(parts...)
}
