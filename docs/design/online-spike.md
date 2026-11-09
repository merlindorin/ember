# Online spike: Lantervale headless, two clients

Status: proposal for the CTO, no code before approval. Base: `wip` 24292a3b.

## Goal

Find out what it costs to run Lantervale's simulation as a server. The spike is done when:

- `lantervale/server` builds with `CGO_ENABLED=0` and runs the game's World on the headless ticker, with no window.
- Two clients join it on a small test map (one Tiled room). Each player walks their own hero, sees the other hero move,
  and one monster chases whichever hero is nearer.
- A GPU-free test runs all of it in one process: the server World and two client Worlds over `net.Lag` at 150 ms. It
  checks that each client sees both heroes, that its own hero answers at once (prediction), and that the monster
  changes target.

Out of scope: persistence, accounts, combat beyond one chasing monster, loot, screens, the run loop (pause, results,
power-ups), QUIC, and any change to how the solo game plays.

## What the game already gives

- **The simulation is pure Go.** Every Lantervale package builds with `CGO_ENABLED=0` (`game`, `hero`, `horde`,
  `arsenal`, `loot`, `bonus`, `gear`, `items`, `stats`, `dungeon`, `arena`, `progress`, `fog`, `lighting`, `feel`,
  `hud`, `screens`, `heroes`, `data`). Only `main.go` needs cgo, through `ember.New`, which installs `gpu.Plugin`.
- **Input already goes through an intent.** `hero/read` turns the window's input into `game.Intent` (Walk, Aim, Goal,
  Dash, Cast…), and the gameplay systems read only the intent in FixedUpdate. The intent is the right thing to send
  over the wire.
- **Gameplay is in FixedUpdate**, with a seeded run (`RunStarted{Seed}`), as decision 0002 requires.

## What the game must change

These are in lantervale-dev's paths: their seat writes them, or the CTO hands them to me for the spike.

1. **One hero per player instead of "the hero".** 27 calls `game.First(heroes.All())` in 22 files assume there is one
   hero. The spike changes only the ones the test map runs: `hero/` (read, walk, use) and `horde/chase.go` (nearest
   hero). The others stay single-hero until a later step.
2. **The intent on the wire.** `game.Intent` holds `Target *Enemy` and `Waypoints`. Pointers can't be sent: the
   message carries the target's `net.ID`, and the server finds the waypoints itself. `hero/read` writes the intent
   locally in solo play, and online it sends the intent with `Client.Send`.
3. **Kinds for mirrors.** `net.RegisterKind("hero", …)` and `("monster", …)` build the same game types on the client
   (sprite, look, light), so `hud`, `lighting` and `fog` draw the mirrors unchanged.
4. **A `lantervale/server` main.** A World with the gameplay plugins only (arena's map and collision, hero, horde), no
   window, no screens, no hud, no feel.

## What ember/net already gives

- **The server is authoritative (0001):** `net.NewServer` and `net.NewClient` are plugins. Intents are numbered,
  applied at the next fixed step and acknowledged in every snapshot. `ClientConnected` and `ClientDisconnected` are
  events.
- **Replication by interest:** spawns, updates and despawns within a radius of each client's focus entity, on a
  spatial grid, with a per-snapshot budget. `Stateful` sends public state when it changes, and `OwnerStateful` sends
  private state (a bag, say) to its owner only.
- **Prediction (0002):** `Predicted` (PredictIntent, PredictStep), replay of unacknowledged intents, blended
  corrections. Remote entities are interpolated, never extrapolated.
- **Transports (0005):** `Pipe`, TCP or TLS through `Listener`, QUIC datagrams in `net/quic`, and `Lag`/`LagJitter` for
  tests. Reconnects resume a session (`Resumed`). `Hello` carries a join ticket.
- **Placed entities:** `PlacedID(key)` gives entities that both sides load from a map (doors, chests) the same ID.
  `Counter` and `Resume` keep IDs unique across a shard restart.
- **Tests:** `net/nettest`, and `node/node3d` already replicates a 3D world.

## What ember is missing

Both are in my paths and generic. Each would get a decision when it lands.

- **2D poses.** `net.Replicated` needs a `scene.Transform` (a 3D pose: Vec3, Quat, Vec3, 40 bytes). A Lantervale hero
  is a `twod.Item` with a `twod.Placement` (Vec2, angle, scale). Proposal: net replicates anything with a pose
  through a small interface, with 2D poses on the wire (Vec2 and angle, 12 bytes), and the interest distance measured
  on the plane. 3D keeps its current pose. This touches `twod` only to read `Placement`, which is engine-render's
  package, so read-only.
- **A pure `New` for servers.** `ember.New` imports `gpu`, so a server can't use it. Proposal: a constructor in a
  cgo-free package (`ember/server`, or `world` with the pure plugins: render's description, anim, ease and audio to a
  null output), and the headless ticker as its runner. The name is up to the CTO.

## Risks

- **The run is single-player.** `game.Run` holds Paused, Over, Picking and InMenu. Pausing and power-up choices stop
  the World, and `RunStarted` resets everything. None of that works shared. The spike avoids it (the server runs no
  screens, and nothing pauses), but online play needs a game design answer: what stops for whom.
- **Prediction needs the movement code on the client.** `PredictStep` must reuse `hero/walk` and `collide.Space.Move`
  with the same map on both sides. If movement reads anything not replicated (a buff, the sheet's speed), predictions
  snap back. The spike will show how often.
- **Units.** `net.Interest` defaults (radius 60, margin 8) are 3D metres, while Lantervale works in pixels. The values
  are game options, but the defaults would be wrong for any 2D game.
- **Bandwidth.** The codec is gob. That's fine for two clients. A horde of hundreds needs a compact codec and 2D poses.
  I'll measure bytes per snapshot in the spike test and report it, without optimising.
- **Machine load.** The spike test runs on one process with no GPU. A manual run (server plus two windows) happens
  once, at load < 10.

## Size

| Part                                                                            | Who                    | Size     |
|---------------------------------------------------------------------------------|------------------------|----------|
| net: 2D poses, plane distance, tests                                            | engine-core            | 1 day    |
| A pure constructor for servers, with a test that it builds with `CGO_ENABLED=0` | engine-core            | 0.5 day  |
| Game: per-player hero in `hero/` and `horde/chase`, intent message, kinds       | lantervale-dev (or me) | 1.5 days |
| `lantervale/server`, test map, the two-client Lag test, bytes per snapshot      | engine-core            | 1 day    |

About 4 days of one engineer, on one branch. The engine parts land first, with docs to doc-writer. The game parts
land after lantervale-dev approves them.

## Open questions for the CTO

1. Who writes the game-side changes (items 1–3): lantervale-dev, or me inside their paths with their review?
2. Where does the server constructor live: `ember/server`, or an option on a pure `world` setup?
3. Should the spike stay a test and a server binary, or should the game also get a `-connect host` flag to play it by
   hand in two windows?
4. Can I take decision numbers for 2D poses in net and the pure constructor when they land (next free: 0063 on the
   board, but 0063 is already taken in `99-decisions/`)?
