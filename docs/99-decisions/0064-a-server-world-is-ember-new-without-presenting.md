# 0064. A server World is ember.New without presenting

- Status: Accepted
- Date: 2026-11-09
- Area: net
- Owner: architect

## Context

A game server must build with `CGO_ENABLED=0` (0004), but `ember.New` installs the GPU, which also installs twod, its
lights and the ui systems, so a server built from `world.New()` simulated less than its clients did.

## Decision

- `server.New(opts…)`, in the package `ember/server`, returns a World with every simulation plugin `ember.New`
  installs (windows' input frames, transforms, twod and its steps, lights, animation, cameras, tweens, audio's null
  output, automation), without the GPU or the desktop runner. It runs on the headless ticker.
- Its one option is `server.Automation(auto)`, as in `ember.New`.
- A test in the ember package fails when `ember.New` installs a system outside `gpu` or `platform/desktop` that
  `server.New` does not, and another fails when `ember/server` depends on cgo.

## Alternatives

- **`world.New()` and the plugins by hand:** no new package, but every game server would repeat the list and drift from
  the client's.
- **A `Server()` option on `ember.New`:** one constructor, but the `ember` package imports `gpu`, so a server binary
  would still need cgo.
- **Moving the pure list into `world`:** no new package, but `world` would import twod, audio and automation, which
  import `world`.

## Consequences

A server and its clients run the same systems, so a game's results don't depend on which side runs them. A server pays
for twod's and light's systems even when it adds no 2D entities or lights, which costs nothing while those groups are
empty.
