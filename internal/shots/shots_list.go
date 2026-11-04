package shots

// The packages and arguments many shots share.
const (
	htmlElements   = "./examples/93-html-elements"
	uiKit          = "./examples/92-ui-kit"
	element        = "-element"
	tilesetTool    = "./tools/tileset"
	walls          = "{tmp}/crypt/tiled/walls.tsj"
	tilesetFixture = "tools/tileset/testdata/crypt"
	settings       = "{tmp}/settings"
)

// Shots are the guide's pictures, by their path under docs/. game-doc-writer
// keeps the list; the format is Shot's.
var Shots = []Shot{ //nolint:gochecknoglobals // the guide's list of pictures
	{Out: "01-getting-started/hello-cube.png", Run: "./examples/01-hello-cube", Frame: 60},
	{Out: "03-windows/map.png", Run: "./examples/10-multi-window", Window: 2},
	{Out: "03-windows/scene.png", Run: "./examples/10-multi-window"},
	{Out: "03-windows/window.png", Run: "./examples/09-window"},
	{
		Out:  "05-ui/02-elements/article.png",
		Run:  htmlElements,
		Args: []string{element, "article"},
	},
	{Out: "05-ui/02-elements/aside.png", Run: htmlElements, Args: []string{element, "aside"}},
	{
		Out:  "05-ui/02-elements/button.png",
		Run:  htmlElements,
		Args: []string{element, "button"},
	},
	{
		Out:  "05-ui/02-elements/canvas.png",
		Run:  htmlElements,
		Args: []string{element, "canvas"},
		Env: map[string]string{
			CaptureMouse: "150,110@0",
		},
	},
	{Out: "05-ui/02-elements/code.png", Run: htmlElements, Args: []string{element, "code"}},
	{
		Out:  "05-ui/02-elements/details.png",
		Run:  htmlElements,
		Args: []string{element, "details"},
	},
	{
		Out:  "05-ui/02-elements/dialog.png",
		Run:  htmlElements,
		Args: []string{element, "dialog"},
	},
	{Out: "05-ui/02-elements/div.png", Run: htmlElements, Args: []string{element, "div"}},
	{Out: "05-ui/02-elements/em.png", Run: htmlElements, Args: []string{element, "em"}},
	{
		Out:  "05-ui/02-elements/footer.png",
		Run:  htmlElements,
		Args: []string{element, "footer"},
	},
	{Out: "05-ui/02-elements/h1.png", Run: htmlElements, Args: []string{element, "h1"}},
	{Out: "05-ui/02-elements/h2.png", Run: htmlElements, Args: []string{element, "h2"}},
	{Out: "05-ui/02-elements/h3.png", Run: htmlElements, Args: []string{element, "h3"}},
	{Out: "05-ui/02-elements/h4.png", Run: htmlElements, Args: []string{element, "h4"}},
	{
		Out:  "05-ui/02-elements/header.png",
		Run:  htmlElements,
		Args: []string{element, "header"},
	},
	{Out: "05-ui/02-elements/hr.png", Run: htmlElements, Args: []string{element, "hr"}},
	{Out: "05-ui/02-elements/img.png", Run: htmlElements, Args: []string{element, "img"}},
	{Out: "05-ui/02-elements/input.png", Run: htmlElements, Args: []string{element, "input"}},
	{Out: "05-ui/02-elements/kbd.png", Run: htmlElements, Args: []string{element, "kbd"}},
	{Out: "05-ui/02-elements/label.png", Run: htmlElements, Args: []string{element, "label"}},
	{Out: "05-ui/02-elements/main.png", Run: htmlElements, Args: []string{element, "main"}},
	{Out: "05-ui/02-elements/meter.png", Run: htmlElements, Args: []string{element, "meter"}},
	{Out: "05-ui/02-elements/nav.png", Run: htmlElements, Args: []string{element, "nav"}},
	{
		Out:  "05-ui/02-elements/option.png",
		Run:  htmlElements,
		Args: []string{element, "option"},
	},
	{Out: "05-ui/02-elements/p.png", Run: htmlElements, Args: []string{element, "p"}},
	{Out: "05-ui/02-elements/pre.png", Run: htmlElements, Args: []string{element, "pre"}},
	{
		Out:  "05-ui/02-elements/progress.png",
		Run:  htmlElements,
		Args: []string{element, "progress"},
	},
	{Out: "05-ui/02-elements/samp.png", Run: htmlElements, Args: []string{element, "samp"}},
	{
		Out:  "05-ui/02-elements/section.png",
		Run:  htmlElements,
		Args: []string{element, "section"},
	},
	{
		Out:  "05-ui/02-elements/select.png",
		Run:  htmlElements,
		Args: []string{element, "select"},
	},
	{Out: "05-ui/02-elements/span.png", Run: htmlElements, Args: []string{element, "span"}},
	{
		Out:  "05-ui/02-elements/strong.png",
		Run:  htmlElements,
		Args: []string{element, "strong"},
	},
	{
		Out:  "05-ui/02-elements/summary.png",
		Run:  htmlElements,
		Args: []string{element, "summary"},
	},
	{Out: "05-ui/02-elements/table.png", Run: htmlElements, Args: []string{element, "table"}},
	{
		Out:  "05-ui/02-elements/textarea.png",
		Run:  htmlElements,
		Args: []string{element, "textarea"},
	},
	{Out: "05-ui/07-kit/alert.png", Run: "./ui/internal/pages/kit-31-alert"},
	{Out: "05-ui/07-kit/button.png", Run: "./ui/internal/pages/kit-09-button"},
	{Out: "05-ui/07-kit/check.png", Run: "./ui/internal/pages/kit-10-check"},
	{Out: "05-ui/07-kit/choice.png", Run: "./ui/internal/pages/kit-17-choice"},
	{Out: "05-ui/07-kit/file-dialog-own-window.png", Run: "./ui/internal/pages/kit-32d-file-dialog-own-window"},
	{Out: "05-ui/07-kit/file-dialog-replace.png", Run: "./ui/internal/pages/kit-32c-file-dialog-replace"},
	{Out: "05-ui/07-kit/file-dialog-save.png", Run: "./ui/internal/pages/kit-32b-file-dialog-save"},
	{Out: "05-ui/07-kit/file-dialog.png", Run: "./ui/internal/pages/kit-32-file-dialog"},
	{Out: "05-ui/07-kit/form.png", Run: "./ui/internal/pages/kit-22-form"},
	{Out: "05-ui/07-kit/grid.png", Run: "./ui/internal/pages/kit-07-grid"},
	{Out: "05-ui/07-kit/hint.png", Run: "./ui/internal/pages/kit-33-hint"},
	{Out: "05-ui/07-kit/icons.png", Run: "./ui/internal/pages/kit-03-icons"},
	{Out: "05-ui/07-kit/image.png", Run: "./ui/internal/pages/kit-04-image"},
	{Out: "05-ui/07-kit/label.png", Run: "./ui/internal/pages/kit-01-label"},
	{Out: "05-ui/07-kit/menu-bar.png", Run: "./ui/internal/pages/kit-29-menu-bar"},
	{Out: "05-ui/07-kit/menu.png", Run: "./ui/internal/pages/kit-28-menu"},
	{Out: "05-ui/07-kit/number-field.png", Run: "./ui/internal/pages/kit-16-number-field"},
	{Out: "05-ui/07-kit/panel.png", Run: "./ui/internal/pages/kit-08-panel"},
	{Out: "05-ui/07-kit/path-pop-up.png", Run: "./ui/internal/pages/kit-19-path-pop-up"},
	{Out: "05-ui/07-kit/pop-up.png", Run: "./ui/internal/pages/kit-18-popup"},
	{Out: "05-ui/07-kit/progress.png", Run: "./ui/internal/pages/kit-13-progress"},
	{Out: "05-ui/07-kit/rich-text.png", Run: "./ui/internal/pages/kit-02-rich-text"},
	{Out: "05-ui/07-kit/row-and-column.png", Run: "./ui/internal/pages/kit-06-row-column"},
	{Out: "05-ui/07-kit/scroll.png", Run: "./ui/internal/pages/kit-21-scroll"},
	{Out: "05-ui/07-kit/separator.png", Run: "./ui/internal/pages/kit-05-separator"},
	{Out: "05-ui/07-kit/sidebar.png", Run: "./ui/internal/pages/kit-25-sidebar"},
	{Out: "05-ui/07-kit/slider.png", Run: "./ui/internal/pages/kit-12-slider"},
	{Out: "05-ui/07-kit/status-bar.png", Run: "./ui/internal/pages/kit-27-status-bar"},
	{Out: "05-ui/07-kit/switch.png", Run: "./ui/internal/pages/kit-11-switch"},
	{Out: "05-ui/07-kit/table-scrolled.png", Run: "./ui/internal/pages/kit-24b-table-scrolled"},
	{Out: "05-ui/07-kit/table-spans.png", Run: "./ui/internal/pages/kit-24c-table-spans"},
	{Out: "05-ui/07-kit/table.png", Run: "./ui/internal/pages/kit-24-table"},
	{Out: "05-ui/07-kit/tabs.png", Run: "./ui/internal/pages/kit-23-tabs"},
	{Out: "05-ui/07-kit/text-area.png", Run: "./ui/internal/pages/kit-15-textarea"},
	{Out: "05-ui/07-kit/text-input-search.png", Run: "./ui/internal/pages/kit-14b-search"},
	{Out: "05-ui/07-kit/text-input-shortcodes.png", Run: "./ui/internal/pages/kit-14b-shortcodes"},
	{Out: "05-ui/07-kit/text-input.png", Run: "./ui/internal/pages/kit-14-textinput"},
	{Out: "05-ui/07-kit/thumbnails.png", Run: "./ui/internal/pages/kit-26-thumbnails"},
	{Out: "05-ui/07-kit/tooltip.png", Run: "./ui/internal/pages/kit-20-tooltip"},
	{Out: "05-ui/07-kit/window.png", Run: "./ui/internal/pages/kit-30-window"},
	{
		Out: "05-ui/09-walkthroughs/controls-pad.gif",
		Run: "./examples/13-controls",
		Env: map[string]string{
			CaptureFrames: "120",
			CaptureKeys:   "enter:205-208",
			"EMBER_CAPTURE_PAD": "connect:0-240;right:20-23;down:50-53;south:80-83;left:110-113;" +
				"south:140-143;east:170-173",
		},
	},
	{
		Out: "05-ui/09-walkthroughs/controls.gif",
		Run: "./examples/13-controls",
		Env: map[string]string{
			CaptureFrames: "200",
			CaptureKeys:   "enter:355-358",
			CaptureMouse: "320,200@0;320,244@15;320,244@25;left:18-21;290,210@40;290,210@52;" +
				"left:44-47;398,150@65;398,150@78;left:78-112;300,150@108;300,150@115;" +
				"320,307@125;320,307@160;wheel:3@135;wheel:3@150;320,41@170;320,41@182;" +
				"left:182-215;240,31@210;240,31@218;428,383@235;428,383@245;" +
				"left:245-275;520,418@270;520,418@280;65,39@290;65,39@320;left:300-310;" +
				"300,250@335",
		},
	},
	{Out: "05-ui/09-walkthroughs/counter.png", Run: "./examples/11-ui-elements"},
	{
		Out: "05-ui/09-walkthroughs/hud.gif",
		Run: "./examples/16-hud-over-world",
		Env: map[string]string{
			CaptureFrames: "90",
			CaptureMouse: "400,120@20;400,120@30;left:30-32;400,180@80;400,180@90;left:90-92;" +
				"240,120@110;240,120@120;left:120-122;320,360@150",
		},
	},
	{
		Out: "05-ui/09-walkthroughs/layout.gif",
		Run: "./examples/12-layout",
		Env: map[string]string{
			CaptureFrames:        "80",
			"EMBER_CAPTURE_SIZE": "640,400@0;900,520@50;900,520@70;420,320@130;420,320@160",
		},
	},
	{
		Out:   "05-ui/09-walkthroughs/no-wrap.png",
		Run:   "./examples/14-text-editing",
		Frame: 60,
		Env: map[string]string{
			CaptureKeys: "tab:10-12;space:25-27",
		},
	},
	{Out: "05-ui/09-walkthroughs/orb.png", Run: "./examples/17-custom-elements"},
	{Out: "05-ui/09-walkthroughs/testing-ui.png", Run: "./examples/18-testing-ui"},
	{
		Out: "05-ui/09-walkthroughs/text-editing.gif",
		Run: "./examples/14-text-editing",
		Env: map[string]string{
			CaptureFrames: "75",
			CaptureKeys: "leftsuper:2-8;down:4-6;enter:16-18;enter:24-26;enter:32-34;enter:40-42;" +
				"enter:48-50;enter:56-58;enter:64-66;enter:72-74;leftsuper:100-114;" +
				"a:104-108",
		},
	},
	{
		Out: "05-ui/09-walkthroughs/themes.gif",
		Run: "./examples/15-themes",
		Env: map[string]string{
			CaptureFrames: "80",
			CaptureKeys:   "t:40-42;t:80-82;t:120-124",
		},
	},
	{
		Out:   "05-ui/focus.png",
		Run:   uiKit,
		Args:  []string{element, "settings"},
		Frame: 45,
		Crop:  [4]int{0, 0, 824, 472},
		Size:  [2]int{412, 236},
		Env: map[string]string{
			CaptureKeys:  "tab:10-11;tab:15-16;tab:20-21;tab:25-26",
			CaptureMouse: "230,88@0;230,88@6;left:3-4",
		},
	},
	{
		Out:  "05-ui/settings.png",
		Run:  uiKit,
		Args: []string{element, "settings"},
		Crop: [4]int{0, 0, 824, 472},
		Size: [2]int{412, 236},
	},
	{
		Out:  "05-ui/styled.png",
		Run:  uiKit,
		Args: []string{element, "styled"},
		Crop: [4]int{0, 0, 624, 212},
		Size: [2]int{312, 106},
	},
	{Out: "06-2d/19-lights/affects.png", Panels: []string{"affects-all", "affects-floor"}},
	{Out: "06-2d/19-lights/budget.png", Frame: 40, Panels: []string{"budget-none", "budget-3"}},
	{Out: "06-2d/19-lights/cone-angle.png", Panels: []string{"angle-0", "angle-up", "angle-down"}},
	{Out: "06-2d/19-lights/cone-size.png", Panels: []string{"cone-size-0", "cone-size-32"}},
	{Out: "06-2d/19-lights/cone-spread.png", Panels: []string{"spread-0.2", "spread-0.5", "spread-1.2"}},
	{
		Out: "06-2d/19-lights/cone.gif",
		Run: "./examples/38-cone-lights",
		Env: map[string]string{
			CaptureEvery:  "4",
			CaptureFrames: "45",
			CaptureMouse:  "520,120@0;520,120@10;600,330@60;340,370@100;100,320@140;140,80@180",
		},
	},
	{Out: "06-2d/19-lights/debug-views.png", Panels: []string{"debug-light", "debug-occluders", "debug-shadows"}},
	{
		Out: "06-2d/19-lights/directional.gif",
		Run: "./examples/39-directional-lights",
		Env: map[string]string{
			CaptureEvery:  "29",
			CaptureFrames: "34",
		},
	},
	{Out: "06-2d/19-lights/exposure.png", Panels: []string{"exposure-0.6", "exposure-1", "exposure-1.6"}},
	{Out: "06-2d/19-lights/lit-ambient-map.png", Panels: []string{"ambient-0", "ambient-map"}},
	{Out: "06-2d/19-lights/lit-ambient.png", Panels: []string{"ambient-0", "ambient-0.15", "ambient-0.4"}},
	{Out: "06-2d/19-lights/lit-layer.png", Panels: []string{"lit-off", "lit-on"}},
	{Out: "06-2d/19-lights/lit-perpixel.png", Panels: []string{"perpixel-off", "perpixel-on"}},
	{Out: "06-2d/19-lights/lit-unlit.png", Panels: []string{"unlit-off", "unlit-on"}},
	{Out: "06-2d/19-lights/occluder-groups.png", Panels: []string{"groups-all", "groups-blockedby"}},
	{
		Out:    "06-2d/19-lights/parts.png",
		Panels: []string{"parts-ambient", "parts-light", "parts-shadows", "parts-unlit"},
	},
	{Out: "06-2d/19-lights/point-color.png", Panels: []string{"color-warm", "color-cold", "color-green"}},
	{Out: "06-2d/19-lights/point-falloff.png", Panels: []string{"falloff-0", "falloff-0.5", "falloff-1"}},
	{
		Out:    "06-2d/19-lights/point-intensity.png",
		Panels: []string{"intensity-0.25", "intensity-1", "intensity-2"},
	},
	{Out: "06-2d/19-lights/point-radius.png", Panels: []string{"radius-80", "radius-140", "radius-220"}},
	{Out: "06-2d/19-lights/point-shadows-off.png", Panels: []string{"point-shadows-on", "point-shadows-off"}},
	{
		Out: "06-2d/19-lights/point.gif",
		Run: "./examples/36-point-lights",
		Env: map[string]string{
			CaptureEvery:  "4",
			CaptureFrames: "36",
			CaptureKeys:   "left:0-45;up:45-70;right:70-160",
		},
	},
	{
		Out: "06-2d/19-lights/probe.gif",
		Run: "./examples/41-probes",
		Env: map[string]string{
			CaptureEvery:  "7",
			CaptureFrames: "26",
			CaptureKeys:   "left:10-70;down:70-110;right:110-182",
		},
	},
	{Out: "06-2d/19-lights/shaded-by.png", Panels: []string{"shadedby-all", "shadedby-0-1"}},
	{
		Out:    "06-2d/19-lights/shadow-color.png",
		Panels: []string{"shadow-black", "shadow-color", "shadow-opacity", "shadow-both"},
	},
	{
		Out:    "06-2d/19-lights/shadow-elevation.png",
		Panels: []string{"elevation-0", "elevation-100", "elevation-200"},
	},
	{Out: "06-2d/19-lights/shadow-fade-from.png", Panels: []string{"fade-linear", "fade-from"}},
	{
		Out:    "06-2d/19-lights/shadow-fade.png",
		Panels: []string{"fade-off", "fade-linear", "fade-smooth", "fade-soft"},
	},
	{Out: "06-2d/19-lights/shadow-height.png", Panels: []string{"height-24", "height-8"}},
	{Out: "06-2d/19-lights/shadow-occluders.png", Panels: []string{"occluders-off", "occluders-on"}},
	{Out: "06-2d/19-lights/shadow-size.png", Panels: []string{"size-0", "size-32"}},
	{
		Out: "06-2d/19-lights/shadows.gif",
		Run: "./examples/37-shadows",
		Env: map[string]string{
			CaptureEvery:  "7",
			CaptureFrames: "38",
			CaptureKeys:   "up:80-130;s:150-151;f:175-176;f:205-206;f:235-236",
			CaptureMouse:  "250,200@0;420,240@70;420,240@150;220,150@264",
		},
	},
	{Out: "06-2d/19-lights/shape-emissive.png", Panels: []string{"emissive-off", "emissive-on"}},
	{
		Out:    "06-2d/19-lights/shape-kinds.png",
		Panels: []string{"shape-rect", "shape-circle", "shape-capsule", "shape-line", "shape-polygon"},
	},
	{Out: "06-2d/19-lights/shape-reach.png", Panels: []string{"reach-8", "reach-24", "reach-80"}},
	{Out: "06-2d/19-lights/shape-shadows.png", Panels: []string{"shadows-off", "shadows-on"}},
	{
		Out: "06-2d/19-lights/shape.gif",
		Run: "./examples/40-shape-lights",
		Env: map[string]string{
			CaptureEvery:  "4",
			CaptureFrames: "55",
			CaptureKeys:   "down:0-50;right:50-150;up:150-175",
		},
	},
	{Out: "06-2d/19-lights/silhouette.png", Panels: []string{"silhouette-box", "silhouette-on"}},
	{Out: "06-2d/19-lights/static.png", Panels: []string{"static-off", "static-on"}},
	{Out: "06-2d/19-lights/sun-angle.png", Panels: []string{"sun-angle-0.2", "sun-angle-0.5", "sun-angle-0.8"}},
	{Out: "06-2d/19-lights/sun-day.png", Panels: []string{"day-dawn", "day-noon", "day-dusk"}},
	{
		Out:    "06-2d/19-lights/sun-length.png",
		Panels: []string{"sun-length-40", "sun-length-120", "sun-length-256"},
	},
	{Out: "06-2d/19-lights/thumb-affects-floor.png", Size: [2]int{160, 120}, Panels: []string{"affects-floor"}},
	{Out: "06-2d/19-lights/thumb-ambient-0.15.png", Size: [2]int{160, 120}, Panels: []string{"ambient-0.15"}},
	{Out: "06-2d/19-lights/thumb-ambient-map.png", Size: [2]int{160, 120}, Panels: []string{"ambient-map"}},
	{Out: "06-2d/19-lights/thumb-angle-up.png", Size: [2]int{160, 120}, Panels: []string{"angle-up"}},
	{Out: "06-2d/19-lights/thumb-color-cold.png", Size: [2]int{160, 120}, Panels: []string{"color-cold"}},
	{Out: "06-2d/19-lights/thumb-cone-size-32.png", Size: [2]int{160, 120}, Panels: []string{"cone-size-32"}},
	{Out: "06-2d/19-lights/thumb-day-noon.png", Size: [2]int{160, 120}, Panels: []string{"day-noon"}},
	{
		Out:    "06-2d/19-lights/thumb-debug-occluders.png",
		Size:   [2]int{160, 120},
		Panels: []string{"debug-occluders"},
	},
	{Out: "06-2d/19-lights/thumb-elevation-100.png", Size: [2]int{160, 120}, Panels: []string{"elevation-100"}},
	{Out: "06-2d/19-lights/thumb-emissive-on.png", Size: [2]int{160, 120}, Panels: []string{"emissive-on"}},
	{Out: "06-2d/19-lights/thumb-exposure-1.6.png", Size: [2]int{160, 120}, Panels: []string{"exposure-1.6"}},
	{Out: "06-2d/19-lights/thumb-fade-from.png", Size: [2]int{160, 120}, Panels: []string{"fade-from"}},
	{Out: "06-2d/19-lights/thumb-fade-smooth.png", Size: [2]int{160, 120}, Panels: []string{"fade-smooth"}},
	{Out: "06-2d/19-lights/thumb-falloff-0.5.png", Size: [2]int{160, 120}, Panels: []string{"falloff-0.5"}},
	{
		Out:    "06-2d/19-lights/thumb-groups-blockedby.png",
		Size:   [2]int{160, 120},
		Panels: []string{"groups-blockedby"},
	},
	{Out: "06-2d/19-lights/thumb-height-8.png", Size: [2]int{160, 120}, Panels: []string{"height-8"}},
	{Out: "06-2d/19-lights/thumb-intensity-2.png", Size: [2]int{160, 120}, Panels: []string{"intensity-2"}},
	{Out: "06-2d/19-lights/thumb-lit-on.png", Size: [2]int{160, 120}, Panels: []string{"lit-on"}},
	{Out: "06-2d/19-lights/thumb-occluders-on.png", Size: [2]int{160, 120}, Panels: []string{"occluders-on"}},
	{Out: "06-2d/19-lights/thumb-perpixel-on.png", Size: [2]int{160, 120}, Panels: []string{"perpixel-on"}},
	{
		Out:    "06-2d/19-lights/thumb-point-shadows-off.png",
		Size:   [2]int{160, 120},
		Panels: []string{"point-shadows-off"},
	},
	{
		Out:   "06-2d/19-lights/thumb-probes.png",
		Run:   "./examples/41-probes",
		Frame: 141,
		Size:  [2]int{160, 96},
		Env: map[string]string{
			CaptureKeys: "left:10-70;down:70-110;right:110-182",
		},
	},
	{Out: "06-2d/19-lights/thumb-radius-140.png", Size: [2]int{160, 120}, Panels: []string{"radius-140"}},
	{Out: "06-2d/19-lights/thumb-radius-220.png", Size: [2]int{160, 120}, Panels: []string{"radius-220"}},
	{Out: "06-2d/19-lights/thumb-reach-80.png", Size: [2]int{160, 120}, Panels: []string{"reach-80"}},
	{Out: "06-2d/19-lights/thumb-shadedby-0-1.png", Size: [2]int{160, 120}, Panels: []string{"shadedby-0-1"}},
	{Out: "06-2d/19-lights/thumb-shadow-color.png", Size: [2]int{160, 120}, Panels: []string{"shadow-color"}},
	{Out: "06-2d/19-lights/thumb-shadows-on.png", Size: [2]int{160, 120}, Panels: []string{"shadows-on"}},
	{Out: "06-2d/19-lights/thumb-shape-polygon.png", Size: [2]int{160, 120}, Panels: []string{"shape-polygon"}},
	{Out: "06-2d/19-lights/thumb-shape-rect.png", Size: [2]int{160, 120}, Panels: []string{"shape-rect"}},
	{Out: "06-2d/19-lights/thumb-silhouette-on.png", Size: [2]int{160, 120}, Panels: []string{"silhouette-on"}},
	{Out: "06-2d/19-lights/thumb-size-32.png", Size: [2]int{160, 120}, Panels: []string{"size-32"}},
	{Out: "06-2d/19-lights/thumb-spread-0.5.png", Size: [2]int{160, 120}, Panels: []string{"spread-0.5"}},
	{Out: "06-2d/19-lights/thumb-spread-1.2.png", Size: [2]int{160, 120}, Panels: []string{"spread-1.2"}},
	{Out: "06-2d/19-lights/thumb-sun-angle-0.2.png", Size: [2]int{160, 120}, Panels: []string{"sun-angle-0.2"}},
	{Out: "06-2d/19-lights/thumb-sun-length-120.png", Size: [2]int{160, 120}, Panels: []string{"sun-length-120"}},
	{Out: "06-2d/19-lights/thumb-unlit-on.png", Size: [2]int{160, 120}, Panels: []string{"unlit-on"}},
	{
		Out:   "06-2d/19-lights/tileset-tool-room.png",
		Run:   tilesetTool,
		Args:  []string{"-room", "{tmp}/dungeon/rooms/crypt.tmj", walls},
		Crop:  [4]int{1680, 994, 820, 460},
		Size:  [2]int{410, 230},
		Files: []string{tilesetFixture, "tools/tileset/testdata/dungeon"},
		Write: map[string]string{"settings/state.json": "{\"folded\":{\"preview-section\":false}}"},
		Env: map[string]string{
			TilesetDir: settings,
		},
	},
	{
		Out:   "06-2d/19-lights/tileset-tool.png",
		Run:   tilesetTool,
		Args:  []string{walls},
		Files: []string{tilesetFixture},
		Env: map[string]string{
			CaptureMouse: "900,384@5;left:8-9",
			TilesetDir:   settings,
		},
	},
	{
		Out: "06-2d/2d-shader.gif",
		Run: "./examples/33-2d-shader",
		Env: map[string]string{
			CaptureFrames: "64",
			CaptureMouse: "180,150@0;180,150@6;left:4-6;440,230@40;440,230@46;left:44-46;" +
				"300,300@80;300,300@86;left:84-86",
		},
	},
	{
		Out: "06-2d/blend-modes.gif",
		Run: "./examples/31-blend-modes",
		Env: map[string]string{
			CaptureEvery:  "3",
			CaptureFrames: "60",
			CaptureMouse:  "320,192@0;140,120@40;500,110@100;470,300@150;320,192@180",
		},
	},
	{
		Out: "06-2d/bunnies.gif",
		Run: "./examples/24-many-sprites",
		Env: map[string]string{
			CaptureEvery:  "6",
			CaptureFrames: "20",
			CaptureKeys:   "space:40-70",
		},
	},
	{
		Out:   "06-2d/clipping.gif",
		Run:   "./examples/25-clipping",
		Frame: 200,
		Env: map[string]string{
			CaptureFrames: "60",
		},
	},
	{
		Out: "06-2d/collision.gif",
		Run: "./examples/27-collision",
		Env: map[string]string{
			CaptureEvery:  "3",
			CaptureFrames: "60",
			CaptureKeys:   "right:0-180",
		},
	},
	{
		Out: "06-2d/easing.gif",
		Run: "./examples/35-easing",
		Env: map[string]string{
			CaptureEvery:  "2",
			CaptureFrames: "60",
		},
	},
	{
		Out: "06-2d/follow.gif",
		Run: "./examples/23-2d-camera",
		Env: map[string]string{
			CaptureFrames: "60",
			CaptureKeys:   "right:0-60;left:60-120",
		},
	},
	{Out: "06-2d/lines.png", Run: "./examples/21-lines-and-shapes", Frame: 10},
	{Out: "06-2d/nine-patch.png", Run: "./examples/22-nine-patch", Frame: 10},
	{
		Out: "06-2d/particles.gif",
		Run: "./examples/34-particles",
		Env: map[string]string{
			CaptureFrames: "90",
			CaptureMouse:  "130,110@0;130,110@40;left:30-32;360,90@60;360,90@180;left:90-92",
		},
	},
	{Out: "06-2d/pixel-perfect.png", Run: "./examples/30-pixel-perfect"},
	{Out: "06-2d/render-to-texture.png", Run: "./examples/32-render-to-texture"},
	{Out: "06-2d/sprites.png", Run: "./examples/19-sprites"},
	{
		Out:   "06-2d/tile-properties.png",
		Run:   tilesetTool,
		Args:  []string{walls},
		Crop:  [4]int{1676, 148, 864, 540},
		Size:  [2]int{432, 270},
		Files: []string{tilesetFixture},
		Env: map[string]string{
			CaptureMouse: "900,384@5;left:8-9",
			TilesetDir:   settings,
		},
	},
	{
		Out: "06-2d/tiled-pictures.gif",
		Run: "./examples/42-tiled-pictures",
		Env: map[string]string{
			CaptureEvery:  "4",
			CaptureFrames: "50",
			CaptureKeys:   "left:5-50;down:50-80;right:80-160;up:160-200",
		},
	},
	{
		Out: "06-2d/tiled.gif",
		Run: "./examples/29-tiled",
		Env: map[string]string{
			CaptureEvery:  "4",
			CaptureFrames: "30",
			CaptureKeys:   "up:0-50;right:50-120",
		},
	},
	{
		Out: "06-2d/tilemap.gif",
		Run: "./examples/26-tilemap",
		Env: map[string]string{
			CaptureEvery:  "8",
			CaptureFrames: "16",
		},
	},
	{
		Out: "06-2d/walk.gif",
		Run: "./examples/20-sprite-sheet",
		Env: map[string]string{
			CaptureFrames: "75",
			CaptureKeys:   "right:0-75;left:75-150",
		},
	},
	{
		Out: "06-2d/y-sort.gif",
		Run: "./examples/28-y-sort",
		Env: map[string]string{
			CaptureEvery:  "4",
			CaptureFrames: "24",
			CaptureKeys:   "up:0-20;left:20-34;up:34-54;right:54-68",
		},
	},
	{
		Out: "07-text/bitmap-font.gif",
		Run: "./examples/44-bitmap-font",
		Env: map[string]string{
			CaptureFrames: "60",
		},
	},
	{
		Out:   "07-text/emoji.png",
		Run:   "./examples/45-emoji",
		Frame: 140,
		Env: map[string]string{
			CaptureText: "gg 🎉 well played 👍🏽@10",
		},
	},
	{
		Out:   "07-text/shortcodes.png",
		Run:   "./examples/45-emoji",
		Frame: 60,
		Env: map[string]string{
			CaptureText: "gg :fi@10",
		},
	},
	{Out: "07-text/text.png", Run: "./examples/43-text", Frame: 120},
	{
		Out: "08-input/actions.gif",
		Run: "./examples/48-actions",
		Env: map[string]string{
			CaptureFrames: "125",
			CaptureKeys:   "d:0-50;s:50-80;enter:205-207",
			CaptureMouse: "500,320@0;430,320@90;left:95-97;190,330@130;left:135-137;wheel:-3@160;" +
				"wheel:-3@175;470,130@200;wheel:3@225;wheel:3@235",
		},
	},
	{
		Out: "08-input/gamepads.gif",
		Run: "./examples/49-gamepads",
		Env: map[string]string{
			CaptureEvery:  "3",
			CaptureFrames: "100",
			"EMBER_CAPTURE_PAD": "connect:10-300;south:30-40;east:45-55;west:60-70;north:75-85;" +
				"lefttrigger:0@90;lefttrigger:1@120;lefttrigger:0@135;" +
				"righttrigger:0@120;righttrigger:1@150;righttrigger:0@160;" +
				"leftstick:0,0@0;leftstick:0,0@160;leftstick:1,0@190;" +
				"leftstick:0.7,-0.7@220;leftstick:0,0@240;rightstick:0,0@240;" +
				"rightstick:-1,0.3@260;rightstick:0,0@280;leftshoulder:250-262;" +
				"1/connect:200-300;1/north:230-260;1/rightstick:0,0@200;" +
				"1/rightstick:0.8,0.8@240",
		},
	},
	{
		Out: "08-input/gestures.gif",
		Run: "./examples/51-gestures",
		Env: map[string]string{
			CaptureFrames: "90",
			"EMBER_CAPTURE_TOUCH": "pinch:0.3@10-40;rotate:1.0@50-80;1:0.4,0.5@95;2:0.6,0.5@95;" +
				"1:0.48,0.42@125;2:0.68,0.42@125;1:up@130;2:up@130;pinch:-0.3@140-165",
		},
	},
	{
		Out: "08-input/keyboard.gif",
		Run: "./examples/46-keyboard",
		Env: map[string]string{
			CaptureFrames: "125",
			CaptureKeys: "d:0-50;leftshift:60-95;a:60-95;enter:110-111;backspace:118-119;" +
				"backspace:124-125;backspace:130-131;backspace:136-137;enter:200-201;" +
				"s:210-240",
			CaptureText: "Heron@150",
		},
	},
	{
		Out: "08-input/mouse.gif",
		Run: "./examples/47-mouse",
		Env: map[string]string{
			CaptureFrames: "145",
			CaptureMouse: "300,300@0;200,180@20;left:25-70;200,180@28;200,290@65;440,180@90;" +
				"left:95-140;440,180@98;530,90@135;400,250@150;back:160-162;" +
				"back:185-187;forward:210-212;wheel:-3@230;wheel:-3@240;wheel:3@265;" +
				"wheel:3@275",
		},
	},
	{Out: "08-input/move-cursor.png", Copy: "examples/47-mouse/move@2x.png"},
	{
		Out: "08-input/touch.gif",
		Run: "./examples/50-touch",
		Env: map[string]string{
			CaptureFrames: "80",
			"EMBER_CAPTURE_TOUCH": "1:0.3,0.6@10;1:0.45,0.4@50;1:0.45,0.4@70;2:0.7,0.5@40;2:0.6,0.7@90;" +
				"1:up@100;2:0.5,0.5@120;2:up@140",
		},
	},
	{
		Out: "09-audio/effects.gif",
		Run: "./examples/54-audio-effects",
		Env: map[string]string{
			CaptureEvery:  "3",
			CaptureFrames: "100",
			CaptureKeys:   "d:0-200;w:200-232;e:234-236",
		},
	},
	{
		Out: "09-audio/music-and-positional.gif",
		Run: "./examples/53-music-and-positional",
		Env: map[string]string{
			CaptureFrames: "100",
			CaptureKeys:   "e:0-120;d:125-200",
		},
	},
	{
		Out: "09-audio/play-a-sound.gif",
		Run: "./examples/52-play-a-sound",
		Env: map[string]string{
			CaptureFrames: "75",
			CaptureKeys:   "space:20-22;space:110-112",
			CaptureMouse: "80,180@0;80,180@40;left:45-47;560,120@70;left:75-77;560,120@80;" +
				"300,280@120;left:135-137",
		},
	},
	{
		Out:  "10-games/tetris.png",
		Run:  "./games/tetris",
		Args: []string{"-seed", "7"},
		Env: map[string]string{
			"EMBER_GAMES_DIR": "{tmp}",
			CaptureKeys: "left:10-11;left:14-15;left:18-19;left:22-23;space:26-27;left:30-31;space:34-35;right:38-39;" +
				"right:42-43;space:46-47;right:50-51;right:54-55;right:58-59;right:62-63;space:66-67;up:100-101;" +
				"left:104-105;left:108-109;left:112-113;space:116-117;up:120-121;left:124-125;space:128-129;" +
				"up:132-133;right:136-137;right:140-141;right:144-145;space:148-149;up:152-153;right:156-157;" +
				"space:160-161;up:164-165;left:168-169;left:172-173;space:176-177;up:180-181;up:184-185;" +
				"left:188-189;space:192-193;up:196-197;left:200-201;left:204-205;left:208-209;left:212-213;" +
				"space:216-217;up:220-221;up:224-225;up:228-229;right:232-233;right:236-237;right:240-241;" +
				"right:244-245;right:248-249;space:252-253",
		},
		Frame: 257,
	},
	{
		Out:  "10-games/2048.png",
		Run:  "./games/2048",
		Args: []string{"-seed", "3"},
		Env: map[string]string{
			"EMBER_GAMES_DIR": "{tmp}",
			CaptureKeys: "left:5-6;down:17-18;right:29-30;down:41-42;left:53-54;down:65-66;right:77-78;down:89-90;" +
				"left:101-102;down:113-114;right:125-126;down:137-138;left:149-150;down:161-162;right:173-174;" +
				"down:185-186;left:197-198;down:209-210;right:221-222;down:233-234;left:245-246;down:257-258;" +
				"right:269-270;down:281-282;left:293-294;down:305-306;right:317-318;down:329-330;left:341-342;" +
				"down:353-354;right:365-366;down:377-378;left:389-390",
		},
		Frame: 420,
	},
}
