# 0049. UI is modern HTML elements and a CSS subset, written from Go

- Status: Accepted
- Date: 2026-11-02
- Area: api
- Owner: user

## Context

`ember/ui` lays screens out with hand-set widths, and nothing keeps content inside its parent. Every screen of the tileset tool needed rounds of alignment and overflow fixes.

## Decision

- A new package, `ember/uiv2`, builds trees of modern HTML elements from Go. It styles them with a modern CSS subset (flex, grid, intrinsic sizes, overflow, the cascade) and draws through `twod.Canvas`. Pure Go.
- Write-only: no HTML parser, no DOM queries, no scripting.
- Layout ports Taffy's block, flex and grid algorithms.
- Headless Chrome is the test reference: an exported tree's boxes must match ours.
- It replaces `ember/ui`'s layout screen by screen, starting with the tileset inspector (design: `docs/design/uiv2.md`).

## Alternatives

- Keep `ember/ui` and add automatic overflow checks: it catches bugs late, and doesn't prevent them.
- Add flex or grid containers to `ember/ui` piecemeal: a second, home-made layout model with no reference to test against.
- Embed a browser engine (CEF, WebView): huge, cgo, no servers or consoles, and UI no longer drawn by the engine.

## Consequences

Layout means what CSS means, and Chrome checks it. In return, ember owns a sizeable layout engine and a cascade, and `ember/ui` and `uiv2` live side by side until the last screen moves.
