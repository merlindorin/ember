package main

// rippleShader pushes the layer's pixels outwards in a ring around each
// live ripple, and lights the ring's crest. uRipples[i] is a ripple's
// centre in the picture (0 to 1 across, y down), its age in seconds, and 1
// while it lives. The ring grows by a third of the height a second and
// fades as it goes; colours are premultiplied, so the light is scaled by
// alpha.
const rippleShader = `
fn effect(vUV: vec2f) -> vec4f {
	var offset = vec2f(0.0);
	var light = 0.0;
	let aspect = uSize.x / uSize.y;
	for (var i = 0; i < 4; i++) {
		let r = uRipples[i];
		if (r.w < 0.5) {
			continue;
		}
		let d = (vUV - r.xy) * vec2f(aspect, 1.0);
		let dist = length(d);
		let ring = dist - r.z * 0.35;
		if (abs(ring) < 0.06 && dist > 0.0) {
			let wave = sin(ring / 0.06 * 3.14159);
			let fade = 1.0 - r.z / 1.6;
			offset += normalize(d) / vec2f(aspect, 1.0) * wave * fade * 0.025;
			light += max(wave, 0.0) * fade;
		}
	}
	let c = color(vUV - offset);
	return vec4f(c.rgb + c.a * light * 0.18, c.a);
}
`
