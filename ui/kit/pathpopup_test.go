package kit_test

import (
	"path/filepath"
	"testing"

	"github.com/merlindorin/ember/automation"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/kit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPathPopUpListsTheFoldersUpToTheRoot(t *testing.T) {
	root := string(filepath.Separator)
	path := filepath.Join(root, "assets", "survivors", "crypt")
	p := kit.PathPopUp(path)
	s := show(t, p)
	clicks := watch[ui.Clicked](s)
	assert.Equal(t, path, kit.Folder(p))

	s.click(p)
	var items []string
	var assets *ui.Node
	_, all := s.auto.Snapshot("", "")
	for _, e := range all {
		if e.Role == automation.RoleMenuItem {
			items = append(items, e.Label)
		}
	}
	assert.Equal(t, []string{"crypt", "survivors", "assets", root}, items, "from the folder up to the root")
	p.Walk(func(n *ui.Node) bool {
		if v, _ := n.Attr("aria-label"); v == "assets" && n != p {
			assets = n
		}
		return assets == nil
	})
	require.NotNil(t, assets)

	s.click(assets)
	require.Len(t, *clicks, 1)
	chosen, ok := kit.FolderOf((*clicks)[0].Node)
	require.True(t, ok)
	assert.Equal(t, filepath.Join(root, "assets"), chosen)
	_, ok = kit.FolderOf(p)
	assert.False(t, ok, "only a path pop-up's items stand for folders")

	kit.SetFolder(p, chosen)
	s.frames(2)
	assert.Equal(t, chosen, kit.Folder(p))
	s.click(p)
	items = items[:0]
	_, all = s.auto.Snapshot("", "")
	for _, e := range all {
		if e.Role == automation.RoleMenuItem {
			items = append(items, e.Label)
		}
	}
	assert.Equal(t, []string{"assets", root}, items)
}

func TestAPathPopUpNamesItsRootAsRootNameSays(t *testing.T) {
	p := kit.PathPopUp(filepath.Join(string(filepath.Separator), "crypt", "old"), kit.RootName("assets"))
	s := show(t, p)
	s.click(p)
	var items []string
	_, all := s.auto.Snapshot("", "")
	for _, e := range all {
		if e.Role == automation.RoleMenuItem {
			items = append(items, e.Label)
		}
	}
	assert.Equal(t, []string{"old", "crypt", "assets"}, items, "the root reads as named")
	menu := p.Children()[len(p.Children())-1]
	root, ok := kit.FolderOf(menu.Children()[2])
	require.True(t, ok)
	assert.Equal(t, string(filepath.Separator), root, "and still stands for the root")
}

func TestAPathPopUpKeepsItsIDAcrossFoldersAndItsLabelHasNoChevron(t *testing.T) {
	p := kit.PathPopUp("/crypt/tiled/walls")
	s := show(t, p)
	e := s.element(t, "walls — tiled › crypt")
	assert.Equal(t, "path", e.ID)
	kit.SetFolder(p, "/crypt/props")
	s.frames(2)
	e = s.element(t, "props — crypt › /")
	assert.Equal(t, "path", e.ID)
}
