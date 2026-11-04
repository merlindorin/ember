package gpu

// srgbWGSL is the exact sRGB transfer, both ways, as CSS and color.Hex
// use it (decision 0050): the renderer encodes what it writes with it and
// decodes what it reads back, so a colour round-trips byte for byte.
// Blending stays in encoded space, as browsers blend.
const srgbWGSL = `
fn srgbEncode(c: vec3f) -> vec3f {
	let v = clamp(c, vec3f(0.0), vec3f(1.0));
	return select(1.055 * pow(v, vec3f(1.0 / 2.4)) - 0.055, v * 12.92, v <= vec3f(0.0031308));
}

fn srgbDecode(c: vec3f) -> vec3f {
	let v = clamp(c, vec3f(0.0), vec3f(1.0));
	return select(pow((v + 0.055) / 1.055, vec3f(2.4)), v / 12.92, v <= vec3f(0.04045));
}
`
