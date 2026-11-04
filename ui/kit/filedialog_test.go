package kit_test

import (
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/merlindorin/ember/geom"
	"github.com/merlindorin/ember/input"
	"github.com/merlindorin/ember/platform"
	"github.com/merlindorin/ember/ui"
	"github.com/merlindorin/ember/ui/align"
	"github.com/merlindorin/ember/ui/kit"
	"github.com/merlindorin/ember/world"
)

// modified is when the test files were last changed.
func modified() time.Time { return time.Date(2026, 11, 2, 14, 30, 0, 0, time.UTC) }

// assets is a pictures folder of pictures, a note and an old folder, and a music folder.
func assets() fstest.MapFS {
	file := func(size int) *fstest.MapFile { return &fstest.MapFile{Data: make([]byte, size), ModTime: modified()} }
	return fstest.MapFS{
		"pictures/sunset.png":     file(29000),
		"pictures/beach.png":      file(3000),
		"pictures/garden.png":     file(32000),
		"pictures/notes.txt":      file(40000),
		"pictures/.hidden":        file(1),
		"pictures/old/winter.png": file(523),
		"music/song.ogg":          file(9000),
		"music/cover.xyz":         file(10),
		"readme.md":               file(100),
	}
}

func day(t time.Time) string { return t.Format("2006-01-02") }

func dialog(t *testing.T, opts ...kit.FileDialogOption) (*shown, *kit.FileDialog) {
	t.Helper()
	s := showIn(t, 960, 640)
	fd := kit.NewFileDialog(s.doc, assets(), append([]kit.FileDialogOption{kit.DateFormat(day)}, opts...)...)
	s.w.AddSystems(fd)
	s.frames(1)
	return s, fd
}

func names(fd *kit.FileDialog) []string {
	var out []string
	for _, r := range kit.TableRows(fd.Table()) {
		out = append(out, kit.CellOf(r, 0).Text())
	}
	return out
}

func TestAFileDialogListsAFolderFoldersFirst(t *testing.T) {
	s, fd := dialog(t, kit.Extensions(".png"))
	fd.Open("pictures")
	s.frames(2)
	assert.True(t, s.doc.Open(fd.Node()))
	assert.True(t, fd.Shown())
	assert.Equal(t, []string{"old", "beach.png", "garden.png", "notes.txt", "sunset.png"}, names(fd),
		"hidden files left out")
	png := kit.TableRows(fd.Table())[3]
	assert.True(t, hasAttr(png, "aria-disabled"), "a file of another type is dimmed")
	row := kit.TableRows(fd.Table())[4]
	assert.Equal(t, []string{"sunset.png", "2026-11-02", "29 KB", "PNG image"},
		[]string{kit.CellOf(row, 0).Text(), kit.CellOf(row, 1).Text(), kit.CellOf(row, 2).Text(), kit.CellOf(row, 3).Text()})
}

func TestAFolderOpensInPlaceAndBackReturns(t *testing.T) {
	s, fd := dialog(t)
	fd.Open("pictures")
	s.frames(2)
	s.click(kit.TableRows(fd.Table())[0])
	s.click(kit.TableRows(fd.Table())[0])
	assert.Equal(t, "pictures/old", fd.Folder(), "a double click opens the folder")
	assert.Equal(t, []string{"winter.png"}, names(fd))
	s.click(find(t, s, "back"))
	assert.Equal(t, "pictures", fd.Folder())
	s.click(find(t, s, "forward"))
	assert.Equal(t, "pictures/old", fd.Folder())
}

func TestADoubleClickedFileIsChosen(t *testing.T) {
	s, fd := dialog(t, kit.Extensions(".png"))
	chosen, cancelled := watch[kit.FileChosen](s), watch[kit.DialogCancelled](s)
	fd.Open("pictures")
	s.frames(2)
	png := kit.TableRows(fd.Table())[3]
	s.click(png)
	s.click(png)
	assert.Empty(t, *chosen, "a dimmed file can't be chosen")
	sunset := kit.TableRows(fd.Table())[4]
	s.click(sunset)
	s.click(sunset)
	require.Len(t, *chosen, 1)
	assert.Equal(t, kit.FileChosen{Dialog: fd, Path: "pictures/sunset.png"}, (*chosen)[0])
	assert.False(t, s.doc.Open(fd.Node()), "it closes once a file is chosen")
	assert.False(t, fd.Shown())
	assert.Empty(t, *cancelled)
}

func TestOpenPicksTheSelectionAndCancelCancels(t *testing.T) {
	s, fd := dialog(t)
	chosen, cancelled := watch[kit.FileChosen](s), watch[kit.DialogCancelled](s)
	fd.Open("pictures")
	s.frames(2)
	s.click(kit.TableRows(fd.Table())[2])
	s.click(find(t, s, "Open"))
	require.Len(t, *chosen, 1)
	assert.Equal(t, "pictures/garden.png", (*chosen)[0].Path)
	fd.Open("pictures")
	s.frames(2)
	s.click(find(t, s, "Cancel"))
	assert.Equal(t, []kit.DialogCancelled{{Dialog: fd}}, *cancelled)
	fd.Open("pictures")
	s.frames(2)
	s.key(input.KeyEscape)
	s.frames(1)
	assert.Len(t, *cancelled, 2, "Escape cancels the modal dialog")
}

func TestTheSearchFieldFiltersTheFolder(t *testing.T) {
	s, fd := dialog(t)
	fd.Open("pictures")
	s.frames(2)
	s.click(kit.SearchInput(fd.SearchField()))
	s.in.AddText('e')
	s.in.AddText('a')
	s.frames(3)
	assert.Equal(t, []string{"beach.png"}, names(fd), "only names holding the term")
}

func TestThePlacesAndRecentFilesAreDownTheSide(t *testing.T) {
	s, fd := dialog(t, kit.Places("pictures", "music"), kit.Recent("pictures/garden.png"))
	chosen := watch[kit.FileChosen](s)
	fd.Open("pictures")
	s.frames(2)
	s.click(find(t, s, "music"))
	assert.Equal(t, "music", fd.Folder())
	assert.Equal(t, []string{"cover.xyz", "song.ogg"}, names(fd))
	s.click(find(t, s, "garden.png"))
	require.Len(t, *chosen, 1, "a recent file is picked")
	assert.Equal(t, "pictures/garden.png", (*chosen)[0].Path)
}

func TestTheGridShowsTheSameFilesInSync(t *testing.T) {
	s, fd := dialog(t)
	fd.Open("pictures")
	s.frames(2)
	s.click(find(t, s, "grid"))
	grid := fd.Thumbnails()
	assert.False(t, hasAttr(grid, "hidden"))
	require.Len(t, grid.Children(), 5)
	s.click(grid.Children()[2])
	assert.Equal(t, 2, s.doc.Selected(kit.TableGrid(fd.Table())), "selecting a thumbnail selects its row")
}

func TestTwoDialogsNeverCross(t *testing.T) {
	s := showIn(t, 960, 640)
	open := kit.NewFileDialog(s.doc, assets(), kit.DateFormat(day))
	save := kit.NewFileDialog(s.doc, assets(), kit.DateFormat(day))
	s.w.AddSystems(open, save)
	chosen, cancelled := watch[kit.FileChosen](s), watch[kit.DialogCancelled](s)
	open.Open("pictures")
	s.frames(2)
	sunset := kit.TableRows(open.Table())[4]
	s.click(sunset)
	s.click(sunset)
	require.Len(t, *chosen, 1)
	assert.Same(t, open, (*chosen)[0].Dialog)
	assert.Empty(t, save.Folder(), "the other dialog never listed anything")
	save.Open("music")
	s.frames(2)
	s.key(input.KeyEscape)
	s.frames(1)
	require.Len(t, *cancelled, 1)
	assert.Same(t, save, (*cancelled)[0].Dialog)
}

func TestADialogOutOfTheDocumentDoesNothing(t *testing.T) {
	s, fd := dialog(t)
	chosen := watch[kit.FileChosen](s)
	fd.Open("pictures")
	s.frames(2)
	s.doc.Body().Remove(fd.Node())
	s.frames(1)
	s.doc.Body().Append(fd.Node())
	s.frames(2)
	sunset := kit.TableRows(fd.Table())[4]
	s.click(sunset)
	s.click(sunset)
	assert.Len(t, *chosen, 1, "back in the Document it reacts again")

	fd.Open("pictures")
	s.frames(2)
	s.w.RemoveEntity(s.doc)
	s.frames(2)
	assert.Len(t, *chosen, 1, "without its Document in the World it emits nothing")
}

func hasAttr(n *ui.Node, name string) bool { _, ok := n.Attr(name); return ok }

// find is the innermost element reading text, or labelled text.
func find(t *testing.T, s *shown, text string) *ui.Node {
	t.Helper()
	var found *ui.Node
	s.doc.HTML().Walk(func(n *ui.Node) bool {
		if n.IsText() {
			return false
		}
		if l, _ := n.Attr("aria-label"); l == text || n.Text() == text {
			found = n
		}
		return true
	})
	require.NotNil(t, found, "no element reads %q", text)
	return found
}

func TestSelectPicksAFileFromTheGame(t *testing.T) {
	s, fd := dialog(t)
	fd.Open("pictures")
	fd.Select("pictures/garden.png")
	s.frames(2)
	assert.Equal(t, 2, s.doc.Selected(kit.TableGrid(fd.Table())))
	assert.Contains(t, find(t, s, "garden.png").Parent().Text(), "PNG image · 32 KB · modified 2026-11-02",
		"the details under the list")
}

func TestTheListHasTheFocusOnceOpen(t *testing.T) {
	s, fd := dialog(t)
	fd.Open("pictures")
	s.frames(2)
	assert.True(t, s.doc.Focused(kit.TableGrid(fd.Table())))
}

func commandKey() input.Key {
	if runtime.GOOS == "darwin" {
		return input.KeyLeftSuper
	}
	return input.KeyLeftControl
}

func TestCommandUpGoesUpAFolder(t *testing.T) {
	s, fd := dialog(t)
	fd.Open("pictures/old")
	s.frames(2)
	s.in.SetKey(commandKey(), true)
	s.frames(1)
	s.key(input.KeyUp)
	s.in.SetKey(commandKey(), false)
	s.frames(2)
	assert.Equal(t, "pictures", fd.Folder())
	s.in.SetKey(commandKey(), true)
	s.frames(1)
	s.key(input.KeyUp)
	s.in.SetKey(commandKey(), false)
	s.frames(2)
	assert.Equal(t, ".", fd.Folder(), "up to the FS's root")
}

func TestTypingOnTheListSearchesTheFolder(t *testing.T) {
	s, fd := dialog(t)
	fd.Open("pictures")
	s.frames(2)
	s.in.SetKey(input.KeyU, true)
	s.in.AddText('u')
	s.frames(1)
	s.in.SetKey(input.KeyU, false)
	s.frames(2)
	assert.Equal(t, "u", s.doc.Value(kit.SearchInput(fd.SearchField())))
	assert.True(t, s.doc.Focused(kit.SearchInput(fd.SearchField())), "the search field takes the rest of the typing")
	assert.Equal(t, []string{"sunset.png"}, names(fd))
	assert.Equal(t, 0, s.doc.Selected(kit.TableGrid(fd.Table())), "the first match selected")
}

func TestTheTypePopUpChoosesWhichFilesCanBeChosen(t *testing.T) {
	s, fd := dialog(t, kit.FileType("Pictures", ".png"), kit.FileType("Notes", ".txt"))
	fd.Open("pictures")
	s.frames(2)
	png := func() *ui.Node { return kit.TableRows(fd.Table())[3] }
	require.True(t, hasAttr(png(), "aria-disabled"))
	s.doc.Focus(find(t, s, "Pictures (.png)").Parent())
	s.frames(1)
	s.key(input.KeyEnter)
	s.key(input.KeyDown)
	s.key(input.KeyEnter)
	assert.Equal(t, "Notes", fd.FileType())
	assert.False(t, hasAttr(png(), "aria-disabled"), "notes can be chosen now")
	assert.True(t, hasAttr(kit.TableRows(fd.Table())[1], "aria-disabled"), "pictures can't")
}

func save(t *testing.T, name string) (*shown, *kit.FileDialog, *[]kit.FileChosen) {
	t.Helper()
	s, fd := dialog(t, kit.Extensions(".png"))
	chosen := watch[kit.FileChosen](s)
	fd.Save("pictures", name)
	s.frames(2)
	return s, fd, chosen
}

func TestSaveAsksForANameAndAddsTheExtension(t *testing.T) {
	s, fd, chosen := save(t, "holiday")
	assert.True(t, fd.Saving())
	assert.Equal(t, "holiday", s.doc.Value(fd.NameField()))
	assert.True(t, s.doc.Focused(fd.NameField()), "the name field has the focus")
	s.click(find(t, s, "Save"))
	require.Len(t, *chosen, 1)
	assert.Equal(t, kit.FileChosen{Dialog: fd, Path: "pictures/holiday.png", Saving: true}, (*chosen)[0])
}

func TestSaveWithoutANameSaysSo(t *testing.T) {
	s, _, chosen := save(t, "  ")
	s.click(find(t, s, "Save"))
	assert.Empty(t, *chosen)
	find(t, s, "Type a name to save under.")
}

func TestAFileSelectedGivesItsNameToSave(t *testing.T) {
	s, fd, _ := save(t, "")
	s.click(kit.TableRows(fd.Table())[2])
	assert.Equal(t, "garden.png", s.doc.Value(fd.NameField()))
}

func TestSavingOverAFileAsksFirst(t *testing.T) {
	s, fd, chosen := save(t, "sunset")
	s.click(find(t, s, "Save"))
	assert.Empty(t, *chosen, "not before the player agrees")
	find(t, s, "Replace “sunset.png”?")
	assert.True(t, s.doc.Focused(find(t, s, "Keep")), "Keep takes the focus, so Enter never overwrites")
	s.click(find(t, s, "Keep"))
	assert.Empty(t, *chosen)
	assert.True(t, s.doc.Open(fd.Node()), "the dialog stays to pick another name")
	s.click(find(t, s, "Save"))
	s.click(find(t, s, "Replace"))
	require.Len(t, *chosen, 1)
	assert.Equal(t, "pictures/sunset.png", (*chosen)[0].Path)
}

func TestEnterInTheNameFieldSaves(t *testing.T) {
	s, _, chosen := save(t, "holiday")
	s.key(input.KeyEnter)
	require.Len(t, *chosen, 1)
	assert.Equal(t, "pictures/holiday.png", (*chosen)[0].Path)
}

func TestRootNameNamesTheFSRoot(t *testing.T) {
	s, fd := dialog(t, kit.RootName("assets"))
	fd.Open("pictures/old")
	s.frames(2)
	assert.Equal(t, "assets", fd.RootName())
	assert.Equal(t, "old — pictures › assets", strings.TrimSuffix(fd.PathPopUp().Children()[1].Text(), ""))
	find(t, s, "assets")
}

func TestPickPressesTheDialogsButton(t *testing.T) {
	s, fd, chosen := save(t, "sunset")
	fd.Pick()
	s.frames(2)
	find(t, s, "Replace “sunset.png”?")
	assert.Empty(t, *chosen)
	fd.Save("pictures", "holiday")
	fd.Pick()
	s.frames(2)
	require.Len(t, *chosen, 1)
	assert.Equal(t, "pictures/holiday.png", (*chosen)[0].Path)
}

func TestTheDialogsControlsLineUp(t *testing.T) {
	s, fd := dialog(t, kit.FileType("Pictures", ".png"))
	fd.Open("pictures")
	s.frames(2)
	assert.Empty(t, align.Check(s.doc), "opening")
	fd.Save("pictures", "x")
	s.frames(2)
	assert.Empty(t, align.Check(s.doc), "saving")
}

func TestKindNamesAGamesOwnFiles(t *testing.T) {
	s, fd := dialog(t, kit.Kind("Track", ".ogg"))
	fd.Open("music")
	s.frames(2)
	rows := kit.TableRows(fd.Table())
	assert.Equal(t, "Track", kit.CellOf(rows[1], 3).Text(), "it wins over the dialog's Ogg sound")
	assert.Equal(t, "Track", fd.Kind(".OGG"))
	assert.Equal(t, "PNG image", fd.Kind(".png"))
}

func TestAnUnknownExtensionNamesItsFile(t *testing.T) {
	s, fd := dialog(t)
	fd.Open("music")
	s.frames(2)
	assert.Equal(t, "XYZ file", kit.CellOf(kit.TableRows(fd.Table())[0], 3).Text())
	assert.Empty(t, fd.Kind(".xyz"))
}

// clickIn clicks n in the window win.
func clickIn(s *shown, win *platform.Window, n *ui.Node) {
	in := win.Input()
	in.SetMouse(n.Box().Center())
	s.frames(1)
	in.SetMouseButton(input.MouseLeft, true)
	s.frames(1)
	in.SetMouseButton(input.MouseLeft, false)
	s.frames(2)
}

func TestOwnWindowShowsTheDialogInAWindowOfItsOwn(t *testing.T) {
	s, fd := dialog(t, kit.OwnWindow())
	assert.True(t, fd.OwnWindow())
	fd.Open("pictures")
	s.frames(2)
	win := fd.Window()
	require.NotNil(t, win)
	assert.Same(t, s.w, world.Of(win))
	assert.Equal(t, geom.V2(900, 600), win.Size())
	assert.Equal(t, geom.Rect{W: 900, H: 600}, fd.Node().Box(), "the dialog fills it")
	lights := fd.Node().Children()[0].Children()[0]
	require.True(t, lights.HasClass("kit-window-lights"))
	assert.Zero(t, lights.Box().W, "the OS window has the lights")
	assert.Empty(t, s.doc.Body().Children(), "out of the game's page")
}

func TestAFileChosenInItsOwnWindowClosesIt(t *testing.T) {
	s, fd := dialog(t, kit.OwnWindow())
	chosen := watch[kit.FileChosen](s)
	fd.Open("pictures")
	s.frames(2)
	win := fd.Window()
	sunset := kit.TableRows(fd.Table())[4]
	clickIn(s, win, sunset)
	clickIn(s, win, sunset)
	require.Len(t, *chosen, 1)
	assert.Equal(t, "pictures/sunset.png", (*chosen)[0].Path)
	assert.Nil(t, world.Of(win), "the window goes")
	assert.Nil(t, fd.Window())
}

func TestClosingItsOwnWindowCancels(t *testing.T) {
	s, fd := dialog(t, kit.OwnWindow())
	cancelled := watch[kit.DialogCancelled](s)
	fd.Open("pictures")
	s.frames(2)
	win := fd.Window()
	world.Emit(s.w, platform.WindowClosed{Window: win})
	s.w.RemoveEntity(win)
	s.frames(3)
	assert.Equal(t, []kit.DialogCancelled{{Dialog: fd}}, *cancelled)
	assert.Nil(t, fd.Window())
	fd.Open("music")
	s.frames(2)
	require.NotNil(t, fd.Window(), "a new window the next time")
	assert.Equal(t, []string{"cover.xyz", "song.ogg"}, names(fd))
}

func TestTurningOwnWindowOffBringsTheDialogBackToThePage(t *testing.T) {
	s, fd := dialog(t, kit.OwnWindow())
	fd.Open("pictures")
	s.frames(2)
	win := fd.Window()
	world.Emit(s.w, platform.WindowClosed{Window: win})
	s.w.RemoveEntity(win)
	s.frames(2)
	fd.SetOwnWindow(false)
	fd.Open("music")
	s.frames(2)
	assert.Nil(t, fd.Window())
	assert.Contains(t, s.doc.Body().Children(), fd.Node(), "back in the game's page")
	assert.True(t, s.doc.Open(fd.Node()))
	assert.Greater(t, fd.Node().Box().W, float32(0), "and shown there")
	assert.Equal(t, []string{"cover.xyz", "song.ogg"}, names(fd))
}

func TestAnOpenBeforeTheDialogIsASystemStillTakesItsOwnWindow(t *testing.T) {
	s := showIn(t, 960, 640)
	fd := kit.NewFileDialog(s.doc, assets(), kit.DateFormat(day), kit.OwnWindow())
	fd.Open("music")
	assert.True(t, fd.Shown())
	s.w.AddSystems(fd)
	s.frames(2)
	require.NotNil(t, fd.Window(), "its window opens at its first frame")
	assert.Empty(t, s.doc.Body().Children(), "never in the game's page")
	assert.Equal(t, []string{"cover.xyz", "song.ogg"}, names(fd))
}

func TestTheDialogsPartsHaveAutomationNames(t *testing.T) {
	s, fd := dialog(t, kit.DialogLabel("open"))
	fd.Open("pictures")
	s.frames(2)
	_, all := s.auto.Snapshot("", "")
	ids := map[string]bool{}
	for _, e := range all {
		ids[e.ID] = true
	}
	has := func(suffix string) bool {
		for id := range ids {
			if strings.HasSuffix(id, suffix) {
				return true
			}
		}
		return false
	}
	assert.True(t, has("open/files/sunset-png"), "the rows under the dialog's name and Files")
	fd.Save("pictures", "x")
	s.frames(2)
	_, all = s.auto.Snapshot("", "")
	found := false
	for _, e := range all {
		found = found || strings.HasSuffix(e.ID, "open/file-name")
	}
	assert.True(t, found, "the name field reads File name")
	assert.Equal(t, "open", fd.DialogLabel())
}

func TestAnUnnamedFileDialogScopesItsPartsAsFileDialog(t *testing.T) {
	s, fd := dialog(t)
	fd.Open("pictures")
	s.frames(2)
	_, all := s.auto.Snapshot("", "")
	var ids []string
	for _, e := range all {
		ids = append(ids, e.ID)
	}
	assert.Contains(t, ids, "file-dialog/files/sunset-png", "its rows")
	assert.Contains(t, ids, "file-dialog/ok", "and its Open button, apart from the dialog")
}

func TestTheViewToggleDrawsBothGlyphs(t *testing.T) {
	s, fd := dialog(t)
	fd.Open("pictures")
	s.frames(2)
	for _, kind := range []string{"list", "grid"} {
		var glyph *ui.Node
		fd.Node().Walk(func(n *ui.Node) bool {
			if v, _ := n.Attr("aria-label"); v == kind && n.HasClass("kit-view-glyph") {
				glyph = n
			}
			return glyph == nil
		})
		if assert.NotNil(t, glyph, kind) {
			for _, bar := range glyph.Children() {
				assert.Positive(t, bar.Box().H, "%s's marks show", kind)
			}
		}
	}
}

func TestOnChooseAndOnCancelRunWhereTheDialogIsMade(t *testing.T) {
	var chosen []string
	var cancels int
	s, fd := dialog(t,
		kit.OnChoose(func(e kit.FileChosen) { chosen = append(chosen, e.Path) }),
		kit.OnCancel(func() { cancels++ }))
	fd.Open("pictures")
	s.frames(2)
	s.click(kit.TableRows(fd.Table())[2])
	s.click(find(t, s, "Open"))
	assert.Equal(t, []string{"pictures/garden.png"}, chosen)
	fd.Open("pictures")
	s.frames(2)
	s.click(find(t, s, "Cancel"))
	assert.Equal(t, 1, cancels)
}

func TestOnChooseAndOnCancelCanTakeTheWorld(t *testing.T) {
	var worlds []*world.World
	s, fd := dialog(t,
		kit.OnChoose(func(w *world.World, _ kit.FileChosen) { worlds = append(worlds, w) }),
		kit.OnCancel(func(w *world.World) { worlds = append(worlds, w) }))
	fd.Open("pictures")
	s.frames(2)
	s.click(kit.TableRows(fd.Table())[2])
	s.click(find(t, s, "Open"))
	fd.Open("pictures")
	s.frames(2)
	s.click(find(t, s, "Cancel"))
	require.Len(t, worlds, 2)
	assert.Same(t, s.w, worlds[0])
	assert.Same(t, s.w, worlds[1])
}

func TestTheRootsNameIsWhatTheDialogsSentencesSay(t *testing.T) {
	s, fd := dialog(t, kit.RootName("Survivors saves"))
	chosen := watch[kit.FileChosen](s)
	fd.Open(".")
	s.frames(2)
	s.click(kit.SearchInput(fd.SearchField()))
	s.in.AddText('z')
	s.in.AddText('q')
	s.frames(3)
	find(t, s, "Nothing in Survivors saves matches “zq”.")

	fd.Save(".", "readme.md")
	s.frames(2)
	fd.Pick()
	s.frames(2)
	assert.Empty(t, *chosen)
	find(t, s, "A file with that name already exists in Survivors saves. Replacing it overwrites what it holds.")
}
