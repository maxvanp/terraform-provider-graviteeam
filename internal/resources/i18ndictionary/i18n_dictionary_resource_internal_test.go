package i18ndictionary

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestParseImportID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		id               string
		wantDomainID     string
		wantDictionaryID string
		wantOK           bool
	}{
		{
			name:             "valid",
			id:               "domain-1/dict-1",
			wantDomainID:     "domain-1",
			wantDictionaryID: "dict-1",
			wantOK:           true,
		},
		{
			name:             "preserves splitN behavior",
			id:               "domain-1/dict-1/extra",
			wantDomainID:     "domain-1",
			wantDictionaryID: "dict-1/extra",
			wantOK:           true,
		},
		{
			name:   "missing separator",
			id:     "domain-1",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotDomainID, gotDictionaryID, gotOK := parseImportID(tt.id)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %t, want %t", gotOK, tt.wantOK)
			}
			if gotDomainID != tt.wantDomainID {
				t.Fatalf("domain ID = %q, want %q", gotDomainID, tt.wantDomainID)
			}
			if gotDictionaryID != tt.wantDictionaryID {
				t.Fatalf("dictionary ID = %q, want %q", gotDictionaryID, tt.wantDictionaryID)
			}
		})
	}
}

func TestBuildBody(t *testing.T) {
	t.Parallel()

	plan := I18nDictionaryModel{
		Name:   types.StringValue("French"),
		Locale: types.StringValue("fr"),
	}

	got := buildBody(plan)
	want := map[string]interface{}{
		"name":   "French",
		"locale": "fr",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestReadIntoModel(t *testing.T) {
	t.Parallel()

	model := I18nDictionaryModel{
		Name:   types.StringValue("old-name"),
		Locale: types.StringValue("old-locale"),
	}

	diags := readIntoModel(context.Background(), &model, map[string]interface{}{
		"name":   "French",
		"locale": "fr",
		"entries": map[string]interface{}{
			"login.title": "Connexion",
			"ignored":     12,
			"empty":       "",
		},
	})
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}

	if got, want := model.Name.ValueString(), "French"; got != want {
		t.Fatalf("name = %q, want %q", got, want)
	}
	if got, want := model.Locale.ValueString(), "fr"; got != want {
		t.Fatalf("locale = %q, want %q", got, want)
	}

	entries := make(map[string]string)
	diags = model.Entries.ElementsAs(context.Background(), &entries, false)
	if diags.HasError() {
		t.Fatalf("entry diagnostics: %v", diags)
	}
	wantEntries := map[string]string{
		"login.title": "Connexion",
		"empty":       "",
	}
	if !reflect.DeepEqual(entries, wantEntries) {
		t.Fatalf("entries = %#v, want %#v", entries, wantEntries)
	}
}

func TestReadIntoModelLeavesEntriesWhenAPIReturnsEmptyEntries(t *testing.T) {
	t.Parallel()

	existing, diags := types.MapValueFrom(context.Background(), types.StringType, map[string]string{
		"existing": "value",
	})
	if diags.HasError() {
		t.Fatalf("setup diagnostics: %v", diags)
	}

	model := I18nDictionaryModel{
		Entries: existing,
	}

	diags = readIntoModel(context.Background(), &model, map[string]interface{}{
		"entries": map[string]interface{}{},
	})
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}

	entries := make(map[string]string)
	diags = model.Entries.ElementsAs(context.Background(), &entries, false)
	if diags.HasError() {
		t.Fatalf("entry diagnostics: %v", diags)
	}
	if want := map[string]string{"existing": "value"}; !reflect.DeepEqual(entries, want) {
		t.Fatalf("entries = %#v, want preserved %#v", entries, want)
	}
}
