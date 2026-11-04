# 0059. ember/ui is uiv2: the old package removed, uiv2 renamed ui

- Status: Accepted
- Date: 2026-11-03
- Area: api
- Owner: user

## Context

[0049](0049-ui-is-modern-html-elements-and-a-css-subset-written-from-go.md) kept `ember/ui` and `uiv2` side by side until the last screen moved. Every demo screen, tool and example now draws with uiv2, so two UI packages only cost upkeep and confuse which one a game should use.

## Decision

- `ember/ui` is deleted, then `ember/uiv2` is renamed `ember/ui` (its `kit` with it). The `kit-` and `ember-` CSS class prefixes stay.
- `ember.New` no longer installs a UI plugin: a UI is a `Document` the game adds to a window, drawn by twod as [0017](0017-ui-is-built-on-2d.md) has it.
- `ember.WithTheme` and `ember.Translator` go. A theme is a Document's sheets (`kit.Theme()`, `Stylesheet`, `CSS`) and a translator is the Document option `WithTranslator`, given to each Document.
- The guide's old UI section goes; a migration table maps each old call to its new one.

## Alternatives

- Keep `ember/ui` frozen beside the new package: no breakage for outside games, but two APIs to document, test and keep cgo-free, and new code may pick the wrong one.
- Delete `ember/ui` but keep the name `uiv2`: one rename fewer, but a permanent "v2" in every import for a package with no v1.

## Consequences

One UI package, styled by CSS and checked against Chrome. Games written against `ember/ui` must migrate by the table; nothing outside this repository is known to use it.
