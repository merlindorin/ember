---
title: ember
layout: hextra-home
cascade:
  type: docs
---

{{< ember/hero image="06-2d/19-lights/point.gif" alt="A night garden lit by a flickering torch: trees and pillars cast soft shadows, a hero walks between them" caption="go run ./examples/36-point-lights" >}}

{{< hextra/hero-badge link="https://github.com/lantervale/ember" >}}
Pure Go · 2D · UI · ECS
{{< /hextra/hero-badge >}}

{{< hextra/hero-headline >}}
Game worlds in plain&nbsp;Go
{{< /hextra/hero-headline >}}

{{< hextra/hero-subtitle >}}
ember is a Go game engine. Sprites, text, tile maps, particles, windows and UI are entities you build with options, and the systems you write run on them, in a window or on a headless server.
{{< /hextra/hero-subtitle >}}

<div class="ember-actions">
{{< hextra/hero-button text="Read the guide" link="docs" >}}
{{< hextra/hero-button text="See the 2D pages" link="docs/2d" >}}
</div>

{{< /ember/hero >}}

{{< ember/section eyebrow="A first program" title="A window, a panel and a system" lead="A whole game: the window and its UI are entities made with options, and `Counter` reads the buttons' `Clicked` events to change the count." >}}

{{< ember/code >}}
{{< ember/labelled label="The whole `main`" >}}{{< file "11-ui-elements/main.go" >}}{{< /ember/labelled >}}
<div class="ember-stack">
{{< ember/labelled label="The system that counts" >}}{{< file "11-ui-elements/counter.go" >}}{{< /ember/labelled >}}
{{< ember/window image="05-ui/09-walkthroughs/counter.png" alt="The counter: a panel with a title, the count and two buttons" caption="go run ./examples/11-ui-elements" >}}
</div>
{{< /ember/code >}}

{{< ember/section eyebrow="The guide" title="What you get" lead="Each section is a set of pages, and each page walks through a complete example." >}}

{{< ember/grid class="ember-row" >}}
{{< ember/feature title="The World" icon="globe-alt" link="/docs/world" output="04-events/output.txt" >}}
Entities, systems, events and children, and a fixed step that counts the same at any frame rate.
{{< /ember/feature >}}
{{< ember/feature title="Windows" icon="view-grid" link="/docs/windows" image="home/windows.gif" >}}
A window is an entity, with its own cameras, input and UI. Resize it and a pixel-art world keeps its pixels square.
{{< /ember/feature >}}
{{< ember/feature title="UI" icon="template" link="/docs/ui" image="05-ui/09-walkthroughs/themes.gif" >}}
Panels, buttons, sliders and text inputs, laid out in rows and columns, styled by themes.
{{< /ember/feature >}}
{{< ember/feature title="2D" icon="photograph" link="/docs/2d" image="06-2d/tiled.gif" >}}
Sprites, tile maps, collision, cameras, shaders, particles and tweens.
{{< /ember/feature >}}
{{< /ember/grid >}}

{{< ember/section eyebrow="Why ember" title="Plain code, all the way down" >}}

{{< ember/grid class="ember-row ember-why" >}}
{{< ember/feature title="Pure-Go servers" icon="server" >}}
The World, 2D, UI and collision don't need cgo, so a game server builds with `CGO_ENABLED=0` and runs the same systems as the game.
{{< /ember/feature >}}
{{< ember/feature title="One kind of thing" icon="cube" >}}
A sprite, a button, a window and a camera are all entities, built with options and changed with setters.
{{< /ember/feature >}}
{{< ember/feature title="Tested pages" icon="beaker" >}}
Every page shows a real example, file by file, and every example has its tests. If the code changes, the page does too.
{{< /ember/feature >}}
{{< ember/feature title="Nothing hidden" icon="eye" >}}
The engine adds no global state of its own. Services are values you make and pass in, so you always see what a game uses.
{{< /ember/feature >}}
{{< /ember/grid >}}
