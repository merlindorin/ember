# Porting Taffy

Taffy c6421c1c, `src/` (about 30k lines of Rust). `go test -run TestTaffy -v` reports each suite's pass count; `TAFFY=<name>`, `TAFFY_VERBOSE=1` and `TAFFY_DUMP=1` narrow and explain it.

| Taffy source | ui | Status |
|---|---|---|
| `geometry.rs`, `util/{math,resolve}.rs`, `style/available_space.rs` | `geometry.go` (NaN is none), `style.go` | done |
| `tree/cache.rs`, `compute/{mod,leaf}.rs` | `cache.go`, `compute.go`, `leaf.go` | done; leaf 30/30 |
| `compute/block.rs` | `block.go` | done, without floats; block 484/484 |
| `compute/flexbox.rs` | `flex.go` | done; flex 1232/1232 |
| `compute/grid/*` | `grid*.go` | done; grid 1076/1076 |
| `compute/oof.rs`, `compute/common/*` | `oof.go`, `common.go` | done |

Counts are of the supported fixtures. Left out until style can say them: alignment safety (`safe`/`unsafe`), `self-start`/`self-end`, grid line names, `stretch` sizes. blockflex is 24/26: `flex-basis: 0%` needs a zero percentage kept apart from `0px`.

Differences from Taffy, each one where Chrome disagrees with it:

- Text is a block container's inline content, not a measured leaf, so a text element in a flex or grid container has an anonymous block item. A single-line flex row measured in definite space shrinks to fit (`minContentMainSize`), as CSS's fit-content does.
- A min or max size transferred through `aspect-ratio` never goes into an axis whose size is already known.
- `Layout` keeps fractional positions, matching Chrome to 0.03 pt; rounding is only for the fixtures that ask for it.

Beyond Taffy (`inline.go`), with Chrome cases in `chrome_test.go`:

- line boxes, wrapping at spaces and U+200B, `white-space`, `text-overflow: ellipsis`, `line-height`, `text-align` (start, end, left, right, center), spans, and atomic inlines (inline-block, inline-flex, inline-grid) on the baseline.

Still to do: `position: sticky`, `z-index` ordering data, clamping `Box.Scroll`, `order`, `overflow-wrap`, padding, borders and margins of inline elements, `text-align: justify`, and relayout of only what's dirty (each `Layout` builds the tree anew).

Not ported: floats, `flex-wrap: balance`, `contain`, right-to-left, vertical writing, Taffy's `TaffyTree` storage and its `calc()` pointer (calc arrives resolved from css).
