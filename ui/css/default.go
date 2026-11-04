package css

import _ "embed"

//go:embed images/select-arrow.png
var selectArrow []byte

// DefaultImages are the images Default names, by url: "ember:select-arrow", the select's
// chevron (a 10×6 PNG).
func DefaultImages() map[string][]byte {
	return map[string][]byte{"ember:select-arrow": selectArrow}
}

// Default is the user-agent sheet: what every element looks like before
// the game's sheets. Export writes the same text into every page, so
// Chrome starts from it too.
const Default = `* { box-sizing: border-box; }
html { font-family: "Go", sans-serif; font-size: 16px; line-height: normal; color: #000; }
html, body, div, section, header, footer, nav, main, aside, article, p, h1, h2, h3, h4,
details, summary, dialog, hr, pre { display: block; }
[hidden] { display: none; }
table { border-spacing: 2px; }
caption { text-align: center; }
html { margin: 0; }
body { margin: 8px; }
p { margin: 1em 0; }
h1 { margin: 0.67em 0; }
h2 { margin: 0.83em 0; }
h3 { margin: 1em 0; }
h4 { margin: 1.33em 0; }
hr { margin: 0.5em auto; border: 1px inset gray; color: gray; overflow: hidden; }
h1 { font-size: 2em; }
h2 { font-size: 1.5em; }
h3 { font-size: 1.17em; }
h4 { font-size: 1em; }
h1, h2, h3, h4, th, strong { font-weight: bold; }
td, th { padding: 1px; }
th { text-align: center; }
em { font-style: italic; }
code, kbd, samp, pre { font-family: "Go Mono", monospace; font-size: 0.8125em; }
pre { margin: 1em 0; white-space: pre; }
button, input, select, textarea {
  appearance: none; margin: 0; padding: 0; border: 0; background: none;
  font-family: inherit; font-size: inherit; font-weight: inherit; font-style: inherit;
  color: inherit; line-height: inherit; letter-spacing: inherit; text-align: inherit;
}
button, input, select, textarea, progress, meter, img, canvas { display: inline-block; }
button, select, input[type="checkbox"], input[type="range"] { cursor: default; }
input, textarea { cursor: text; }
button {
  padding: 1px 6px; border: 1px solid #767676; border-radius: 3px; background-color: #efefef; text-align: center;
}
button:hover { background-color: #e5e5e5; }
button:active { background-color: #f5f5f5; }
input, select, textarea { padding: 1px 2px; border: 1px solid #767676; border-radius: 2px; background-color: #fff; }
::placeholder { color: #757575; }
button:disabled {
  color: rgba(16, 16, 16, 0.3); background-color: rgba(239, 239, 239, 0.3); border-color: rgba(118, 118, 118, 0.3);
}
input:disabled, textarea:disabled {
  color: #545454; background-color: rgba(239, 239, 239, 0.3); border-color: rgba(118, 118, 118, 0.3);
}
select:disabled { color: gray; border-color: rgba(118, 118, 118, 0.3); opacity: 0.7; }
option { display: block; padding: 0 2px 1px; }
option:disabled { color: gray; }
input, select { white-space: nowrap; }
input { overflow: hidden; }
select { overflow: clip; }
textarea { white-space: pre-wrap; overflow: auto; overflow-wrap: break-word; }
select {
  padding-right: 20px; background-image: url(ember:select-arrow); background-repeat: no-repeat;
  background-position: right 6px center;
}
input[type="checkbox"] { width: 13px; height: 13px; padding: 0; border-radius: 2px; }
input[type="checkbox"]:checked { background-color: #0075ff; border-color: #0075ff; color: #fff; }
input[type="radio"] { width: 13px; height: 13px; padding: 0; border-radius: 50%; }
input[type="radio"]:checked { border-color: #0075ff; color: #0075ff; }
/* Disabled controls and the switch in native Chrome's colours, measured from its own controls at 2x (headless Chrome
   through chrome.ScreenshotScaled, 2026-11-03); its switch, unrendered there, follows the checkbox's. */
input[type="checkbox"]:disabled { background-color: #f8f8f8; }
input[type="checkbox"]:checked:disabled { background-color: #d1d1d1; border-color: #d1d1d1; color: #ededed; }
input[type="range"]:disabled { color: #cbcbcb; }
input[type="radio"]:disabled { background-color: #f8f8f8; }
input[type="radio"]:checked:disabled { border-color: #d1d1d1; color: #d1d1d1; }
input[type="checkbox"][switch] {
  width: 32px; height: 18px; border: 1px solid #767676; border-radius: 9px; background-color: #efefef; color: #767676;
}
input[type="checkbox"][switch]:checked { background-color: #0075ff; border-color: #0075ff; color: #fff; }
input[type="checkbox"][switch]:disabled {
  background-color: #f8f8f8; border-color: rgba(118, 118, 118, 0.3); color: #d1d1d1;
}
input[type="checkbox"][switch]:checked:disabled { background-color: #d1d1d1; border-color: #d1d1d1; color: #f8f8f8; }
input[type="range"] { width: 129px; height: 16px; padding: 0; border: 0; background: none; color: #0075ff; }
progress { width: 160px; height: 16px; color: #0075ff; }
meter { width: 80px; height: 16px; }
summary { display: list-item; }
details > summary { list-style-type: disclosure-closed; }
details[open] > summary { list-style-type: disclosure-open; }
details:open > summary { list-style-type: disclosure-open; }
dialog { position: absolute; left: 0; right: 0; margin: auto; width: fit-content; height: fit-content; padding: 1em;
  border: solid; background-color: #fff; color: #000; }
dialog:modal {
  position: fixed; overflow: auto; top: 0; bottom: 0; max-width: calc(100% - 6px - 2em);
  max-height: calc(100% - 6px - 2em);
}
dialog::backdrop { background-color: rgba(0, 0, 0, 0.1); }
dialog:not([open]) { display: none; }
[role="menu"]:not([open]) { display: none; }
:focus-visible { outline: 2px solid #0075ff; outline-offset: 1px; }
`
