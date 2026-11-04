package main

import "embed"

//go:embed frost.css wood.css wood.png fonts
var files embed.FS

// emberLook is a warm look written in Go: the kit's palette, its CSS custom
// properties, set again for a dialog of class "ember", and a heavier frame.
const emberLook = `
.ember {
  --panel-color: #2a1a12;
  --border-color: #b5652a;
  --button-color: #4a2a18;
  --hover-color: #6a3a20;
  --accent-color: #ff9a3c;
  color: #f2e6c9;
  font-size: 17px;
  padding: 20px;
  border-width: 2px;
  border-radius: 4px;
}`
