# 0010. Studio is the official tool

- Status: Superseded by [0016](0016-the-editor-and-studio-are-removed-for-now.md)
- Date: 2026-10-27
- Area: tools
- Owner: user

## Context

Model viewing, scene editing and live inspection were heading towards separate tools and example viewers.

## Decision

- `tools/studio` is the one editor: it views every format `asset` reads, edits scenes, and edits a running game through `SceneService`.
- Examples stay small and never grow into tools.
- ember commands follow the user's kong CLI conventions, and keep their data in `~/.ember`.

## Alternatives

- A viewer per format, or editors in examples: quicker to start, but duplicated and unmaintained.

## Consequences

Format support and editing features arrive in one place.
