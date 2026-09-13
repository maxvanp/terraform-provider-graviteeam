package importid

import "testing"

func TestValid(t *testing.T) {
	tests := []struct {
		name  string
		parts []string
		want  bool
	}{
		{name: "valid", parts: []string{"domain-1", "resource-1"}, want: true},
		{name: "empty", parts: []string{"domain-1", ""}},
		{name: "whitespace", parts: []string{"domain-1", " \t"}},
		{name: "no parts", parts: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Valid(tt.parts); got != tt.want {
				t.Fatalf("Valid(%q) = %t, want %t", tt.parts, got, tt.want)
			}
		})
	}
}
