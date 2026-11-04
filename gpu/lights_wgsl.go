package gpu

// shadowsWGSL fills a light's row of the shadow maps: each column an angle
// round it, the distance, as a share of its radius, it reaches through the
// occluder picture before an occluder of its groups blocks it (it shines
// out of what it stands in).
const shadowsWGSL = lightConsts + `
@group(0) @binding(0) var<uniform> rows: Rows;
@group(0) @binding(1) var occluders: texture_2d<f32>;

@vertex
fn full_vs(@builtin(vertex_index) i: u32) -> @builtin(position) vec4f {
	let p = vec2f(f32((i << 1u) & 2u), f32(i & 2u));
	return vec4f(p * 2.0 - 1.0, 0.0, 1.0);
}

// shadowEnd is where the shadow of a run of occluder left at exit, as
// tall as height (a share of MAX_HEIGHT, 1 for no height given), ends from
// a light elevation texels high: exit·e/(e−h), or past any radius.
fn shadowEnd(exit: f32, height: f32, elevation: f32) -> f32 {
	if (elevation <= 0.0 || height >= 0.999) {
		return 1e9;
	}
	let h = height * MAX_HEIGHT;
	if (h >= elevation) {
		return 1e9;
	}
	return exit * elevation / (elevation - h);
}

@fragment
fn shadow_fs(@builtin(position) at: vec4f) -> @location(0) vec4f {
	let l = rows.at[u32(at.y)];
	let theta = at.x / ANGLES * TAU - PI;
	let dir = vec2f(cos(theta), sin(theta));
	let size = vec2i(textureDimensions(occluders));
	let elevation = floor(l.w / 8.0);
	let mask = groupMask(l.w - 8.0 * elevation);
	let steps = i32(ceil(l.z));
	let scale = 1.0 / max(l.z, 1.0);
	var out = vec4f(NO_SHADOW);
	var exits = vec2f(0.0);
	var n = 0;
	var inside = true;
	var run = false;
	var start = 0.0;
	var height = 0.0;
	for (var s = 0; s <= steps; s++) {
		let p = vec2i(floor(l.xy + dir * f32(s)));
		if (any(p < vec2i(0)) || any(p >= size)) {
			break;
		}
		let texel = textureLoad(occluders, p, 0);
		let blocked = dot(texel.rgb, mask) > 0.5;
		if (inside) {
			inside = blocked;
			continue;
		}
		if (blocked) {
			if (!STRETCHES) {
				return vec4f(f32(s) * scale, 1e9, NO_SHADOW, NO_SHADOW);
			}
			if (!run) {
				run = true;
				start = f32(s);
				height = 0.0;
			}
			height = max(height, texel.a);
			continue;
		}
		if (!run) {
			continue;
		}
		run = false;
		let a = start * scale;
		let b = shadowEnd(f32(s), height, elevation) * scale;
		if (n > 0 && a <= out[2 * n - 1]) {
			out[2 * n - 1] = max(out[2 * n - 1], b);
		} else if (n < 2) {
			out[2 * n] = a;
			out[2 * n + 1] = b;
			exits[n] = f32(s) * scale;
			n++;
		} else {
			out[3] = max(out[3], b);
		}
		if (out[2 * n - 1] >= 1.0) {
			break;
		}
	}
	if (run) {
		let a = start * scale;
		if (n > 0 && a <= out[2 * n - 1]) {
			out[2 * n - 1] = 1e9;
		} else if (n < 2) {
			out[2 * n] = a;
			out[2 * n + 1] = 1e9;
			exits[n] = a;
			n++;
		} else {
			out[3] = 1e9;
		}
	}
	for (var i = 0; i < n; i++) {
		out[2 * i + 1] = packEnd(exits[i], out[2 * i + 1]);
	}
	return out;
}
`

// sunShadowWGSL fills a directional light's shadow picture: for each
// texel, whether its rays reach it, marching back towards them through
// the occluder picture, Length texels at most.
const sunShadowWGSL = lightConsts + `
struct Sun { dir: vec2f, length: f32, groups: f32 };

@group(0) @binding(0) var<uniform> sun: Sun;
@group(0) @binding(1) var occluders: texture_2d<f32>;

@vertex
fn full_vs(@builtin(vertex_index) i: u32) -> @builtin(position) vec4f {
	let p = vec2f(f32((i << 1u) & 2u), f32(i & 2u));
	return vec4f(p * 2.0 - 1.0, 0.0, 1.0);
}

@fragment
fn sun_shadow_fs(@builtin(position) at: vec4f) -> @location(0) vec4f {
	let size = vec2f(textureDimensions(occluders));
	let start = at.xy / SUN_SCALE;
	let mask = groupMask(sun.groups);
	let steps = i32(ceil(sun.length));
	var inside = true;
	for (var s = 0; s <= steps; s++) {
		let p = start - sun.dir * f32(s);
		if (any(p < vec2f(0.0)) || any(p >= size)) {
			break;
		}
		let blocked = dot(textureLoad(occluders, vec2i(p), 0).rgb, mask) > 0.5;
		if (blocked && !inside) {
			return vec4f(0.0, 0.0, 0.0, 1.0);
		}
		inside = inside && blocked;
	}
	return vec4f(1.0, 0.0, 0.0, 1.0);
}
`

// lightPictureWGSL adds each light to the light picture: a point's or a
// cone's soft disc, a shape's light round its exact distance, dark past
// what its shadow maps give.
const lightPictureWGSL = lightConsts + `
// GPULight is one light, in the light picture's points: where it stands,
// radius, falloff; colour, kind; a cone's or a rectangle's direction, the
// cosine of a cone's half spread and its feather; its first row in the
// shadow maps, how many, the columns its softness spans and its rim; its
// shape, first point and how many; its shape's size.
struct GPULight {
	a: vec4f,
	color: vec4f,
	cone: vec4f,
	shadow: vec4f,
	shape: vec4f,
	size: vec4f,
	height: vec4f,
	tint: vec4f,
};

struct Lights { at: array<GPULight, 256> };
struct Points { at: array<vec4f, 512> };

@group(0) @binding(0) var shadowMaps: texture_2d<f32>;
@group(0) @binding(1) var<uniform> rows: Rows;
@group(0) @binding(2) var<uniform> lights: Lights;
@group(0) @binding(3) var<uniform> points: Points;

struct LightOut {
	@builtin(position) pos: vec4f,
	@location(0) local: vec2f,
	@location(1) @interpolate(flat) index: u32,
};

@vertex
fn light_vs(@location(0) ndc: vec2f, @location(1) local: vec2f, @location(2) index: f32) -> LightOut {
	return LightOut(vec4f(ndc, 0.0, 1.0), local, u32(index + 0.5));
}

// faded is how much of a shadow's darkness is left u of the way along it
// that fades by curve (1 linear, 2 smooth, 3 soft) from begin of its
// length.
fn faded(u: f32, curve: i32, begin: f32) -> f32 {
	let t = clamp((u - begin) / max(1.0 - begin, 1e-4), 0.0, 1.0);
	if (curve == 2) {
		return 1.0 - smoothstep(0.0, 1.0, t);
	} else if (curve == 3) {
		return (1.0 - t) * (1.0 - t);
	}
	return 1.0 - t;
}

// lit is how much light reaches r along a ray whose shadowed stretches
// are s (two, each its start and packEnd's exit and end, shares of the
// radius), their edges edge wide, each fading out along the length past
// what casts it by curve from begin of it, 0 for not.
fn lit(s: vec4f, r: f32, edge: f32, curve: i32, begin: f32) -> f32 {
	var v = 1.0;
	for (var i = 0; i < 2; i++) {
		let a = s[2 * i];
		let ends = unpackEnd(s[2 * i + 1]);
		let exit = max(ends.x, a);
		let b = ends.y;
		// A shadow that ends softens over its last quarter rather than
		// stopping square across the ray.
		let tail = select(edge, max(edge, (b - exit) * 0.25), b < 1e8);
		var dark = smoothstep(a, a + edge, r) * (1.0 - smoothstep(b - tail, b, r));
		if (curve > 0 && b < 1e8) {
			dark *= faded((r - exit) / max(b - exit, 1e-4), curve, begin);
		}
		v = min(v, 1.0 - dark);
	}
	return v;
}

// visible is how much of row's light reaches off, an offset from it, at
// r, a share of its radius: the visibility over the columns its softness
// spans, each blended between the two columns round it. A light of size
// texels widens that span with r's distance behind the shadow's start,
// its edge sharp there (contact hardening).
fn visible(row: i32, off: vec2f, r: f32, radius: f32, soft: f32, edge: f32, size: f32, curve: i32,
	begin: f32, stretches: bool) -> f32 {
	let x = (atan2(off.y, off.x) + PI) / TAU * ANGLES - 0.5;
	let n = i32(ANGLES);
	var spread = soft;
	if (STRETCHES && size > 0.0) {
		let c = ((i32(floor(x + 0.5)) % n) + n) % n;
		let a = textureLoad(shadowMaps, vec2i(c, row), 0).r;
		if (a < r) {
			let behind = (r - a) * radius;
			spread += min(size * behind / max(a * radius * r * radius, 1.0) * ANGLES / TAU, 24.0);
		}
	}
	let taps = select(1, select(5, 9, size > 0.0), spread > 0.25);
	var sum = 0.0;
	var weights = 0.0;
	for (var k = 0; k < taps; k++) {
		var xs = x;
		var weight = 1.0;
		if (taps > 1) {
			let t = f32(k) / f32(taps - 1) * 2.0 - 1.0;
			xs = x + spread * t;
			weight = 1.0 - abs(t) * 0.8;
		}
		let c = floor(xs);
		let c0 = ((i32(c) % n) + n) % n;
		let c1 = (c0 + 1) % n;
		let s0 = textureLoad(shadowMaps, vec2i(c0, row), 0);
		let s1 = textureLoad(shadowMaps, vec2i(c1, row), 0);
		if (stretches) {
			sum += weight * mix(lit(s0, r, edge, curve, begin), lit(s1, r, edge, curve, begin), xs - c);
		} else {
			sum += weight * mix(1.0 - smoothstep(s0.r, s0.r + edge, r), 1.0 - smoothstep(s1.r, s1.r + edge, r), xs - c);
		}
		weights += weight;
	}
	return sum / weights;
}

fn segment(p: vec2f, a: vec2f, b: vec2f) -> f32 {
	let ab = b - a;
	let t = clamp(dot(p - a, ab) / max(dot(ab, ab), 1e-6), 0.0, 1.0);
	return length(p - a - ab * t);
}

fn polygon(p: vec2f, first: i32, count: i32) -> f32 {
	var d = 1e9;
	var inside = false;
	for (var i = 0; i < count; i++) {
		let a = points.at[first + i].xy;
		let b = points.at[first + (i + 1) % count].xy;
		d = min(d, segment(p, a, b));
		if ((a.y > p.y) != (b.y > p.y) && p.x < (b.x - a.x) * (p.y - a.y) / (b.y - a.y) + a.x) {
			inside = !inside;
		}
	}
	return select(d, -d, inside);
}

// shapeDistance is how far local stands outside light l's shape, negative
// inside.
fn shapeDistance(l: GPULight, local: vec2f) -> f32 {
	let kind = i32(l.shape.x + 0.5);
	let first = i32(l.shape.y + 0.5);
	if (kind == 0) {
		let dir = l.cone.xy;
		let q = abs(vec2f(dot(local, dir), dot(local, vec2f(-dir.y, dir.x)))) - l.size.xy * 0.5;
		return length(max(q, vec2f(0.0))) + min(max(q.x, q.y), 0.0);
	} else if (kind == 1) {
		return length(local - points.at[first].xy) - l.size.x;
	} else if (kind == 2) {
		return segment(local, points.at[first].xy, points.at[first + 1].xy) - l.size.x;
	} else if (kind == 3) {
		return segment(local, points.at[first].xy, points.at[first + 1].xy);
	}
	return polygon(local, first, i32(l.shape.z + 0.5));
}

@fragment
fn light_fs(in: LightOut) -> @location(0) vec4f {
	let l = lights.at[in.index];
	let radius = l.a.z;
	let kind = i32(l.color.w + 0.5);
	var f = 0.0;
	if (kind == 3) {
		let d = max(shapeDistance(l, in.local), 0.0);
		f = 1.0 - smoothstep(l.a.w * radius, radius, d);
	} else {
		let r = length(in.local);
		if (r >= radius) {
			return vec4f(0.0);
		}
		f = 1.0 - smoothstep(l.a.w * radius, radius, r);
		if (kind == 1) {
			let c = dot(in.local / max(r, 1e-6), l.cone.xy);
			f *= smoothstep(l.cone.z, l.cone.z + l.cone.w, c);
		}
	}
	let count = i32(l.shadow.y + 0.5);
	var v = 1.0;
	if (count > 0 && f > 0.0) {
		let p = l.a.xy + in.local;
		v = 0.0;
		for (var k = 0; k < count; k++) {
			let row = i32(l.shadow.x + 0.5) + k;
			let s = rows.at[row];
			let off = p - s.xy / OCCLUDER_SCALE;
			v += visible(row, off, length(off) / max(s.z / OCCLUDER_SCALE, 1.0), s.z, l.shadow.z, l.shadow.w,
				l.height.x, i32(l.height.y + 0.5), l.height.w, STRETCHES && l.height.z > 0.5);
		}
		v /= f32(count);
	}
	return vec4f(l.color.rgb * f * mix(l.tint.rgb, vec3f(1.0), v), 0.0);
}
`

// sunWGSL adds a directional light over the whole light picture, where
// its shadow picture lets it through.
const sunWGSL = `
@group(0) @binding(0) var sunShadow: texture_2d<f32>;
@group(0) @binding(1) var sunSampler: sampler;
// SunColor is a directional light's colour and how much of it its
// shadows let through.
struct SunColor { color: vec4f, tint: vec4f };

@group(0) @binding(2) var<uniform> sun: SunColor;

struct SunOut {
	@builtin(position) pos: vec4f,
	@location(0) uv: vec2f,
};

@vertex
fn sun_vs(@builtin(vertex_index) i: u32) -> SunOut {
	let p = vec2f(f32((i << 1u) & 2u), f32(i & 2u));
	return SunOut(vec4f(p.x * 2.0 - 1.0, 1.0 - p.y * 2.0, 0.0, 1.0), p);
}

@fragment
fn sun_fs(in: SunOut) -> @location(0) vec4f {
	let v = textureSampleLevel(sunShadow, sunSampler, in.uv, 0.0).r;
	return vec4f(sun.color.rgb * mix(sun.tint.rgb, vec3f(1.0), v), 0.0);
}
`

// lightConsts are what the lighting shaders agree on: the shadow maps'
// columns, the occluder picture's texels a point, the sun pictures' share
// of it, and the shadow rows.
const lightConsts = `
const PI = 3.14159265;
const TAU = 6.2831853;
const ANGLES = 512.0;
const OCCLUDER_SCALE = 1.0;
const SUN_SCALE = 0.5;
const NO_SHADOW = 9.0;
const STRETCHES = true;
const MAX_HEIGHT = 1024.0;

// Rows are the shadow-casting lights, a row of the shadow maps each:
// where it stands in the occluder picture and its radius there, in
// texels, and its occluder groups' bits plus 8 times its elevation in
// texels.
struct Rows { at: array<vec4f, 256> };

// packEnd keeps, in one float of a shadow map, where a shadowed stretch
// leaves what casts it and where it ends, shares of the radius up to 4,
// the end past 4 for one that doesn't; the bits never make a NaN or a
// denormal, which a write could flush.
fn packEnd(exit: f32, end: f32) -> f32 {
	let hi = 128u + u32(round(clamp(exit, 0.0, 4.0) / 4.0 * 32000.0));
	var lo = 65535u;
	if (end < 1e8) {
		lo = min(u32(round(clamp(end, 0.0, 4.0) / 4.0 * 65534.0)), 65534u);
	}
	return bitcast<f32>((hi << 16u) | lo);
}

// unpackEnd is packEnd's exit and end, 1e9 for an end it didn't keep.
fn unpackEnd(v: f32) -> vec2f {
	let u = bitcast<u32>(v);
	let lo = u & 0xFFFFu;
	return vec2f(f32(i32(u >> 16u) - 128) / 32000.0 * 4.0, select(f32(lo) / 65534.0 * 4.0, 1e9, lo == 65535u));
}

fn groupMask(bits: f32) -> vec3f {
	let b = u32(bits + 0.5);
	return vec3f(f32(b & 1u), f32((b >> 1u) & 1u), f32((b >> 2u) & 1u));
}
`

const shadowAngles = 512

// sunScale is a directional light's shadow picture, as a share of the
// occluder picture: SUN_SCALE.
const sunScale = 0.5

// max2DLights and maxLightPoints are how many lights a window draws, and
// how many shape points they have between them: the uniform arrays' sizes.
const (
	max2DLights    = 256
	maxLightPoints = 512
)

// glowWGSL makes an emissive sprite's glow texture: at each texel, the
// strongest of its picture's texels within pad, each fading to nothing
// at pad away, so the glow is full at the picture's edge whatever its
// size and reach; written as a pass's picture is (colour in sRGB, times
// alpha).
const glowWGSL = srgbWGSL + `
struct Blur { region: vec4f, pad: f32, step: f32, _a: f32, _b: f32 };

@group(0) @binding(0) var<uniform> blur: Blur;
@group(0) @binding(1) var source: texture_2d<f32>;

@vertex
fn full_vs(@builtin(vertex_index) i: u32) -> @builtin(position) vec4f {
	let p = vec2f(f32((i << 1u) & 2u), f32(i & 2u));
	return vec4f(p * 2.0 - 1.0, 0.0, 1.0);
}

@fragment
fn glow_fs(@builtin(position) at: vec4f) -> @location(0) vec4f {
	let center = blur.region.xy + at.xy - vec2f(blur.pad);
	let lo = blur.region.xy;
	let hi = blur.region.xy + blur.region.zw;
	let step = blur.step;
	let n = i32(ceil(blur.pad / step));
	var best = vec4f(0.0);
	for (var j = -n; j <= n; j++) {
		for (var i = -n; i <= n; i++) {
			let o = vec2f(f32(i), f32(j)) * step;
			let p = clamp(center + o, lo, hi - vec2f(1.0));
			let d = length(p - center);
			if (d > blur.pad + 0.5) {
				continue;
			}
			let w = 1.0 - smoothstep(0.0, max(blur.pad, 1e-3), d);
			let t = textureLoad(source, vec2i(p), 0) * w;
			if (t.a > best.a) {
				best = t;
			}
		}
	}
	if (best.a <= 0.0) {
		return vec4f(0.0);
	}
	return vec4f(srgbEncode(best.rgb / best.a) * best.a, best.a);
}
`

// probesWGSL reads the light picture at each probe, a texel a probe: the
// light a white lit pixel there takes, capped as the 2D shader caps it.
const probesWGSL = `
struct Probes { at: array<vec4f, 1024> };

@group(0) @binding(0) var<uniform> probes: Probes;
@group(0) @binding(1) var lightPicture: texture_2d<f32>;
@group(0) @binding(2) var lightSampler: sampler;

@vertex
fn full_vs(@builtin(vertex_index) i: u32) -> @builtin(position) vec4f {
	let p = vec2f(f32((i << 1u) & 2u), f32(i & 2u));
	return vec4f(p * 2.0 - 1.0, 0.0, 1.0);
}

@fragment
fn probe_fs(@builtin(position) at: vec4f) -> @location(0) vec4f {
	let uv = probes.at[u32(at.x)].xy;
	return vec4f(min(textureSampleLevel(lightPicture, lightSampler, uv, 0.0).rgb, vec3f(1.0)), 1.0);
}
`

// lightDebugWGSL shows one of a window's lighting textures over it, F3's
// F5: the light picture or the occluder picture where they lie under the
// window, or the shadow maps stretched across it, a band a light at least
// 8 pixels tall, its first shadow from dark (near) to light (far), teal
// where nothing blocks it.
const lightDebugWGSL = `
struct Debug { mode: f32, scale: f32, margin: f32, rows: f32, rect: vec2f, fb: vec2f, panel: vec4f, area: vec4f };
@group(0) @binding(0) var<uniform> d: Debug;
@group(0) @binding(1) var picture: texture_2d<f32>;

@vertex
fn full_vs(@builtin(vertex_index) i: u32) -> @builtin(position) vec4f {
	let p = vec2f(f32((i << 1u) & 2u), f32(i & 2u));
	return vec4f(p * 2.0 - 1.0, 0.0, 1.0);
}

@fragment
fn debug_fs(@builtin(position) at: vec4f) -> @location(0) vec4f {
	if (all(at.xy >= d.panel.xy) && all(at.xy < d.panel.zw)) {
		return vec4f(0.0);
	}
	if (d.area.z > d.area.x && (any(at.xy < d.area.xy) || any(at.xy >= d.area.zw))) {
		return vec4f(0.0);
	}
	let size = vec2f(textureDimensions(picture));
	if (d.mode > 2.5) {
		var lo = vec2f(0.0);
		var span = d.fb;
		if (d.area.z > d.area.x) {
			lo = d.area.xy;
			span = d.area.zw - d.area.xy;
		}
		let f = (at.xy - lo) / span;
		let shown = max(min(d.rows, floor(span.y / 8.0)), 1.0);
		let row = f.y * shown;
		if (fract(row) * span.y / shown < 1.0 && row >= 1.0) {
			return vec4f(0.0, 0.0, 0.0, 0.9);
		}
		let start = textureLoad(picture, vec2i(i32(f.x * size.x), i32(row)), 0).r;
		if (start >= 1.0) {
			return vec4f(0.1, 0.32, 0.34, 0.9);
		}
		let near = vec3f(0.12, 0.02, 0.2);
		let far = vec3f(1.0, 0.85, 0.45);
		return vec4f(mix(near, far, sqrt(max(start, 0.0))), 0.9);
	}
	let points = at.xy / d.scale + vec2f(d.margin);
	let texel = vec2i(points / d.rect * size);
	let c = textureLoad(picture, texel, 0);
	if (d.mode > 1.5) {
		return vec4f(c.rgb * 0.7 + vec3f(select(c.a, 0.0, c.a > 0.999)) * 0.3, 0.85);
	}
	return vec4f(min(c.rgb, vec3f(1.0)), 0.9);
}
`

// lightCopyWGSL copies the light picture into a shaded one, texel for
// texel, before the lights an excluded occluder reaches are redrawn.
const lightCopyWGSL = `
@group(0) @binding(0) var picture: texture_2d<f32>;
@group(0) @binding(1) var pictureSampler: sampler;

@vertex
fn full_vs(@builtin(vertex_index) i: u32) -> @builtin(position) vec4f {
	let p = vec2f(f32((i << 1u) & 2u), f32(i & 2u));
	return vec4f(p * 2.0 - 1.0, 0.0, 1.0);
}

@fragment
fn copy_fs(@builtin(position) at: vec4f) -> @location(0) vec4f {
	return textureLoad(picture, vec2i(at.xy), 0);
}
`

// maskClearWGSL clears the part of a silhouette mask the scissor leaves:
// a light's reach, before its silhouettes are drawn in.
const maskClearWGSL = `
@vertex
fn full_vs(@builtin(vertex_index) i: u32) -> @builtin(position) vec4f {
	let p = vec2f(f32((i << 1u) & 2u), f32(i & 2u));
	return vec4f(p * 2.0 - 1.0, 0.0, 1.0);
}

@fragment
fn clear_fs() -> @location(0) vec4f {
	return vec4f(0.0);
}
`

// silhouetteMaskWGSL draws silhouette quads into the mask atlas, an
// instance a quad: the sprite's picture laid on the ground from its foot,
// away from the light casting it, as long as it is tall (d·h/(e−h) under a
// light with an elevation, never past its reach), at its light's tile, its
// alpha kept where it is, within the light's area.
const silhouetteMaskWGSL = `
struct GPULight {
	a: vec4f,
	color: vec4f,
	cone: vec4f,
	shadow: vec4f,
	shape: vec4f,
	size: vec4f,
	height: vec4f,
	tint: vec4f,
};
struct Lights { at: array<GPULight, 256> };
struct Casts { at: array<vec4f, 512> };

@group(0) @binding(0) var<uniform> lights: Lights;
@group(0) @binding(1) var<uniform> casts: Casts;
@group(0) @binding(2) var<uniform> page: vec4f;
@group(1) @binding(0) var picture: texture_2d<f32>;
@group(1) @binding(1) var pictureSampler: sampler;

struct MaskOut {
	@builtin(position) pos: vec4f,
	@location(0) uv: vec2f,
	@location(1) at: vec2f,
	@location(2) @interpolate(flat) light: u32,
};

@vertex
fn mask_vs(@builtin(vertex_index) i: u32, @location(0) foot: vec2f, @location(1) size: vec2f,
	@location(2) rect: vec4f, @location(3) uv: vec4f, @location(4) light: f32) -> MaskOut {
	var corners = array<vec2f, 6>(vec2f(0.0, 0.0), vec2f(1.0, 0.0), vec2f(1.0, 1.0), vec2f(0.0, 0.0), vec2f(1.0,
		1.0), vec2f(0.0, 1.0));
	let c = corners[i];
	let k = u32(light + 0.5);
	let l = lights.at[k];
	let away = foot - l.a.xy;
	let d = max(length(away), 1e-3);
	let dir = away / d;
	let elevation = casts.at[2u * k].z;
	var long = size.y;
	if (elevation > size.y) {
		long = d * size.y / (elevation - size.y);
	}
	long = min(long, l.a.z - d);
	var out: MaskOut;
	out.light = k;
	if (long <= 0.0) {
		out.pos = vec4f(2.0, 2.0, 0.0, 1.0);
		return out;
	}
	let across = vec2f(dir.y, -dir.x) * size.x;
	let along = dir * (-long / rect.w);
	let local = rect.xy + c * rect.zw;
	let footLocal = vec2f(rect.x + rect.z * 0.5, rect.y + rect.w);
	let p = foot + across * (local.x - footLocal.x) + along * (local.y - footLocal.y);
	let a = (p + casts.at[2u * k].xy) / page.x * 2.0 - 1.0;
	out.pos = vec4f(a.x, -a.y, 0.0, 1.0);
	out.uv = mix(uv.xy, uv.zw, c);
	out.at = p;
	return out;
}

@fragment
fn mask_fs(in: MaskOut) -> @location(0) vec4f {
	let area = casts.at[2u * in.light + 1u];
	if (in.at.x < area.x || in.at.y < area.y || in.at.x > area.x + area.z || in.at.y > area.y + area.w) {
		discard;
	}
	let a = textureSampleLevel(picture, pictureSampler, in.uv, 0.0).a;
	return vec4f(a);
}
`
