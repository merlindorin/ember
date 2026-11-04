// A file system over localStorage for Go's os package in a page (decision
// 0054's W3): the game's saves and settings go through os.ReadFile,
// os.WriteFile, MkdirAll and Rename as on the desktop, each file a key
// "ember:file:<path>" holding base64 and its mtime, each directory a key
// "ember:dir:<path>". Loaded before wasm_exec.js, which then keeps it. Its
// callbacks are called at once, so Go never waits on the browser for a
// file. Standard output and error go to the console, line by line.
(() => {
	const fileKey = p => "ember:file:" + p, dirKey = p => "ember:dir:" + p;
	const S_IFDIR = 0o40000, S_IFREG = 0o100000;
	const constants = { O_RDONLY: 0, O_WRONLY: 1, O_RDWR: 2, O_CREAT: 64, O_EXCL: 128, O_TRUNC: 512, O_APPEND: 1024, O_DIRECTORY: 65536 };
	const decoder = new TextDecoder("utf-8");
	const output = { 1: "", 2: "" };
	const fds = new Map();
	let nextFD = 10;

	const fail = code => { const e = new Error(code); e.code = code; return e; };
	const clean = p => {
		const parts = [];
		for (const s of String(p).split("/")) {
			if (s === "" || s === ".") continue;
			if (s === "..") parts.pop(); else parts.push(s);
		}
		return "/" + parts.join("/");
	};
	const parent = p => p === "/" ? "/" : clean(p + "/..");
	const toBytes = b64 => Uint8Array.from(atob(b64), c => c.charCodeAt(0));
	const toB64 = bytes => {
		let s = "";
		for (let i = 0; i < bytes.length; i += 0x8000) s += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
		return btoa(s);
	};
	const readFile = p => {
		const v = localStorage.getItem(fileKey(p));
		if (v === null) return null;
		const { d, m } = JSON.parse(v);
		return { data: toBytes(d), mtime: m };
	};
	const writeFile = (p, data) => localStorage.setItem(fileKey(p), JSON.stringify({ d: toB64(data), m: Date.now() }));
	const isDir = p => {
		if (p === "/" || localStorage.getItem(dirKey(p)) !== null) return true;
		const prefix = p + "/";
		for (let i = 0; i < localStorage.length; i++) {
			const k = localStorage.key(i);
			if (k.startsWith(fileKey(prefix)) || k.startsWith(dirKey(prefix))) return true;
		}
		return false;
	};
	const stat = p => {
		const f = readFile(p);
		const base = { dev: 0, ino: 0, nlink: 1, uid: 0, gid: 0, rdev: 0, blksize: 4096, blocks: 0, atimeMs: 0, ctimeMs: 0 };
		if (f) return { ...base, mode: S_IFREG | 0o644, size: f.data.length, mtimeMs: f.mtime, atimeMs: f.mtime, ctimeMs: f.mtime, isDirectory: () => false };
		if (isDir(p)) return { ...base, mode: S_IFDIR | 0o755, size: 0, mtimeMs: 0, isDirectory: () => true };
		return null;
	};
	const flush = f => { if (f.dirty) { writeFile(f.path, f.data); f.dirty = false; } };
	const fdOf = fd => { const f = fds.get(fd); if (!f) throw fail("EBADF"); return f; };
	const done = (cb, fn) => { let r; try { r = fn(); } catch (e) { cb(e.code ? e : fail("EIO")); return; } cb(null, r); };

	globalThis.fs = {
		constants,
		writeSync(fd, buf) {
			if (fd === 1 || fd === 2) {
				output[fd] += decoder.decode(buf);
				const nl = output[fd].lastIndexOf("\n");
				if (nl !== -1) {
					(fd === 2 ? console.error : console.log)(output[fd].substring(0, nl));
					output[fd] = output[fd].substring(nl + 1);
				}
				return buf.length;
			}
			throw fail("EBADF");
		},
		open(path, flags, mode, cb) {
			done(cb, () => {
				const p = clean(path);
				if (isDir(p) && !readFile(p)) {
					if (flags & (constants.O_WRONLY | constants.O_RDWR)) throw fail("EISDIR");
					const fd = nextFD++;
					fds.set(fd, { path: p, dir: true });
					return fd;
				}
				let f = readFile(p);
				if (f && (flags & constants.O_CREAT) && (flags & constants.O_EXCL)) throw fail("EEXIST");
				if (!f) {
					if (!(flags & constants.O_CREAT)) throw fail("ENOENT");
					if (!isDir(parent(p))) throw fail("ENOENT");
					f = { data: new Uint8Array(0), mtime: Date.now() };
					writeFile(p, f.data);
				}
				let data = f.data;
				if (flags & constants.O_TRUNC) data = new Uint8Array(0);
				const fd = nextFD++;
				fds.set(fd, { path: p, data, pos: (flags & constants.O_APPEND) ? data.length : 0, dirty: !!(flags & constants.O_TRUNC), append: !!(flags & constants.O_APPEND) });
				return fd;
			});
		},
		close(fd, cb) { done(cb, () => { const f = fdOf(fd); if (!f.dir) flush(f); fds.delete(fd); }); },
		fsync(fd, cb) { done(cb, () => { const f = fds.get(fd); if (f && !f.dir) flush(f); }); },
		read(fd, buf, offset, length, position, cb) {
			done(cb, () => {
				const f = fdOf(fd);
				if (f.dir) throw fail("EISDIR");
				const at = position === null ? f.pos : position;
				const n = Math.max(0, Math.min(length, f.data.length - at));
				buf.set(f.data.subarray(at, at + n), offset);
				if (position === null) f.pos += n;
				return n;
			});
		},
		write(fd, buf, offset, length, position, cb) {
			done(cb, () => {
				if (fd === 1 || fd === 2) return this.writeSync(fd, buf.subarray(offset, offset + length));
				const f = fdOf(fd);
				if (f.dir) throw fail("EISDIR");
				const at = f.append ? f.data.length : (position === null ? f.pos : position);
				if (at + length > f.data.length) {
					const grown = new Uint8Array(at + length);
					grown.set(f.data);
					f.data = grown;
				}
				f.data.set(buf.subarray(offset, offset + length), at);
				if (position === null) f.pos = at + length;
				f.dirty = true;
				return length;
			});
		},
		fstat(fd, cb) { done(cb, () => { const f = fdOf(fd); return f.dir ? stat(f.path) : { ...stat(f.path), size: f.data.length }; }); },
		stat(path, cb) { done(cb, () => { const s = stat(clean(path)); if (!s) throw fail("ENOENT"); return s; }); },
		lstat(path, cb) { this.stat(path, cb); },
		mkdir(path, perm, cb) {
			done(cb, () => {
				const p = clean(path);
				if (stat(p)) throw fail("EEXIST");
				if (!isDir(parent(p))) throw fail("ENOENT");
				localStorage.setItem(dirKey(p), "1");
			});
		},
		readdir(path, cb) {
			done(cb, () => {
				const p = clean(path);
				if (!isDir(p)) throw fail("ENOTDIR");
				const names = new Set();
				const prefix = p === "/" ? "/" : p + "/";
				for (let i = 0; i < localStorage.length; i++) {
					const k = localStorage.key(i);
					for (const kind of [fileKey(prefix), dirKey(prefix)]) {
						if (k.startsWith(kind)) names.add(k.substring(kind.length).split("/")[0]);
					}
				}
				return [...names].filter(n => n !== "");
			});
		},
		rename(from, to, cb) {
			done(cb, () => {
				const a = clean(from), b = clean(to);
				const f = readFile(a);
				if (!f) throw fail("ENOENT");
				writeFile(b, f.data);
				localStorage.removeItem(fileKey(a));
			});
		},
		unlink(path, cb) { done(cb, () => { const p = clean(path); if (!readFile(p)) throw fail("ENOENT"); localStorage.removeItem(fileKey(p)); }); },
		rmdir(path, cb) { done(cb, () => { localStorage.removeItem(dirKey(clean(path))); }); },
		ftruncate(fd, length, cb) { done(cb, () => { const f = fdOf(fd); const d = new Uint8Array(length); d.set(f.data.subarray(0, length)); f.data = d; f.dirty = true; }); },
		truncate(path, length, cb) { done(cb, () => { const p = clean(path); const f = readFile(p); if (!f) throw fail("ENOENT"); const d = new Uint8Array(length); d.set(f.data.subarray(0, length)); writeFile(p, d); }); },
		chmod(path, mode, cb) { cb(null); },
		chown(path, uid, gid, cb) { cb(null); },
		fchmod(fd, mode, cb) { cb(null); },
		fchown(fd, uid, gid, cb) { cb(null); },
		lchown(path, uid, gid, cb) { cb(null); },
		utimes(path, atime, mtime, cb) { cb(null); },
		link(path, link, cb) { cb(fail("ENOSYS")); },
		symlink(path, link, cb) { cb(fail("ENOSYS")); },
		readlink(path, cb) { cb(fail("ENOSYS")); },
	};
})();
