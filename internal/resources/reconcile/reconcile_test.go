package reconcile

import (
	"reflect"
	"testing"
)

func TestDiffStrings(t *testing.T) {
	tests := map[string]struct {
		desired      []string
		current      []string
		wantToAdd    []string
		wantToRemove []string
	}{
		"keeps overlapping values": {
			desired:      []string{"b", "c"},
			current:      []string{"a", "b"},
			wantToAdd:    []string{"c"},
			wantToRemove: []string{"a"},
		},
		"deduplicates and sorts output": {
			desired:      []string{"c", "b", "c", "a"},
			current:      []string{"d", "b", "d"},
			wantToAdd:    []string{"a", "c"},
			wantToRemove: []string{"d"},
		},
		"handles empty desired": {
			desired:      nil,
			current:      []string{"b", "a"},
			wantToRemove: []string{"a", "b"},
		},
		"handles empty current": {
			desired:   []string{"b", "a"},
			current:   nil,
			wantToAdd: []string{"a", "b"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			gotToAdd, gotToRemove := DiffStrings(tt.desired, tt.current)
			if !reflect.DeepEqual(gotToAdd, tt.wantToAdd) {
				t.Fatalf("toAdd mismatch: got %#v want %#v", gotToAdd, tt.wantToAdd)
			}
			if !reflect.DeepEqual(gotToRemove, tt.wantToRemove) {
				t.Fatalf("toRemove mismatch: got %#v want %#v", gotToRemove, tt.wantToRemove)
			}
		})
	}
}
