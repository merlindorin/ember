# 0014. The guide teaches by example

- Status: Accepted
- Date: 2026-10-28
- Area: docs
- Owner: user

## Context

The public API is reviewed one example at a time, and docs written apart from the code drift.

## Decision

- Each guide page is one sentence or two and one complete example, in the spirit of Gin's docs. Its code and output are included from `examples/` when the site is built, and a test checks them.
- `ember/docs` is plain Markdown that reads on GitHub. The Hugo and Hextra site lives in `.hugo`.
- A page appears when its example is reviewed. Engine decisions are the guide's last section.

## Alternatives

- Hand-written API docs: more freedom, but they drift from the code.
- A full site planned up front: a complete map, but empty pages and premature names.

## Consequences

The examples are the source of truth, and the guide grows with the review.
