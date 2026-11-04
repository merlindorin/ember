package world //nolint:testpackage // white-box: the unexported systems sort

import (
	"sort"
	"testing"
)

type mockSystem struct {
	name     string
	phase    Phase
	priority int
}

func (m *mockSystem) Update(Frame)      {}
func (m *mockSystem) Remove(Identifier) {}
func (m *mockSystem) Phase() Phase      { return m.phase }
func (m *mockSystem) Priority() int     { return m.priority }

func TestSystemsSort_ByPhase(t *testing.T) {
	sysList := systems{
		&mockSystem{name: "fixed", phase: FixedUpdate, priority: 0},
		&mockSystem{name: "startup", phase: Startup, priority: 0},
		&mockSystem{name: "update", phase: Update, priority: 0},
		&mockSystem{name: "preupdate", phase: PreUpdate, priority: 0},
		&mockSystem{name: "postupdate", phase: PostUpdate, priority: 0},
	}

	sort.Sort(sysList)

	expected := []string{"startup", "preupdate", "fixed", "update", "postupdate"}
	for i, sys := range sysList {
		if m, ok := sys.(*mockSystem); !ok || m.name != expected[i] {
			t.Errorf("Expected %s at index %d, got %s", expected[i], i, m.name)
		}
	}
}

func TestSystemsSort_ByPriorityWithinPhase(t *testing.T) {
	sysList := systems{
		&mockSystem{name: "update_low", phase: Update, priority: 1},
		&mockSystem{name: "update_high", phase: Update, priority: 100},
		&mockSystem{name: "update_mid", phase: Update, priority: 50},
	}

	sort.Sort(sysList)

	expected := []string{"update_high", "update_mid", "update_low"}
	for i, sys := range sysList {
		if m, ok := sys.(*mockSystem); !ok || m.name != expected[i] {
			t.Errorf("Expected %s at index %d, got %s", expected[i], i, m.name)
		}
	}
}

func TestSystemsSort_PhaseOverridesPriority(t *testing.T) {
	sysList := systems{
		&mockSystem{name: "render_high", phase: PostUpdate, priority: 1000},
		&mockSystem{name: "startup_low", phase: Startup, priority: 1},
	}

	sort.Sort(sysList)

	if m, ok := sysList[0].(*mockSystem); !ok || m.name != "startup_low" {
		t.Errorf("Expected startup_low first (phase takes precedence), got %s", m.name)
	}
	if m, ok := sysList[1].(*mockSystem); !ok || m.name != "render_high" {
		t.Errorf("Expected render_high second, got %s", m.name)
	}
}

func TestSystemsSort_MixedPhasesAndPriorities(t *testing.T) {
	sysList := systems{
		&mockSystem{name: "render_1", phase: PostUpdate, priority: 10},
		&mockSystem{name: "startup_1", phase: Startup, priority: 50},
		&mockSystem{name: "update_1", phase: Update, priority: 100},
		&mockSystem{name: "startup_2", phase: Startup, priority: 100},
		&mockSystem{name: "render_2", phase: PostUpdate, priority: 50},
		&mockSystem{name: "update_2", phase: Update, priority: 50},
	}

	sort.Sort(sysList)

	expected := []string{
		"startup_2",
		"startup_1",
		"update_1",
		"update_2",
		"render_2",
		"render_1",
	}

	for i, sys := range sysList {
		if m, ok := sys.(*mockSystem); !ok || m.name != expected[i] {
			t.Errorf("Expected %s at index %d, got %s", expected[i], i, m.name)
		}
	}
}

type systemWithoutPhase struct {
	name     string
	priority int
}

func (s *systemWithoutPhase) Update(Frame)      {}
func (s *systemWithoutPhase) Remove(Identifier) {}
func (s *systemWithoutPhase) Priority() int     { return s.priority }

func TestSystemsSort_DefaultPhaseIsUpdate(t *testing.T) {
	sys1 := &systemWithoutPhase{name: "no_phase", priority: 50}
	sys2 := &mockSystem{name: "explicit_update", phase: Update, priority: 100}

	sysList := systems{sys1, sys2}
	sort.Sort(sysList)

	if sysList[0] != sys2 {
		t.Error("Expected explicit_update first (higher priority)")
	}
	if sysList[1] != sys1 {
		t.Error("Expected no_phase second (lower priority)")
	}
}

func TestSystemsSort_NoPriorityDefaultsToZero(t *testing.T) {
	type systemWithoutPriority struct {
		mockSystem
	}

	sys1 := &systemWithoutPriority{mockSystem{name: "no_priority", phase: Update}}
	sys2 := &mockSystem{name: "with_priority", phase: Update, priority: 10}

	sysList := systems{sys1, sys2}
	sort.Sort(sysList)

	if sysList[0] != sys2 {
		t.Error("Expected with_priority first (higher priority)")
	}
	if sysList[1] != sys1 {
		t.Error("Expected no_priority second (default priority 0)")
	}
}
