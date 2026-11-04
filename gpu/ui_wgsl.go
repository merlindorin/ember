package gpu

import "strings"

// uiShader is the 2D shader, reading the light picture alone or, shaded,
// the light picture each vertex's occluder groups pick: the window's
// lists drew something ShadedBy fewer groups.
func uiShader(shaded bool) string {
	if shaded {
		return strings.Replace(uiWGSL, "{{light}}", shadedLightWGSL, 1)
	}
	return strings.Replace(uiWGSL, "{{light}}", plainLightWGSL, 1)
}

// plainLightWGSL reads the light picture alone.
const plainLightWGSL = `	// lightSpread says how the light is read: (0, 0) not at all, x −1 at
	// lightUV, else round it, all four sides, or all but the top when y is
	// negative.
	let spread = in.lightSpread;
	var light = textureSampleLevel(lightPicture, lightSampler, in.lightUV, 0.0).rgb;
	if (spread.x > 0.0) {
		let s = vec2f(spread.x, abs(spread.y));
		light = max(max(textureSampleLevel(lightPicture, lightSampler, in.lightUV + vec2f(s.x, 0.0), 0.0).rgb,
			textureSampleLevel(lightPicture, lightSampler, in.lightUV - vec2f(s.x, 0.0), 0.0).rgb),
			textureSampleLevel(lightPicture, lightSampler, in.lightUV + vec2f(0.0, s.y), 0.0).rgb);
		if (spread.y > 0.0) {
			light = max(light, textureSampleLevel(lightPicture, lightSampler, in.lightUV - vec2f(0.0, s.y), 0.0).rgb);
		}
	}
`

// shadedLightWGSL reads the light picture the vertex's groups pick.
const shadedLightWGSL = `	// lightSpread says how the light is read: (0, 0) not at all, x −1 − k
	// at lightUV, else round it by x − 2k, all four sides, or all but the
	// top when y is negative; k is the occluder groups left out, which
	// picks the light picture.
	let spread = in.lightSpread;
	var left = 0.0;
	var sx = spread.x;
	if (spread.x < 0.0) {
		left = round(-spread.x - 1.0);
	} else if (spread.x > 0.0) {
		left = floor(spread.x / 2.0);
		sx = spread.x - 2.0 * left;
	}
	let k = u32(left) & 127u;
	let picture = u32(shadeMap[k / 4u][k % 4u] + 0.5);
	var light = lightAt(picture, in.lightUV);
	if (spread.x > 0.0) {
		let s = vec2f(sx, abs(spread.y));
		light = max(max(lightAt(picture, in.lightUV + vec2f(s.x, 0.0)), lightAt(picture, in.lightUV - vec2f(s.x, 0.0))),
			lightAt(picture, in.lightUV + vec2f(0.0, s.y)));
		if (spread.y > 0.0) {
			light = max(light, lightAt(picture, in.lightUV - vec2f(0.0, s.y)));
		}
	}
`

// uiWGSL draws the 2D lists: shapes, images, masks and blends, the
// colours written in sRGB (decision 0050). Samples are taken at level 0 everywhere,
// as the branches on a vertex's kind aren't uniform control flow.
const uiWGSL = srgbWGSL + `
struct Draw {
	screen: vec2f,
	scale: f32,
	time: f32,
	hdr: f32,
	exposure: f32,
	masked: f32,
	blend: f32,
	// targets has bit n set when the batch's image n holds encoded,
	// premultiplied colour (a pass's target, a glow, an Encoded texture),
	// decoded after filtering.
	targets: u32,
	// round is the batch's rounded clip in points (x, y, w, h), zero w for
	// none, and radii its corners' radii: top-left, top-right,
	// bottom-right, bottom-left (decision 0058).
	round: vec4f,
	radii: vec4f,
};

@group(0) @binding(0) var<uniform> u: Draw;

@group(1) @binding(0) var atlas: texture_2d<f32>;
@group(1) @binding(1) var mask: texture_2d<f32>;
@group(1) @binding(2) var image0: texture_2d<f32>;
@group(1) @binding(3) var image1: texture_2d<f32>;
@group(1) @binding(4) var image2: texture_2d<f32>;
@group(1) @binding(5) var image3: texture_2d<f32>;
@group(1) @binding(6) var image4: texture_2d<f32>;
@group(1) @binding(7) var image5: texture_2d<f32>;
@group(1) @binding(8) var image6: texture_2d<f32>;
@group(1) @binding(9) var image7: texture_2d<f32>;
@group(1) @binding(10) var atlasSampler: sampler;
@group(1) @binding(11) var maskSampler: sampler;
@group(1) @binding(12) var sampler0: sampler;
@group(1) @binding(13) var sampler1: sampler;
@group(1) @binding(14) var sampler2: sampler;
@group(1) @binding(15) var sampler3: sampler;
@group(1) @binding(16) var sampler4: sampler;
@group(1) @binding(17) var sampler5: sampler;
@group(1) @binding(18) var sampler6: sampler;
@group(1) @binding(19) var sampler7: sampler;

// lightPicture is the window's light picture, white when nothing lights it;
// shaded1 to 4 are those of drawings ShadedBy fewer occluder groups or
// lit by fewer lights (Affects), and shadeMap says which of them each key
// reads, 0 for lightPicture.
@group(2) @binding(0) var lightPicture: texture_2d<f32>;
@group(2) @binding(1) var lightSampler: sampler;
@group(2) @binding(2) var shaded1: texture_2d<f32>;
@group(2) @binding(3) var shaded2: texture_2d<f32>;
@group(2) @binding(4) var shaded3: texture_2d<f32>;
@group(2) @binding(5) var<uniform> shadeMap: array<vec4f, 32>;
@group(2) @binding(6) var shaded4: texture_2d<f32>;
// exposure.x scales the light read, the camera's Exposure.
@group(2) @binding(7) var<uniform> exposure: vec4f;

fn lightAt(picture: u32, uv: vec2f) -> vec3f {
	switch picture {
		case 1u: { return textureSampleLevel(shaded1, lightSampler, uv, 0.0).rgb; }
		case 2u: { return textureSampleLevel(shaded2, lightSampler, uv, 0.0).rgb; }
		case 3u: { return textureSampleLevel(shaded3, lightSampler, uv, 0.0).rgb; }
		case 4u: { return textureSampleLevel(shaded4, lightSampler, uv, 0.0).rgb; }
		default: { return textureSampleLevel(lightPicture, lightSampler, uv, 0.0).rgb; }
	}
}

struct VertexOut {
	@builtin(position) pos: vec4f,
	@location(0) uv: vec2f,
	@location(1) color: vec4f,
	@location(2) @interpolate(flat) shape: vec4f,
	@location(3) @interpolate(flat) extra: f32,
	@location(4) lightUV: vec2f,
	@location(5) @interpolate(flat) lightSpread: vec2f,
};

@vertex
fn vs(@location(0) pos: vec2f, @location(1) uv: vec2f, @location(2) color: vec4f, @location(3) shape: vec4f,
	@location(4) extra: f32, @location(5) lightUV: vec2f, @location(6) lightSpread: vec2f) -> VertexOut {
	var out: VertexOut;
	let p = pos / u.screen * 2.0 - 1.0;
	out.pos = vec4f(p.x, -p.y, 0.0, 1.0);
	out.uv = uv;
	out.color = color;
	out.shape = shape;
	out.extra = extra;
	out.lightUV = lightUV;
	out.lightSpread = lightSpread;
	return out;
}

fn acesInverse(y: vec3f) -> vec3f {
	let a = 2.43 * y - 2.51;
	let b = 0.59 * y - 0.03;
	let c = 0.14 * y;
	return (-b - sqrt(b * b - 4.0 * a * c)) / (2.0 * a);
}

fn roundRect(p: vec2f, hs: vec2f, radius: f32) -> f32 {
	let r = min(radius, min(hs.x, hs.y));
	let q = abs(p) - (hs - r);
	return length(max(q, vec2f(0.0))) + min(max(q.x, q.y), 0.0) - r;
}

fn cover(d: f32) -> f32 { return clamp(0.5 - d * u.scale, 0.0, 1.0); }

// roundedClip is how much of the pixel at pos the batch's rounded clip
// leaves: its corner's radius picked by the quadrant pos is in. pos is in
// pixels and scale is x's points-to-pixels; passes scale both axes alike.
fn roundedClip(pos: vec2f) -> f32 {
	let h = u.round.zw * 0.5;
	let q = pos / u.scale - (u.round.xy + h);
	var r = u.radii.x;
	if (q.x >= 0.0) {
		if (q.y >= 0.0) { r = u.radii.z; } else { r = u.radii.y; }
	} else if (q.y >= 0.0) {
		r = u.radii.w;
	}
	return cover(roundRect(q, h, r));
}

fn orb(p: vec2f, R: f32, fill: f32, liquid: vec3f) -> vec4f {
	let d = length(p) - R;
	let a = cover(d);
	let surface = R - 2.0 * R * fill
		+ sin(p.x / R * 5.0 + u.time * 2.3) * R * 0.035
		+ sin(p.x / R * 9.0 - u.time * 3.1) * R * 0.02;
	let inLiquid = clamp((p.y - surface) * u.scale + 0.5, 0.0, 1.0);
	let depth = clamp((p.y + R) / (2.0 * R), 0.0, 1.0);
	let swirl = 0.5 + 0.5 * sin(p.x / R * 6.0 + p.y / R * 4.0 + u.time * 1.3);
	let lc = liquid * (0.55 + 0.6 * depth) * (0.85 + 0.3 * swirl);
	let glass = vec3f(0.015, 0.012, 0.012);
	var c = mix(glass, lc, inLiquid);
	let edge = smoothstep(R * 0.55, R, length(p));
	c *= 1.0 - edge * 0.65;
	let hp = (p - vec2f(-0.32, -0.42) * R) / (R * vec2f(0.42, 0.26));
	c += vec3f(1.0) * pow(max(1.0 - dot(hp, hp), 0.0), 2.0) * 0.55;
	return vec4f(c, a);
}

fn image(slot: u32, uv: vec2f) -> vec4f {
	switch slot {
		case 0u: { return textureSampleLevel(image0, sampler0, uv, 0.0); }
		case 1u: { return textureSampleLevel(image1, sampler1, uv, 0.0); }
		case 2u: { return textureSampleLevel(image2, sampler2, uv, 0.0); }
		case 3u: { return textureSampleLevel(image3, sampler3, uv, 0.0); }
		case 4u: { return textureSampleLevel(image4, sampler4, uv, 0.0); }
		case 5u: { return textureSampleLevel(image5, sampler5, uv, 0.0); }
		case 6u: { return textureSampleLevel(image6, sampler6, uv, 0.0); }
		default: { return textureSampleLevel(image7, sampler7, uv, 0.0); }
	}
}

@fragment
fn fs(in: VertexOut) -> @location(0) vec4f {
	let bits = u32(in.shape.x + 0.5);
	let kind = bits & 15u;
	let slot = bits >> 4u;
	var c = in.color;
	let hs = in.shape.yz;
	if (kind == 1u) {
		c.a *= textureSampleLevel(atlas, atlasSampler, in.uv / vec2f(textureDimensions(atlas)), 0.0).r;
	} else if (kind == 2u) {
		c.a *= cover(roundRect(in.uv, hs, in.shape.w));
	} else if (kind == 3u) {
		c = orb(in.uv, hs.x, in.shape.w, in.color.rgb);
	} else if (kind == 4u) {
		let t = in.shape.w;
		c.a *= cover(abs(length(in.uv) - (hs.x - t * 0.5)) - t * 0.5);
	} else if (kind == 6u) {
		var t = image(slot, in.uv);
		if (((u.targets >> slot) & 1u) == 1u) {
			if (t.a > 0.0) { t = vec4f(srgbDecode(t.rgb / t.a), t.a); } else { t = vec4f(0.0); }
		} else {
			if (t.a > 0.0) { t = vec4f(srgbDecode(srgbEncode(t.rgb) / t.a), t.a); } else { t = vec4f(0.0); }
		}
		c *= t;
		if (u.masked > 0.5) {
			c.a *= textureSampleLevel(mask, maskSampler, in.uv, 0.0).a;
		}
	} else if (kind == 7u) {
		let d = roundRect(in.uv, hs, in.shape.w);
		let k = 1.0 - clamp(d / in.extra, 0.0, 1.0);
		if (d > 0.0) { c.a *= k * k; } else { c.a = 0.0; }
	} else if (kind == 5u) {
		let t = in.extra;
		let outer = roundRect(in.uv, hs, in.shape.w);
		let inner = roundRect(in.uv, hs - t, max(in.shape.w - t, 0.0));
		c.a *= cover(max(outer, -inner));
	}
	if (u.round.z > 0.0) {
		c.a *= roundedClip(in.pos.xy);
	}
{{light}}	if (spread.x != 0.0) {
		c = vec4f(c.rgb * min(light * exposure.x, vec3f(1.0)), c.a);
	}
	var rgb: vec3f;
	if (u.hdr > 1.5) {
		rgb = c.rgb;
	} else if (u.hdr > 0.5) {
		rgb = acesInverse(min(max(c.rgb, vec3f(0.0)), vec3f(0.999))) / u.exposure;
	} else {
		rgb = srgbEncode(c.rgb);
	}
	if (u.blend > 4.5) {
		return vec4f(rgb, c.a);
	}
	if (u.blend > 2.5) {
		return vec4f(rgb * c.a, c.a);
	} else if (u.blend > 1.5) {
		return vec4f(mix(vec3f(1.0), rgb, c.a), c.a);
	}
	return vec4f(rgb, c.a);
}
`
