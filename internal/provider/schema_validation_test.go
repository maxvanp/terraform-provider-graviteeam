package provider

import (
	"context"
	"testing"

	fwdatasource "github.com/hashicorp/terraform-plugin-framework/datasource"
	fwprovider "github.com/hashicorp/terraform-plugin-framework/provider"
	fwresource "github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestProviderSchema(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	p := New("test")()

	req := fwprovider.SchemaRequest{}
	resp := &fwprovider.SchemaResponse{}
	p.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("provider schema diagnostics: %+v", resp.Diagnostics)
	}

	diagnostics := resp.Schema.ValidateImplementation(ctx)
	if diagnostics.HasError() {
		t.Fatalf("provider schema validation diagnostics: %+v", diagnostics)
	}
}

func TestResourceSchemas(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	p := New("test")()

	for _, constructor := range p.Resources(ctx) {
		r := constructor()
		req := fwresource.SchemaRequest{}
		resp := &fwresource.SchemaResponse{}
		r.Schema(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Fatalf("%T schema diagnostics: %+v", r, resp.Diagnostics)
		}

		diagnostics := resp.Schema.ValidateImplementation(ctx)
		if diagnostics.HasError() {
			t.Fatalf("%T schema validation diagnostics: %+v", r, diagnostics)
		}
	}
}

func TestDataSourceSchemas(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	p := New("test")()

	for _, constructor := range p.DataSources(ctx) {
		d := constructor()
		req := fwdatasource.SchemaRequest{}
		resp := &fwdatasource.SchemaResponse{}
		d.Schema(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Fatalf("%T schema diagnostics: %+v", d, resp.Diagnostics)
		}

		diagnostics := resp.Schema.ValidateImplementation(ctx)
		if diagnostics.HasError() {
			t.Fatalf("%T schema validation diagnostics: %+v", d, diagnostics)
		}
	}
}
