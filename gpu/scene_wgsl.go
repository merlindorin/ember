package gpu

// The 3D renderer's shaders: the lit scene, the shadow map, gizmo lines,
// billboards, and the bloom, composite and present passes. geom's
// projections map depth to -1..1; every vertex stage maps it to WebGPU's
// 0..1, so depths and shadow lookups keep their values. The shadow map's
// lookups flip y, since a render target's first row is its top, and noise
// that depends on the pixel counts it from the bottom, as the OpenGL
// renderer did, so pictures match its references.

const sceneWGSL = `
struct Frame {
	viewProj: mat4x4f,
	lightVP: mat4x4f,
	camPos: vec3f,
	time: f32,
	sunDir: vec3f,
	ambient: f32,
	sunColor: vec3f,
	fogDensity: f32,
	sky: vec3f,
	shadowTexel: f32,
	ground: vec3f,
	shadows: f32,
	fogColor: vec3f,
	lightCount: f32,
	right: vec3f,
	_r: f32,
	up: vec3f,
	_u: f32,
	lightPos: array<vec4f, 32>,
	lightColor: array<vec4f, 32>,
};

struct Material {
	albedo: vec4f,
	emissive: vec3f,
	specular: f32,
	wind: f32,
	unlit: f32,
	textured: f32,
	_m: f32,
};

@group(0) @binding(0) var<uniform> frame: Frame;
@group(0) @binding(1) var shadowMap: texture_depth_2d;
@group(0) @binding(2) var shadowSampler: sampler_comparison;
@group(1) @binding(0) var<uniform> material: Material;
@group(1) @binding(1) var materialTexture: texture_2d<f32>;
@group(1) @binding(2) var materialSampler: sampler;

// depthToWebGPU maps a clip position from geom's -w..w depth to 0..w.
fn depthToWebGPU(p: vec4f) -> vec4f { return vec4f(p.xy, (p.z + p.w) * 0.5, p.w); }

fn sway(p: vec4f, height: f32, wind: f32) -> vec4f {
	var wp = p;
	if (wind > 0.0) {
		let h = max(height, 0.0);
		let t = frame.time + wp.x * 0.21 + wp.z * 0.17;
		wp.x += sin(t * 1.7) * wind * h * h * 0.02;
		wp.z += cos(t * 1.3) * wind * h * h * 0.015;
	}
	return wp;
}

// normalMatrix is the inverse transpose of m, up to scale and sign of the
// determinant, which normalising leaves alone.
fn normalMatrix(m: mat3x3f) -> mat3x3f {
	let c0 = cross(m[1], m[2]);
	let c1 = cross(m[2], m[0]);
	let c2 = cross(m[0], m[1]);
	return mat3x3f(c0, c1, c2) * sign(dot(m[0], c0));
}

struct Lit {
	@builtin(position) pos: vec4f,
	@location(0) world: vec3f,
	@location(1) normal: vec3f,
	@location(2) color: vec4f,
	@location(3) uv: vec2f,
};

@vertex
fn lit_vs(@location(0) pos: vec3f, @location(1) normal: vec3f, @location(2) uv: vec2f, @location(3) color: vec4f,
	@location(4) m0: vec4f, @location(5) m1: vec4f, @location(6) m2: vec4f, @location(7) m3: vec4f,
		@location(8) tint: vec4f) -> Lit {
	let model = mat4x4f(m0, m1, m2, m3);
	let wp = sway(model * vec4f(pos, 1.0), pos.y, material.wind);
	var out: Lit;
	out.world = wp.xyz;
	out.normal = normalMatrix(mat3x3f(m0.xyz, m1.xyz, m2.xyz)) * normal;
	out.color = color * tint;
	out.uv = uv;
	out.pos = depthToWebGPU(frame.viewProj * wp);
	return out;
}

fn sunShadow(world: vec3f, n: vec3f, ndl: f32) -> f32 {
	if (frame.shadows < 0.5) { return 1.0; }
	let lp = frame.lightVP * vec4f(world + n * 0.04, 1.0);
	let q = lp.xyz / lp.w;
	let uv = vec2f(q.x * 0.5 + 0.5, 0.5 - q.y * 0.5);
	let z = q.z * 0.5 + 0.5;
	if (z > 1.0 || uv.x < 0.0 || uv.x > 1.0 || uv.y < 0.0 || uv.y > 1.0) { return 1.0; }
	let bias = max(0.0015 * (1.0 - ndl), 0.0004);
	var s = 0.0;
	for (var x = -1; x <= 1; x++) {
		for (var y = -1; y <= 1; y++) {
			s += textureSampleCompareLevel(shadowMap, shadowSampler, uv + vec2f(f32(x), -f32(y)) * frame.shadowTexel, z - bias);
		}
	}
	return s / 9.0;
}

@fragment
fn lit_fs(in: Lit, @builtin(front_facing) front: bool) -> @location(0) vec4f {
	var base = material.albedo * in.color;
	if (material.textured > 0.5) {
		// The branch is on a uniform, so implicit derivatives pick the mip
		// level.
		base *= textureSample(materialTexture, materialSampler, in.uv);
	}
	var color: vec3f;
	if (material.unlit > 0.5) {
		color = base.rgb + material.emissive;
	} else {
		var n = normalize(in.normal);
		if (!front) { n = -n; }
		let v = normalize(frame.camPos - in.world);
		let l = -frame.sunDir;
		let ndl = max(dot(n, l), 0.0);
		let sh = sunShadow(in.world, n, ndl);
		let hemi = mix(frame.ground, frame.sky, n.y * 0.5 + 0.5) * frame.ambient;
		color = base.rgb * (hemi + frame.sunColor * ndl * sh);
		let h = normalize(l + v);
		color += frame.sunColor * pow(max(dot(n, h), 0.0), 48.0) * material.specular * sh * ndl;
		for (var i = 0; i < i32(frame.lightCount); i++) {
			let d = frame.lightPos[i].xyz - in.world;
			let dist = length(d);
			let range = frame.lightPos[i].w;
			if (dist > range) { continue; }
			let ld = d / dist;
			var att = 1.0 - dist / range;
			att *= att;
			let lndl = max(dot(n, ld), 0.0) * 0.85 + 0.15;
			let lc = frame.lightColor[i].rgb * att;
			color += base.rgb * lc * lndl;
			let lh = normalize(ld + v);
			color += lc * pow(max(dot(n, lh), 0.0), 32.0) * material.specular;
		}
		color += material.emissive;
	}
	let dist = length(frame.camPos - in.world);
	let fog = 1.0 - exp(-pow(dist * frame.fogDensity, 2.0));
	return vec4f(mix(color, frame.fogColor, fog), base.a);
}

@vertex
fn shadow_vs(@location(0) pos: vec3f, @location(4) m0: vec4f, @location(5) m1: vec4f, @location(6) m2: vec4f,
	@location(7) m3: vec4f) -> @builtin(position) vec4f {
	let wp = sway(mat4x4f(m0, m1, m2, m3) * vec4f(pos, 1.0), pos.y, material.wind);
	return depthToWebGPU(frame.lightVP * wp);
}

struct Colored {
	@builtin(position) pos: vec4f,
	@location(0) color: vec4f,
	@location(1) uv: vec2f,
};

@vertex
fn line_vs(@location(0) pos: vec3f, @location(1) color: vec4f) -> Colored {
	var out: Colored;
	out.color = color;
	out.pos = depthToWebGPU(frame.viewProj * vec4f(pos, 1.0));
	return out;
}

@fragment
fn line_fs(in: Colored) -> @location(0) vec4f { return in.color; }

@vertex
fn billboard_vs(@location(0) center: vec3f, @location(1) corner: vec3f, @location(2) color: vec4f) -> Colored {
	let c = cos(corner.z);
	let s = sin(corner.z);
	let k = mat2x2f(c, s, -s, c) * corner.xy;
	let wp = center + frame.right * k.x + frame.up * k.y;
	var out: Colored;
	out.color = color;
	out.uv = corner.xy / max(length(corner.xy) * 0.7071, 1e-5);
	out.pos = depthToWebGPU(frame.viewProj * vec4f(wp, 1.0));
	return out;
}

@fragment
fn billboard_fs(in: Colored) -> @location(0) vec4f {
	let d = length(in.uv);
	var a = clamp(1.0 - d, 0.0, 1.0);
	a = a * a * (3.0 - 2.0 * a);
	return vec4f(in.color.rgb * a * in.color.a, a * in.color.a);
}
`

// postWGSL is the fullscreen passes: bloom's prefilter, down and up
// samples, the composite and the present, all reading src (and the
// composite bloom too) and one Post struct.
const postWGSL = srgbWGSL + `
struct Post {
	texel: vec2f,
	size: vec2f,
	threshold: f32,
	bloomStrength: f32,
	exposure: f32,
	vignette: f32,
	saturation: f32,
	contrast: f32,
	time: f32,
	nearest: f32,
	shadowTint: vec3f,
	_s: f32,
	highlightTint: vec3f,
	_h: f32,
};

@group(0) @binding(0) var<uniform> post: Post;
@group(0) @binding(1) var src: texture_2d<f32>;
@group(0) @binding(2) var srcSampler: sampler;
@group(0) @binding(3) var bloom: texture_2d<f32>;
@group(0) @binding(4) var bloomSampler: sampler;

struct Full {
	@builtin(position) pos: vec4f,
	@location(0) uv: vec2f,
};

@vertex
fn full_vs(@builtin(vertex_index) i: u32) -> Full {
	let p = vec2f(f32((i << 1u) & 2u), f32(i & 2u));
	return Full(vec4f(p.x * 2.0 - 1.0, 1.0 - p.y * 2.0, 0.0, 1.0), p);
}

fn at(uv: vec2f) -> vec4f { return textureSampleLevel(src, srcSampler, uv, 0.0); }

@fragment
fn prefilter_fs(in: Full) -> @location(0) vec4f {
	let c = at(in.uv).rgb;
	let b = max(c.r, max(c.g, c.b));
	let knee = post.threshold * 0.5;
	var soft = clamp(b - post.threshold + knee, 0.0, 2.0 * knee);
	soft = soft * soft / (4.0 * knee + 1e-4);
	let w = max(soft, b - post.threshold) / max(b, 1e-4);
	return vec4f(min(c * w, vec3f(64.0)), 1.0);
}

@fragment
fn down_fs(in: Full) -> @location(0) vec4f {
	var s = at(in.uv) * 4.0;
	s += at(in.uv + vec2f(-1.0, -1.0) * post.texel);
	s += at(in.uv + vec2f(1.0, -1.0) * post.texel);
	s += at(in.uv + vec2f(-1.0, 1.0) * post.texel);
	s += at(in.uv + vec2f(1.0, 1.0) * post.texel);
	return s / 8.0;
}

@fragment
fn up_fs(in: Full) -> @location(0) vec4f {
	var s = at(in.uv + vec2f(-2.0, 0.0) * post.texel);
	s += at(in.uv + vec2f(2.0, 0.0) * post.texel);
	s += at(in.uv + vec2f(0.0, -2.0) * post.texel);
	s += at(in.uv + vec2f(0.0, 2.0) * post.texel);
	s += at(in.uv + vec2f(-1.0, 1.0) * post.texel) * 2.0;
	s += at(in.uv + vec2f(1.0, 1.0) * post.texel) * 2.0;
	s += at(in.uv + vec2f(-1.0, -1.0) * post.texel) * 2.0;
	s += at(in.uv + vec2f(1.0, -1.0) * post.texel) * 2.0;
	return s / 12.0;
}

fn aces(x: vec3f) -> vec3f {
	return clamp((x * (2.51 * x + 0.03)) / (x * (2.43 * x + 0.59) + 0.14), vec3f(0.0), vec3f(1.0));
}

@fragment
fn composite_fs(in: Full) -> @location(0) vec4f {
	let hdr = at(in.uv).rgb + textureSampleLevel(bloom, bloomSampler, in.uv, 0.0).rgb * post.bloomStrength;
	var c = aces(hdr * post.exposure);
	let luma = dot(c, vec3f(0.2126, 0.7152, 0.0722));
	c = mix(vec3f(luma), c, post.saturation);
	c *= mix(post.shadowTint, post.highlightTint, smoothstep(0.0, 0.8, luma));
	c = (c - 0.5) * post.contrast + 0.5;
	let q = in.uv - 0.5;
	c *= 1.0 - post.vignette * dot(q, q) * 2.2;
	c = srgbEncode(c);
	// Rows count from the bottom, as the references' noise did.
	let frag = vec2f(in.pos.x, post.size.y - in.pos.y);
	let n = fract(sin(dot(frag + post.time, vec2f(12.9898, 78.233))) * 43758.5453);
	return vec4f(c + (n - 0.5) / 255.0, 1.0);
}

@fragment
fn present_fs(in: Full) -> @location(0) vec4f {
	var uv = in.uv;
	if (post.nearest > 0.5) {
		uv = (floor(in.uv * post.size) + 0.5) / post.size;
	}
	return vec4f(at(uv).rgb, 1.0);
}
`

// depthCopyWGSL takes sample 0 of the multisampled scene depth into a
// float texture post effects read.
const depthCopyWGSL = `
@group(0) @binding(0) var depthMS: texture_depth_multisampled_2d;
@group(0) @binding(1) var depthOne: texture_depth_2d;

@vertex
fn full_vs(@builtin(vertex_index) i: u32) -> @builtin(position) vec4f {
	let p = vec2f(f32((i << 1u) & 2u), f32(i & 2u));
	return vec4f(p.x * 2.0 - 1.0, 1.0 - p.y * 2.0, 0.0, 1.0);
}

@fragment
fn copy_ms_fs(@builtin(position) pos: vec4f) -> @location(0) vec4f {
	return vec4f(textureLoad(depthMS, vec2i(pos.xy), 0), 0.0, 0.0, 1.0);
}

@fragment
fn copy_fs(@builtin(position) pos: vec4f) -> @location(0) vec4f {
	return vec4f(textureLoad(depthOne, vec2i(pos.xy), 0), 0.0, 0.0, 1.0);
}
`
