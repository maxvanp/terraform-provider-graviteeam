package i18ndictionary

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewI18nDictionaryResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_i18n_dictionary"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewI18nDictionaryResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	assertStringAttribute(t, resp.Schema.Attributes, "id", false, false, true)
	assertStringAttribute(t, resp.Schema.Attributes, "domain_id", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "name", true, false, false)
	assertStringAttribute(t, resp.Schema.Attributes, "locale", true, false, false)
	assertMapAttribute(t, resp.Schema.Attributes, "entries", types.StringType)
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&I18nDictionaryResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not a client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected diagnostics for unexpected provider data")
	}
}

func TestConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&I18nDictionaryResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
	}
}

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

func TestI18nDictionaryCRUDManagesEntriesSeparately(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "test-token",
			"token_type":   "bearer",
			"expires_in":   3600,
		})
	})

	var dictionaryBodies []map[string]interface{}
	var entriesBodies []map[string]string
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/i18n/dictionaries", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected dictionary collection method %s", r.Method)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create body: %v", err)
		}
		dictionaryBodies = append(dictionaryBodies, body)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     "dict-123",
			"name":   body["name"],
			"locale": body["locale"],
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/i18n/dictionaries/dict-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":     "dict-123",
				"name":   "French",
				"locale": "fr",
				"entries": map[string]interface{}{
					"login.title": "Connexion",
				},
			})
		case http.MethodPut:
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			dictionaryBodies = append(dictionaryBodies, body)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":     "dict-123",
				"name":   body["name"],
				"locale": body["locale"],
			})
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected dictionary item method %s", r.Method)
		}
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/i18n/dictionaries/dict-123/entries", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("unexpected dictionary entries method %s", r.Method)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode entries body: %v", err)
		}
		entriesBodies = append(entriesBodies, body)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"entries": body})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &I18nDictionaryResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createEntries := mapValue(t, map[string]string{"login.title": "Connexion"})
	createPlan := i18nDictionaryPlan(t, schemaResp.Schema, I18nDictionaryModel{
		DomainID: types.StringValue("domain-123"),
		Name:     types.StringValue("French"),
		Locale:   types.StringValue("fr"),
		Entries:  createEntries,
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: createPlan}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("create diagnostics: %#v", createResp.Diagnostics)
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: createResp.State}, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
	}
	var readState I18nDictionaryModel
	if diags := readResp.State.Get(context.Background(), &readState); diags.HasError() {
		t.Fatalf("get read state: %#v", diags)
	}
	readEntries := map[string]string{}
	if diags := readState.Entries.ElementsAs(context.Background(), &readEntries, false); diags.HasError() {
		t.Fatalf("get read entries: %#v", diags)
	}
	if want := map[string]string{"login.title": "Connexion"}; !reflect.DeepEqual(readEntries, want) {
		t.Fatalf("read entries = %#v, want %#v", readEntries, want)
	}

	updateEntries := mapValue(t, map[string]string{"login.title": "Bienvenue"})
	updatePlan := i18nDictionaryPlan(t, schemaResp.Schema, I18nDictionaryModel{
		DomainID: types.StringValue("domain-123"),
		Name:     types.StringValue("French updated"),
		Locale:   types.StringValue("fr"),
		Entries:  updateEntries,
	})
	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  updatePlan,
		State: readResp.State,
	}, updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("update diagnostics: %#v", updateResp.Diagnostics)
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: updateResp.State}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}

	wantDictionaryBodies := []map[string]interface{}{
		{"name": "French", "locale": "fr"},
		{"name": "French updated", "locale": "fr"},
	}
	if !reflect.DeepEqual(dictionaryBodies, wantDictionaryBodies) {
		t.Fatalf("dictionary bodies = %#v, want %#v", dictionaryBodies, wantDictionaryBodies)
	}
	wantEntriesBodies := []map[string]string{
		{"login.title": "Connexion"},
		{"login.title": "Bienvenue"},
	}
	if !reflect.DeepEqual(entriesBodies, wantEntriesBodies) {
		t.Fatalf("entries bodies = %#v, want %#v", entriesBodies, wantEntriesBodies)
	}
}

func TestI18nDictionaryReadRemovesMissingDictionaryAndReportsErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantRemove bool
	}{
		{name: "missing dictionary", statusCode: http.StatusNotFound, wantRemove: true},
		{name: "server error", statusCode: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/i18n/dictionaries/dict-123", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				http.Error(w, "read failed", tt.statusCode)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &I18nDictionaryResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			state := i18nDictionaryState(t, schemaResp.Schema, I18nDictionaryModel{
				ID:       types.StringValue("dict-123"),
				DomainID: types.StringValue("domain-123"),
				Name:     types.StringValue("French"),
				Locale:   types.StringValue("fr"),
				Entries:  mapValue(t, map[string]string{"login.title": "Connexion"}),
			})

			readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
			if tt.wantRemove {
				if readResp.Diagnostics.HasError() {
					t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
				}
				if !readResp.State.Raw.IsNull() {
					t.Fatalf("expected missing dictionary to remove state, got %#v", readResp.State.Raw)
				}
				return
			}
			if !readResp.Diagnostics.HasError() {
				t.Fatal("expected read diagnostics")
			}
		})
	}
}

func TestI18nDictionaryReportsLifecycleErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/i18n/dictionaries", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		http.Error(w, "create failed", http.StatusInternalServerError)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/i18n/dictionaries/dict-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			http.Error(w, "update failed", http.StatusInternalServerError)
		case http.MethodDelete:
			http.Error(w, "delete failed", http.StatusInternalServerError)
		default:
			t.Fatalf("method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &I18nDictionaryResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := i18nDictionaryPlan(t, schemaResp.Schema, I18nDictionaryModel{
		DomainID: types.StringValue("domain-123"),
		Name:     types.StringValue("French"),
		Locale:   types.StringValue("fr"),
		Entries:  nullEntries(),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	state := i18nDictionaryState(t, schemaResp.Schema, I18nDictionaryModel{
		ID:       types.StringValue("dict-123"),
		DomainID: types.StringValue("domain-123"),
		Name:     types.StringValue("French"),
		Locale:   types.StringValue("fr"),
		Entries:  nullEntries(),
	})
	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("expected update diagnostics")
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("expected delete diagnostics")
	}
}

func TestI18nDictionaryReportsCreateEntriesError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/i18n/dictionaries", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "dict-123", "name": "French", "locale": "fr"})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/i18n/dictionaries/dict-123/entries", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s, want PUT", r.Method)
		}
		http.Error(w, "entries failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &I18nDictionaryResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := i18nDictionaryPlan(t, schemaResp.Schema, I18nDictionaryModel{
		DomainID: types.StringValue("domain-123"),
		Name:     types.StringValue("French"),
		Locale:   types.StringValue("fr"),
		Entries:  mapValue(t, map[string]string{"login.title": "Connexion"}),
	})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected create entries diagnostics")
	}
}

func TestI18nDictionaryReportsUpdateEntriesError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/i18n/dictionaries/dict-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s, want PUT", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "dict-123", "name": "French", "locale": "fr"})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/i18n/dictionaries/dict-123/entries", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s, want PUT", r.Method)
		}
		http.Error(w, "entries failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &I18nDictionaryResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := i18nDictionaryPlan(t, schemaResp.Schema, I18nDictionaryModel{
		DomainID: types.StringValue("domain-123"),
		Name:     types.StringValue("French"),
		Locale:   types.StringValue("fr"),
		Entries:  mapValue(t, map[string]string{"login.title": "Connexion"}),
	})
	state := i18nDictionaryState(t, schemaResp.Schema, I18nDictionaryModel{
		ID:       types.StringValue("dict-123"),
		DomainID: types.StringValue("domain-123"),
		Name:     types.StringValue("French"),
		Locale:   types.StringValue("fr"),
		Entries:  nullEntries(),
	})

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected update entries diagnostics")
	}
}

func TestI18nDictionaryCreateReportsInvalidEntriesData(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/i18n/dictionaries", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "dict-123", "name": "French", "locale": "fr"})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &I18nDictionaryResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := i18nDictionaryPlan(t, schemaResp.Schema, I18nDictionaryModel{
		DomainID: types.StringValue("domain-123"),
		Name:     types.StringValue("French"),
		Locale:   types.StringValue("fr"),
		Entries: types.MapValueMust(types.StringType, map[string]attr.Value{
			"login.title": types.StringUnknown(),
		}),
	})

	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid entries diagnostics")
	}
}

func TestI18nDictionaryUpdateReportsInvalidEntriesData(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/i18n/dictionaries/dict-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s, want PUT", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "dict-123", "name": "French", "locale": "fr"})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &I18nDictionaryResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := i18nDictionaryPlan(t, schemaResp.Schema, I18nDictionaryModel{
		DomainID: types.StringValue("domain-123"),
		Name:     types.StringValue("French"),
		Locale:   types.StringValue("fr"),
		Entries: types.MapValueMust(types.StringType, map[string]attr.Value{
			"login.title": types.StringUnknown(),
		}),
	})
	state := i18nDictionaryState(t, schemaResp.Schema, I18nDictionaryModel{
		ID:       types.StringValue("dict-123"),
		DomainID: types.StringValue("domain-123"),
		Name:     types.StringValue("French"),
		Locale:   types.StringValue("fr"),
		Entries:  nullEntries(),
	})

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid entries diagnostics")
	}
}

func TestI18nDictionaryImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&I18nDictionaryResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "missing-separator",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func TestI18nDictionaryConfigureAcceptsClient(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &I18nDictionaryResource{}
	var resp resource.ConfigureResponse

	resourceUnderTest.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: client.New("http://example.test", "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("configure diagnostics: %#v", resp.Diagnostics)
	}
	if resourceUnderTest.client == nil {
		t.Fatal("expected client to be configured")
	}
}

func TestI18nDictionaryImportStateSetsDomainAndID(t *testing.T) {
	var schemaResp resource.SchemaResponse
	NewI18nDictionaryResource().Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{State: i18nDictionaryState(t, schemaResp.Schema, I18nDictionaryModel{
		ID:       types.StringValue("placeholder"),
		DomainID: types.StringValue("placeholder"),
		Name:     types.StringValue("French"),
		Locale:   types.StringValue("fr"),
		Entries:  nullEntries(),
	})}

	(&I18nDictionaryResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain-123/dict-123",
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", resp.Diagnostics)
	}
	var imported I18nDictionaryModel
	if diags := resp.State.Get(context.Background(), &imported); diags.HasError() {
		t.Fatalf("get imported state: %#v", diags)
	}
	if got, want := imported.DomainID.ValueString(), "domain-123"; got != want {
		t.Fatalf("domain id = %q, want %q", got, want)
	}
	if got, want := imported.ID.ValueString(), "dict-123"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
}

func TestI18nDictionaryCreateReadUpdateAndDeleteReportInvalidStateData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &I18nDictionaryResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	entriesType := tftypes.Map{ElementType: tftypes.String}
	raw := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"id":        tftypes.String,
			"domain_id": tftypes.Number,
			"name":      tftypes.String,
			"locale":    tftypes.String,
			"entries":   entriesType,
		}},
		map[string]tftypes.Value{
			"id":        tftypes.NewValue(tftypes.String, "dict-123"),
			"domain_id": tftypes.NewValue(tftypes.Number, 123),
			"name":      tftypes.NewValue(tftypes.String, "messages"),
			"locale":    tftypes.NewValue(tftypes.String, "en"),
			"entries": tftypes.NewValue(entriesType, map[string]tftypes.Value{
				"hello": tftypes.NewValue(tftypes.String, "Hello"),
			}),
		},
	)

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: raw}}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw}}, readResp)
	if !readResp.Diagnostics.HasError() {
		t.Fatal("expected read diagnostics")
	}

	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: raw},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw},
	}, updateResp)
	if !updateResp.Diagnostics.HasError() {
		t.Fatal("expected update diagnostics")
	}

	invalidStateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{
		Plan: i18nDictionaryPlan(t, schemaResp.Schema, I18nDictionaryModel{
			ID:       types.StringValue("dict-123"),
			DomainID: types.StringValue("domain-123"),
			Name:     types.StringValue("messages"),
			Locale:   types.StringValue("en"),
			Entries:  mapValue(t, map[string]string{"hello": "Hello"}),
		}),
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw},
	}, invalidStateResp)
	if !invalidStateResp.Diagnostics.HasError() {
		t.Fatal("expected invalid state diagnostics")
	}

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw}}, deleteResp)
	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("expected delete diagnostics")
	}
}

func i18nDictionaryPlan(t *testing.T, schema resourceschema.Schema, model I18nDictionaryModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func i18nDictionaryState(t *testing.T, schema resourceschema.Schema, model I18nDictionaryModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}

func mapValue(t *testing.T, values map[string]string) types.Map {
	t.Helper()

	value, diags := types.MapValueFrom(context.Background(), types.StringType, values)
	if diags.HasError() {
		t.Fatalf("map value diagnostics: %#v", diags)
	}
	return value
}

func nullEntries() types.Map {
	return types.MapNull(types.StringType)
}

func assertStringAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
	t.Helper()

	attr, ok := attrs[name].(schema.StringAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want schema.StringAttribute", name, attrs[name])
	}
	if attr.Required != required || attr.Optional != optional || attr.Computed != computed {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t, want required:%t optional:%t computed:%t",
			name, attr.Required, attr.Optional, attr.Computed, required, optional, computed)
	}
}

func assertMapAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, elemType attr.Type) {
	t.Helper()

	attr, ok := attrs[name].(schema.MapAttribute)
	if !ok {
		t.Fatalf("%s attribute = %T, want schema.MapAttribute", name, attrs[name])
	}
	if !attr.Optional || attr.Required || attr.Computed {
		t.Fatalf("%s flags = required:%t optional:%t computed:%t, want optional only",
			name, attr.Required, attr.Optional, attr.Computed)
	}
	if !reflect.DeepEqual(attr.ElementType, elemType) {
		t.Fatalf("%s element type = %#v, want %#v", name, attr.ElementType, elemType)
	}
}
