package main

import "github.com/merlindorin/ember/ui"

// elements are the kit's components by their page's name, each built in
// a file of its group.
func elements() map[string]func(*gallery) *ui.Node {
	return map[string]func(*gallery) *ui.Node{
		"button": button,

		"label":          label,
		"rich-text":      richText,
		"icons":          icons,
		"image":          image,
		"separator":      separator,
		"row-and-column": rowAndColumn,
		"grid":           grid,
		"panel":          panel,
		"check":          check,
		"switch":         switches,
		"slider":         slider,
		"progress":       progress,
		"text-input":     textInput,
		"text-area":      textArea,
		"number-field":   numberField,
		"choice":         choice,
		"canvas":         mapCanvas,
		"settings":       settings,
		"styled":         styled,
		"pop-up":         popUp,
		"path-pop-up":    pathPopUp,
		"tooltip":        tooltip,
		"scroll":         scroll,
		"form":           form,
		"tabs":           tabs,
		"table":          table,
		"sidebar":        sidebar,
		"thumbnails":     thumbnails,
		"status-bar":     statusBar,
		"menu":           menu,
		"menu-bar":       menuBar,
		"window":         window,
		"alert":          alert,
		"file-dialog":    fileDialog,
		"hint":           hint,
	}
}
