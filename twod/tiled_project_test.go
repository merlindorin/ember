package twod //nolint:testpackage // beside the classes' project tests, sharing crypt

import (
	"encoding/json"
	"testing"

	"github.com/merlindorin/ember/color"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// projectOf is p's folders, top-level properties and each type's colour,
// as Tiled reads them.
type projectOf struct {
	Folders       []string `json:"folders"`
	Properties    []any    `json:"properties"`
	PropertyTypes []struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"propertyTypes"`
}

func readProject(t *testing.T, b []byte) projectOf {
	t.Helper()
	var p projectOf
	require.NoError(t, json.Unmarshal(b, &p))
	return p
}

func (p projectOf) colour(name string) string {
	for _, c := range p.PropertyTypes {
		if c.Name == name {
			return c.Color
		}
	}
	return ""
}

func TestAProjectListsItsFoldersInTheOrderGiven(t *testing.T) {
	p := readProject(t, TiledProject(ProjectFolders("../library", "."), crypt()))
	assert.Equal(t, []string{"../library", "."}, p.Folders)
	assert.Equal(t, []string{"."}, readProject(t, TiledProject(crypt())).Folders, "without folders, the project's own")
}

func TestFoldersChangeNothingElseOfTheProject(t *testing.T) {
	plain := TiledProject(crypt())
	assert.Equal(t, string(plain), string(TiledProject(ProjectFolders("."), crypt())))
	var a, b map[string]any
	require.NoError(t, json.Unmarshal(plain, &a))
	require.NoError(t, json.Unmarshal(TiledProject(crypt(), ProjectFolders("a", "b")), &b))
	delete(a, "folders")
	delete(b, "folders")
	assert.Equal(t, a, b)
}

func TestAClassIsDrawnInItsColourAsTiledWritesIt(t *testing.T) {
	p := readProject(t, TiledProject(Classes(
		Class{Name: "level", On: OnMap, Color: color.Hex("#4f8fd8")},
		Class{Name: "place", On: OnObject, Color: color.Hex("#a0a0a040")},
		Class{Name: "plain", On: OnObject},
	)))
	assert.Equal(t, "#ff4f8fd8", p.colour("level"), "#aarrggbb")
	assert.Equal(t, "#40a0a0a0", p.colour("place"), "its alpha first")
	assert.Equal(t, "#ffa0a0a4", p.colour("plain"), "no colour: Tiled's own grey")
	assert.Equal(t, "#ffa0a0a4", p.colour("ember.Tile"))
}

func TestAProjectHasTheEmptyPropertiesTiledWrites(t *testing.T) {
	p := readProject(t, TiledProject(crypt()))
	assert.NotNil(t, p.Properties, "Tiled writes \"properties\": [] at every save; without it a saved project differs")
	assert.Empty(t, p.Properties)
}

// Tiled 1.12 writes a class's members by name, whatever order they were
// declared or saved in, and always writes its members and uses, empty or
// not: a project written otherwise changes at Tiled's first save.
func TestAClassIsWrittenAsTiledWritesIt(t *testing.T) {
	var p struct {
		PropertyTypes []map[string]json.RawMessage `json:"propertyTypes"`
	}
	require.NoError(t, json.Unmarshal(TiledProject(crypt(), Classes(
		Class{Name: "marker", On: OnObject},
		Class{Name: "gate", On: OnObject, Properties: []Property{
			{Name: "to", Kind: KindString}, {Name: "Lock", Kind: KindBool}, {Name: "key_id", Kind: KindInt},
		}},
	)), &p))
	for _, c := range p.PropertyTypes {
		var name string
		require.NoError(t, json.Unmarshal(c["name"], &name))
		if string(c["type"]) != `"class"` {
			continue
		}
		require.Contains(t, c, "members", name)
		require.Contains(t, c, "useAs", name)
		var members []struct {
			Name string `json:"name"`
		}
		require.NoError(t, json.Unmarshal(c["members"], &members))
		names := make([]string, len(members))
		for i, m := range members {
			names[i] = m.Name
		}
		assert.IsIncreasing(t, names, "%s's members by name, as Tiled orders them", name)
		if name == "marker" {
			assert.JSONEq(t, `[]`, string(c["members"]))
		}
	}
}

func TestAProjectIsNotEscapedForHTML(t *testing.T) {
	b := string(TiledProject(Classes(Class{Name: "gate", On: OnObject, Properties: []Property{
		{Name: "a<b", Kind: KindString, Default: "x&y"},
	}})))
	assert.Contains(t, b, `"a<b"`, "written as Tiled writes it")
	assert.Contains(t, b, `"x&y"`)
}
