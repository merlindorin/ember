// Package net is server-authoritative networking for ember.
//
// The server World owns the simulation. Entities that embed a Network
// component and a pose, a scene.Transform in 3D or a twod placement in 2D,
// are replicated to clients: the server sends spawns, pose and state
// updates and despawns at a fixed tick rate, filtered by an interest
// radius around each client's focus entity, measured on the ground plane
// (x and z in 3D, x and y in 2D). Clients mirror those entities with
// factories registered per Kind, smooth their poses, and talk back with
// typed messages.
//
// Client messages are intents: Client.Send numbers them, the server queues
// them on arrival and applies them at the start of the next FixedUpdate step,
// and every snapshot acknowledges the last sequence number applied, so a
// simulation stepped at a fixed rate gives the same result at any frame rate.
//
// The same game runs single-player through an integrated server connected
// with Pipe, or online with a dedicated server behind Listen:
//
//	srvConn, cliConn := net.Pipe()
//	srv := net.NewServer()
//	server.AddPlugins(srv)
//	srv.Connect(srvConn)
//	client.AddPlugins(net.NewClient(cliConn))
package net

import (
	"encoding/gob"
	"hash/fnv"
	"reflect"
	"sync"

	"github.com/lantervale/ember/geom"
	"github.com/lantervale/ember/world"
)

// ID identifies a replicated entity across all peers: the region that
// owns it in the high 16 bits and a counter in the low 48.
type ID uint64

const counterBits = 48

// MaxCounter is the largest counter an ID can hold.
const MaxCounter = 1<<counterBits - 1

// PlacedRegion holds the IDs of entities placed in a level that the
// server and its clients both load: their IDs come from the level, not
// from a server counter.
const PlacedRegion uint16 = 0xFFFF

// PlacedID is the ID of a placed entity, from a key every peer derives
// alike, such as its scene node IDs.
func PlacedID(key string) ID {
	h := fnv.New64a()
	_, _ = h.Write([]byte(key))
	return NewID(PlacedRegion, h.Sum64())
}

func NewID(region uint16, counter uint64) ID {
	return ID(uint64(region)<<counterBits | counter&MaxCounter)
}

func (id ID) Region() uint16  { return uint16(id >> counterBits) }
func (id ID) Counter() uint64 { return uint64(id) & MaxCounter }

// ClientID identifies a connection on the server. 0 is the server itself.
type ClientID uint32

// Network marks an entity as replicated.
type Network struct {
	NetID ID
	Kind  string
	Owner ClientID
}

func (n *Network) GetNetwork() *Network { return n }

// Replicated is any entity the server sends to clients. It must also have
// a pose: a scene.Transform, or a twod placement (a twod.Item); net
// panics on one with neither when it joins a World.
type Replicated interface {
	world.BasicFace
	GetNetwork() *Network
}

// Stateful entities replicate extra gameplay state alongside their
// transform. NetState must return a gob-encodable value registered with
// Register and comparable with ==; it is only sent when it changes.
type Stateful interface {
	NetState() any
	ApplyNetState(state any)
}

// OwnerStateful entities also replicate private state, such as an
// inventory, to their owning client only. The same rules as NetState
// apply: registered, comparable, sent on change.
type OwnerStateful interface {
	OwnerState() any
	ApplyOwnerState(state any)
}

// ClientConnected is emitted on the server, during FixedUpdate, when a
// client joins.
type ClientConnected struct{ ID ClientID }

// ClientDisconnected is emitted on the server, during FixedUpdate, when a
// client leaves. Ack is the last intent applied for it.
type ClientDisconnected struct {
	ID  ClientID
	Ack uint32
}

// ServerDisconnected is emitted on the client when the server connection drops.
type ServerDisconnected struct{}

//nolint:gochecknoglobals // process-wide registries behind RegisterKind and Register, like gob's
var (
	kindsMu sync.RWMutex
	kinds   = map[string]func() Replicated{}

	registered sync.Map
)

// RegisterKind tells clients how to build the mirror of a replicated entity.
// The factory must return an entity with its BasicEntity set.
func RegisterKind(kind string, factory func() Replicated) {
	kindsMu.Lock()
	defer kindsMu.Unlock()
	kinds[kind] = factory
}

func kindFactory(kind string) (func() Replicated, bool) {
	kindsMu.RLock()
	defer kindsMu.RUnlock()
	f, ok := kinds[kind]
	return f, ok
}

// Register makes a message or state type encodable. Handle, HandleClient
// and Send register automatically; call it in a shared init for types
// that only one side of a split binary sends.
func Register(v any) {
	t := reflect.TypeOf(v)
	if t == nil {
		return
	}
	if _, loaded := registered.LoadOrStore(t, true); !loaded {
		gob.Register(v)
	}
}

// The wire messages below are what a Codec carries: the server sends a
// Welcome then Snapshots, the client sends Intents. Any other value is an
// application message (Send, Broadcast, Handle).

// Pose is where an entity is. A 2D pose is on the ground plane: (x, y) is
// (x, 0, y), its rotation a yaw and its scale (sx, 1, sy).
type Pose struct {
	Position geom.Vec3
	Rotation geom.Quat
	Scale    geom.Vec3
}

type Welcome struct{ ID ClientID }

// Resumed follows Welcome when the server handed this connection a session
// left by a previous one (a reconnect): LastSeq is the last intent it
// applied from that connection.
type Resumed struct {
	Client  ClientID
	LastSeq uint32
}

type Spawn struct {
	ID      ID
	Kind    string
	Owner   ClientID
	Pose    Pose
	State   any
	Private any
}

type Update struct {
	ID         ID
	Pose       Pose
	State      any
	HasState   bool
	Private    any
	HasPrivate bool
	Owner      ClientID
	HasOwner   bool
}

type Intent struct {
	Seq uint32
	Msg any
}

// Poses carries entity poses in datagrams, on links that offer them: lost
// and reordered datagrams cost nothing, since each pose is absolute and is
// sent again until a datagram carrying it is acknowledged. Ack is the last
// intent applied when the poses were taken, for prediction.
type Poses struct {
	Seq     uint32
	Tick    uint64
	Ack     uint32
	Updates []Update
}

// PosesAck acknowledges datagrams: Last is the newest Poses.Seq received,
// bit n of Mask stands for Last-1-n.
type PosesAck struct {
	Last uint32
	Mask uint64
}

type Snapshot struct {
	// Tick is the server's fixed step when the snapshot was taken.
	Tick uint64
	// Ack is the last intent sequence number the server applied.
	Ack      uint32
	Spawns   []Spawn
	Updates  []Update
	Despawns []ID
}

func init() { //nolint:gochecknoinits // the wire messages are registered with gob before any peer decodes
	Register(Poses{})
	Register(PosesAck{})
	Register(Welcome{})
	Register(Resumed{})
	Register(Intent{})
	Register(Snapshot{})
}
