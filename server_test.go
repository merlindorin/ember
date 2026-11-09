package ember_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/merlindorin/ember"
	"github.com/merlindorin/ember/server"
	"github.com/merlindorin/ember/world"
)

func TestAServerSimulatesLikeAClient(t *testing.T) {
	client := systemsOf(ember.New(ember.Headless()))
	srv := systemsOf(server.New())
	for name := range srv {
		if !client[name] {
			t.Errorf("server.New installs %s, ember.New does not", name)
		}
	}
	for name := range client {
		if !srv[name] && !strings.Contains(name, "/gpu") && !strings.Contains(name, "/platform/desktop") {
			t.Errorf("ember.New installs %s, server.New does not", name)
		}
	}
}

func systemsOf(w *world.World) map[string]bool {
	names := map[string]bool{}
	for _, s := range w.Systems() {
		t := reflect.TypeOf(s)
		for t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		name := t.PkgPath() + "." + t.Name()
		if n, ok := s.(interface{ Name() string }); ok {
			name += " " + n.Name()
		}
		names[name] = true
	}
	return names
}
