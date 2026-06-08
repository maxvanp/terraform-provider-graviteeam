package theme

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

func TestMetadata(t *testing.T) {
	t.Parallel()

	var resp resource.MetadataResponse
	NewThemeResource().Metadata(context.Background(), resource.MetadataRequest{
		ProviderTypeName: "graviteeam",
	}, &resp)

	if got, want := resp.TypeName, "graviteeam_theme"; got != want {
		t.Fatalf("type name = %q, want %q", got, want)
	}
}

func TestSchemaAttributes(t *testing.T) {
	t.Parallel()

	var resp resource.SchemaResponse
	NewThemeResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)

	if attr := resp.Schema.Attributes["domain_id"]; attr == nil || !attr.IsRequired() {
		t.Fatalf("domain_id should be required")
	}
	for _, name := range []string{
		"logo_url",
		"logo_width",
		"favicon_url",
		"primary_button_color_hex",
		"secondary_button_color_hex",
		"primary_text_color_hex",
		"secondary_text_color_hex",
		"css",
	} {
		attr, ok := resp.Schema.Attributes[name]
		if !ok {
			t.Fatalf("missing schema attribute %q", name)
		}
		if !attr.IsOptional() {
			t.Fatalf("attribute %q should be optional", name)
		}
	}
	if attr := resp.Schema.Attributes["id"]; attr == nil || !attr.IsComputed() {
		t.Fatalf("id should be computed")
	}
}

func TestConfigureRejectsUnexpectedProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&ThemeResource{}).Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: "not-a-client",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected configure diagnostic")
	}
}

func TestConfigureAllowsNilProviderData(t *testing.T) {
	t.Parallel()

	var resp resource.ConfigureResponse
	(&ThemeResource{}).Configure(context.Background(), resource.ConfigureRequest{}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected configure diagnostics: %#v", resp.Diagnostics)
	}
}

func TestBuildBody(t *testing.T) {
	t.Parallel()

	resource := &ThemeResource{}
	plan := ThemeModel{
		LogoURL:                 types.StringValue("https://example.com/logo.png"),
		LogoWidth:               types.Int64Value(180),
		FaviconURL:              types.StringValue("https://example.com/favicon.ico"),
		PrimaryButtonColorHex:   types.StringValue("#111111"),
		SecondaryButtonColorHex: types.StringValue("#222222"),
		PrimaryTextColorHex:     types.StringValue("#333333"),
		SecondaryTextColorHex:   types.StringValue("#444444"),
		CSS:                     types.StringValue("body { color: #333; }"),
	}

	got := resource.buildBody(plan)
	want := map[string]interface{}{
		"logoUrl":                 "https://example.com/logo.png",
		"logoWidth":               int64(180),
		"faviconUrl":              "https://example.com/favicon.ico",
		"primaryButtonColorHex":   "#111111",
		"secondaryButtonColorHex": "#222222",
		"primaryTextColorHex":     "#333333",
		"secondaryTextColorHex":   "#444444",
		"css":                     "body { color: #333; }",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildBodySkipsUnknownOptionalFields(t *testing.T) {
	t.Parallel()

	resource := &ThemeResource{}
	plan := ThemeModel{
		LogoURL:                 types.StringUnknown(),
		LogoWidth:               types.Int64Unknown(),
		FaviconURL:              types.StringUnknown(),
		PrimaryButtonColorHex:   types.StringUnknown(),
		SecondaryButtonColorHex: types.StringUnknown(),
		PrimaryTextColorHex:     types.StringUnknown(),
		SecondaryTextColorHex:   types.StringUnknown(),
		CSS:                     types.StringUnknown(),
	}

	if got := resource.buildBody(plan); len(got) != 0 {
		t.Fatalf("body = %#v, want no fields for unknown plan values", got)
	}
}

func TestBuildBodySkipsNullOptionalFields(t *testing.T) {
	t.Parallel()

	resource := &ThemeResource{}
	plan := ThemeModel{
		LogoURL:                 types.StringNull(),
		LogoWidth:               types.Int64Null(),
		FaviconURL:              types.StringNull(),
		PrimaryButtonColorHex:   types.StringNull(),
		SecondaryButtonColorHex: types.StringNull(),
		PrimaryTextColorHex:     types.StringNull(),
		SecondaryTextColorHex:   types.StringNull(),
		CSS:                     types.StringNull(),
	}

	if got := resource.buildBody(plan); len(got) != 0 {
		t.Fatalf("body = %#v, want no fields for null plan values", got)
	}
}

func TestBuildUpdateBodyMergesCurrentAndClearsRemovedFields(t *testing.T) {
	t.Parallel()

	resource := &ThemeResource{}
	current := map[string]interface{}{
		"id":                      "theme-1",
		"logoUrl":                 "https://example.com/old-logo.png",
		"logoWidth":               float64(120),
		"faviconUrl":              "https://example.com/old.ico",
		"primaryButtonColorHex":   "#000000",
		"secondaryButtonColorHex": "#010101",
		"primaryTextColorHex":     "#020202",
		"secondaryTextColorHex":   "#030303",
		"css":                     ".old {}",
		"apiManaged":              "preserve-me",
	}
	state := ThemeModel{
		LogoURL:                 types.StringValue("https://example.com/old-logo.png"),
		LogoWidth:               types.Int64Value(120),
		FaviconURL:              types.StringValue("https://example.com/old.ico"),
		PrimaryButtonColorHex:   types.StringValue("#000000"),
		SecondaryButtonColorHex: types.StringValue("#010101"),
		PrimaryTextColorHex:     types.StringValue("#020202"),
		SecondaryTextColorHex:   types.StringValue("#030303"),
		CSS:                     types.StringValue(".old {}"),
	}
	plan := ThemeModel{
		LogoURL:                 types.StringValue("https://example.com/new-logo.png"),
		LogoWidth:               types.Int64Null(),
		FaviconURL:              types.StringNull(),
		PrimaryButtonColorHex:   types.StringValue("#111111"),
		SecondaryButtonColorHex: types.StringNull(),
		PrimaryTextColorHex:     types.StringValue("#222222"),
		SecondaryTextColorHex:   types.StringNull(),
		CSS:                     types.StringValue(".new {}"),
	}

	got := resource.buildUpdateBody(plan, state, current)
	want := map[string]interface{}{
		"id":                      "theme-1",
		"logoUrl":                 "https://example.com/new-logo.png",
		"logoWidth":               0,
		"faviconUrl":              "",
		"primaryButtonColorHex":   "#111111",
		"secondaryButtonColorHex": "",
		"primaryTextColorHex":     "#222222",
		"secondaryTextColorHex":   "",
		"css":                     ".new {}",
		"apiManaged":              "preserve-me",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestBuildUpdateBodyClearsRemainingRemovedFields(t *testing.T) {
	t.Parallel()

	resource := &ThemeResource{}
	state := ThemeModel{
		LogoURL:               types.StringValue("https://example.com/old-logo.png"),
		PrimaryButtonColorHex: types.StringValue("#000000"),
		PrimaryTextColorHex:   types.StringValue("#020202"),
		CSS:                   types.StringValue(".old {}"),
	}
	plan := ThemeModel{
		LogoURL:               types.StringNull(),
		PrimaryButtonColorHex: types.StringNull(),
		PrimaryTextColorHex:   types.StringNull(),
		CSS:                   types.StringNull(),
	}

	got := resource.buildUpdateBody(plan, state, map[string]interface{}{"apiManaged": "preserve-me"})
	want := map[string]interface{}{
		"logoUrl":               "",
		"primaryButtonColorHex": "",
		"primaryTextColorHex":   "",
		"css":                   "",
		"apiManaged":            "preserve-me",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestReadIntoModel(t *testing.T) {
	t.Parallel()

	resource := &ThemeResource{}
	model := ThemeModel{}

	resource.readIntoModel(&model, map[string]interface{}{
		"id":                      "theme-1",
		"logoUrl":                 "https://example.com/logo.png",
		"logoWidth":               float64(180),
		"faviconUrl":              "https://example.com/favicon.ico",
		"primaryButtonColorHex":   "#111111",
		"secondaryButtonColorHex": "#222222",
		"primaryTextColorHex":     "#333333",
		"secondaryTextColorHex":   "#444444",
		"css":                     ".theme {}",
	})

	if got, want := model.ID.ValueString(), "theme-1"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
	if got, want := model.LogoURL.ValueString(), "https://example.com/logo.png"; got != want {
		t.Fatalf("logo URL = %q, want %q", got, want)
	}
	if got, want := model.LogoWidth.ValueInt64(), int64(180); got != want {
		t.Fatalf("logo width = %d, want %d", got, want)
	}
	if got, want := model.FaviconURL.ValueString(), "https://example.com/favicon.ico"; got != want {
		t.Fatalf("favicon URL = %q, want %q", got, want)
	}
	if got, want := model.PrimaryButtonColorHex.ValueString(), "#111111"; got != want {
		t.Fatalf("primary button color = %q, want %q", got, want)
	}
	if got, want := model.SecondaryButtonColorHex.ValueString(), "#222222"; got != want {
		t.Fatalf("secondary button color = %q, want %q", got, want)
	}
	if got, want := model.PrimaryTextColorHex.ValueString(), "#333333"; got != want {
		t.Fatalf("primary text color = %q, want %q", got, want)
	}
	if got, want := model.SecondaryTextColorHex.ValueString(), "#444444"; got != want {
		t.Fatalf("secondary text color = %q, want %q", got, want)
	}
	if got, want := model.CSS.ValueString(), ".theme {}"; got != want {
		t.Fatalf("css = %q, want %q", got, want)
	}
}

func TestReadIntoModelClearsEmptyValues(t *testing.T) {
	t.Parallel()

	resource := &ThemeResource{}
	model := ThemeModel{
		LogoURL:               types.StringValue("old"),
		LogoWidth:             types.Int64Value(10),
		FaviconURL:            types.StringValue("old"),
		PrimaryButtonColorHex: types.StringValue("old"),
		CSS:                   types.StringValue("old"),
	}

	resource.readIntoModel(&model, map[string]interface{}{
		"logoUrl":               "",
		"logoWidth":             float64(0),
		"faviconUrl":            "",
		"primaryButtonColorHex": "",
		"css":                   "",
	})

	if !model.LogoURL.IsNull() {
		t.Fatalf("logo URL should be null, got %q", model.LogoURL.ValueString())
	}
	if !model.LogoWidth.IsNull() {
		t.Fatalf("logo width should be null, got %d", model.LogoWidth.ValueInt64())
	}
	if !model.FaviconURL.IsNull() {
		t.Fatalf("favicon URL should be null, got %q", model.FaviconURL.ValueString())
	}
	if !model.PrimaryButtonColorHex.IsNull() {
		t.Fatalf("primary button color should be null, got %q", model.PrimaryButtonColorHex.ValueString())
	}
	if !model.CSS.IsNull() {
		t.Fatalf("css should be null, got %q", model.CSS.ValueString())
	}
}

func TestThemeCRUDMergesCurrentThemeOnUpdate(t *testing.T) {
	var bodies []map[string]interface{}
	var methods []string

	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/themes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("collection method = %s, want POST", r.Method)
		}
		methods = append(methods, "create")
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode create body: %v", err)
		}
		bodies = append(bodies, body)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id":                    "theme-123",
			"logoUrl":               body["logoUrl"],
			"logoWidth":             body["logoWidth"],
			"primaryButtonColorHex": body["primaryButtonColorHex"],
			"css":                   body["css"],
		})
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/themes/theme-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			methods = append(methods, "read")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":                    "theme-123",
				"logoUrl":               "https://example.test/logo.png",
				"logoWidth":             float64(160),
				"faviconUrl":            "https://example.test/favicon.ico",
				"primaryButtonColorHex": "#111111",
				"css":                   ".old {}",
				"apiManaged":            "preserve-me",
			})
		case http.MethodPut:
			methods = append(methods, "update")
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode update body: %v", err)
			}
			bodies = append(bodies, body)
			_ = json.NewEncoder(w).Encode(body)
		case http.MethodDelete:
			methods = append(methods, "delete")
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("item method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ThemeResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	createPlan := themePlan(t, schemaResp.Schema, ThemeModel{
		DomainID:              types.StringValue("domain-123"),
		LogoURL:               types.StringValue("https://example.test/logo.png"),
		LogoWidth:             types.Int64Value(160),
		PrimaryButtonColorHex: types.StringValue("#111111"),
		CSS:                   types.StringValue(".old {}"),
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

	updatePlan := themePlan(t, schemaResp.Schema, ThemeModel{
		DomainID:              types.StringValue("domain-123"),
		LogoURL:               types.StringValue("https://example.test/new-logo.png"),
		LogoWidth:             types.Int64Null(),
		FaviconURL:            types.StringNull(),
		PrimaryButtonColorHex: types.StringValue("#222222"),
		CSS:                   types.StringValue(".new {}"),
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

	if !reflect.DeepEqual(methods, []string{"create", "read", "read", "update", "delete"}) {
		t.Fatalf("methods = %#v", methods)
	}
	if got := bodies[1]["apiManaged"]; got != "preserve-me" {
		t.Fatalf("apiManaged = %#v, want preserved", got)
	}
	if got := bodies[1]["logoUrl"]; got != "https://example.test/new-logo.png" {
		t.Fatalf("logoUrl = %#v", got)
	}
	if got := bodies[1]["logoWidth"]; got != float64(0) {
		t.Fatalf("logoWidth = %#v, want clear zero", got)
	}
	if got := bodies[1]["faviconUrl"]; got != "" {
		t.Fatalf("faviconUrl = %#v, want clear string", got)
	}
}

func TestThemeReadRemovesMissingThemeAndReportsErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantRemove bool
	}{
		{
			name:       "missing theme",
			statusCode: http.StatusNotFound,
			wantRemove: true,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			wantRemove: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
			})
			mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/themes/theme-123", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				http.Error(w, "read failed", tt.statusCode)
			})
			server := httptest.NewServer(mux)
			defer server.Close()

			resourceUnderTest := &ThemeResource{
				client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
			}
			var schemaResp resource.SchemaResponse
			resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
			state := themeState(t, schemaResp.Schema, ThemeModel{
				ID:       types.StringValue("theme-123"),
				DomainID: types.StringValue("domain-123"),
				LogoURL:  types.StringValue("https://example.test/logo.png"),
			})

			readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			resourceUnderTest.Read(context.Background(), resource.ReadRequest{State: state}, readResp)
			if tt.wantRemove {
				if readResp.Diagnostics.HasError() {
					t.Fatalf("read diagnostics: %#v", readResp.Diagnostics)
				}
				if !readResp.State.Raw.IsNull() {
					t.Fatalf("expected missing theme to remove state, got %#v", readResp.State.Raw)
				}
				return
			}
			if !readResp.Diagnostics.HasError() {
				t.Fatal("expected read diagnostics")
			}
		})
	}
}

func TestThemeReportsLifecycleErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/themes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("collection method = %s, want POST", r.Method)
		}
		http.Error(w, "create failed", http.StatusInternalServerError)
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/themes/theme-123", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":      "theme-123",
				"logoUrl": "https://example.test/logo.png",
			})
		case http.MethodPut:
			http.Error(w, "update failed", http.StatusInternalServerError)
		case http.MethodDelete:
			http.Error(w, "delete failed", http.StatusInternalServerError)
		default:
			t.Fatalf("item method = %s", r.Method)
		}
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ThemeResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := themePlan(t, schemaResp.Schema, ThemeModel{
		DomainID: types.StringValue("domain-123"),
		LogoURL:  types.StringValue("https://example.test/logo.png"),
	})

	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Create(context.Background(), resource.CreateRequest{Plan: plan}, createResp)
	if !createResp.Diagnostics.HasError() {
		t.Fatal("expected create diagnostics")
	}

	state := themeState(t, schemaResp.Schema, ThemeModel{
		ID:       types.StringValue("theme-123"),
		DomainID: types.StringValue("domain-123"),
		LogoURL:  types.StringValue("https://example.test/logo.png"),
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

func TestThemeDeleteReportsInvalidStateData(t *testing.T) {
	t.Parallel()

	resourceUnderTest := &ThemeResource{}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	raw := tftypes.NewValue(
		tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"id":                         tftypes.String,
			"domain_id":                  tftypes.Number,
			"logo_url":                   tftypes.String,
			"logo_width":                 tftypes.Number,
			"favicon_url":                tftypes.String,
			"primary_button_color_hex":   tftypes.String,
			"secondary_button_color_hex": tftypes.String,
			"primary_text_color_hex":     tftypes.String,
			"secondary_text_color_hex":   tftypes.String,
			"css":                        tftypes.String,
		}},
		map[string]tftypes.Value{
			"id":                         tftypes.NewValue(tftypes.String, "theme-123"),
			"domain_id":                  tftypes.NewValue(tftypes.Number, 123),
			"logo_url":                   tftypes.NewValue(tftypes.String, "https://example.test/logo.png"),
			"logo_width":                 tftypes.NewValue(tftypes.Number, nil),
			"favicon_url":                tftypes.NewValue(tftypes.String, nil),
			"primary_button_color_hex":   tftypes.NewValue(tftypes.String, nil),
			"secondary_button_color_hex": tftypes.NewValue(tftypes.String, nil),
			"primary_text_color_hex":     tftypes.NewValue(tftypes.String, nil),
			"secondary_text_color_hex":   tftypes.NewValue(tftypes.String, nil),
			"css":                        tftypes.NewValue(tftypes.String, nil),
		},
	)

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: raw},
	}, deleteResp)
	if !deleteResp.Diagnostics.HasError() {
		t.Fatal("expected invalid state diagnostics")
	}
}

func TestThemeDeleteIgnores404(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/themes/theme-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s, want DELETE", r.Method)
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ThemeResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	state := themeState(t, schemaResp.Schema, ThemeModel{
		ID:       types.StringValue("theme-123"),
		DomainID: types.StringValue("domain-123"),
		LogoURL:  types.StringValue("https://example.test/logo.png"),
	})

	deleteResp := &resource.DeleteResponse{}
	resourceUnderTest.Delete(context.Background(), resource.DeleteRequest{State: state}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("delete diagnostics: %#v", deleteResp.Diagnostics)
	}
}

func TestThemeUpdateReportsPreReadError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/management/auth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"bearer"}`))
	})
	mux.HandleFunc("/management/organizations/DEFAULT/environments/DEFAULT/domains/domain-123/themes/theme-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		http.Error(w, "read before update failed", http.StatusInternalServerError)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	resourceUnderTest := &ThemeResource{
		client: client.New(server.URL, "admin", "adminadmin", "DEFAULT", "DEFAULT"),
	}
	var schemaResp resource.SchemaResponse
	resourceUnderTest.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	plan := themePlan(t, schemaResp.Schema, ThemeModel{
		DomainID: types.StringValue("domain-123"),
		LogoURL:  types.StringValue("https://example.test/logo.png"),
	})
	state := themeState(t, schemaResp.Schema, ThemeModel{
		ID:       types.StringValue("theme-123"),
		DomainID: types.StringValue("domain-123"),
		LogoURL:  types.StringValue("https://example.test/logo.png"),
	})

	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	resourceUnderTest.Update(context.Background(), resource.UpdateRequest{Plan: plan, State: state}, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected update pre-read diagnostics")
	}
}

func TestThemeImportRejectsInvalidID(t *testing.T) {
	var resp resource.ImportStateResponse
	(&ThemeResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "missing-separator",
	}, &resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected invalid import id diagnostics")
	}
}

func TestThemeImportStateSetsAttributes(t *testing.T) {
	var schemaResp resource.SchemaResponse
	(&ThemeResource{}).Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	resp := resource.ImportStateResponse{State: themeState(t, schemaResp.Schema, ThemeModel{
		ID:       types.StringValue("old-theme"),
		DomainID: types.StringValue("old-domain"),
		LogoURL:  types.StringValue("https://example.test/logo.png"),
	})}

	(&ThemeResource{}).ImportState(context.Background(), resource.ImportStateRequest{
		ID: "domain-123/theme-123",
	}, &resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("import diagnostics: %#v", resp.Diagnostics)
	}
	var state ThemeModel
	if diags := resp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("get import state: %#v", diags)
	}
	if got, want := state.DomainID.ValueString(), "domain-123"; got != want {
		t.Fatalf("domain_id = %q, want %q", got, want)
	}
	if got, want := state.ID.ValueString(), "theme-123"; got != want {
		t.Fatalf("id = %q, want %q", got, want)
	}
}

func themePlan(t *testing.T, schema resourceschema.Schema, model ThemeModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{Schema: schema}
	if diags := plan.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set plan: %#v", diags)
	}
	return plan
}

func themeState(t *testing.T, schema resourceschema.Schema, model ThemeModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{Schema: schema}
	if diags := state.Set(context.Background(), &model); diags.HasError() {
		t.Fatalf("set state: %#v", diags)
	}
	return state
}
