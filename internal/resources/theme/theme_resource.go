package theme

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

var (
	_ resource.Resource                = &ThemeResource{}
	_ resource.ResourceWithImportState = &ThemeResource{}
)

type ThemeResource struct {
	client *client.Client
}

func NewThemeResource() resource.Resource {
	return &ThemeResource{}
}

func (r *ThemeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_theme"
}

func (r *ThemeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM Theme (branding, colors, custom CSS)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the theme",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain this theme belongs to",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"logo_url": schema.StringAttribute{
				Optional:    true,
				Description: "URL of the logo image",
			},
			"logo_width": schema.Int64Attribute{
				Optional:    true,
				Description: "Width of the logo in pixels",
			},
			"favicon_url": schema.StringAttribute{
				Optional:    true,
				Description: "URL of the favicon",
			},
			"primary_button_color_hex": schema.StringAttribute{
				Optional:    true,
				Description: "Primary button color in hex format (e.g. #4CAF50)",
			},
			"secondary_button_color_hex": schema.StringAttribute{
				Optional:    true,
				Description: "Secondary button color in hex format",
			},
			"primary_text_color_hex": schema.StringAttribute{
				Optional:    true,
				Description: "Primary text color in hex format",
			},
			"secondary_text_color_hex": schema.StringAttribute{
				Optional:    true,
				Description: "Secondary text color in hex format",
			},
			"css": schema.StringAttribute{
				Optional:    true,
				Description: "Custom CSS to apply to login pages",
			},
		},
	}
}

func (r *ThemeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", "Expected *client.Client")
		return
	}
	r.client = c
}

func (r *ThemeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ThemeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.buildBody(plan)

	result, err := r.client.CreateTheme(ctx, plan.DomainID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating theme", err.Error())
		return
	}

	plan.ID = types.StringValue(result["id"].(string))

	r.readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ThemeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ThemeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	themes, err := r.client.GetThemes(ctx, state.DomainID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading theme", err.Error())
		return
	}

	if len(themes) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}

	// Find our theme by ID, or take the first one
	var result map[string]interface{}
	for _, t := range themes {
		if id, ok := t["id"].(string); ok && id == state.ID.ValueString() {
			result = t
			break
		}
	}
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.readIntoModel(&state, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ThemeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ThemeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ThemeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	body := r.buildBody(plan)

	result, err := r.client.UpdateTheme(ctx, plan.DomainID.ValueString(), plan.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating theme", err.Error())
		return
	}

	r.readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ThemeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ThemeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteTheme(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting theme", err.Error())
	}
}

func (r *ThemeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/theme_id, got: %s", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *ThemeResource) buildBody(plan ThemeModel) map[string]interface{} {
	body := map[string]interface{}{}

	if !plan.LogoURL.IsNull() && !plan.LogoURL.IsUnknown() {
		body["logoUrl"] = plan.LogoURL.ValueString()
	}
	if !plan.LogoWidth.IsNull() && !plan.LogoWidth.IsUnknown() {
		body["logoWidth"] = plan.LogoWidth.ValueInt64()
	}
	if !plan.FaviconURL.IsNull() && !plan.FaviconURL.IsUnknown() {
		body["faviconUrl"] = plan.FaviconURL.ValueString()
	}
	if !plan.PrimaryButtonColorHex.IsNull() && !plan.PrimaryButtonColorHex.IsUnknown() {
		body["primaryButtonColorHex"] = plan.PrimaryButtonColorHex.ValueString()
	}
	if !plan.SecondaryButtonColorHex.IsNull() && !plan.SecondaryButtonColorHex.IsUnknown() {
		body["secondaryButtonColorHex"] = plan.SecondaryButtonColorHex.ValueString()
	}
	if !plan.PrimaryTextColorHex.IsNull() && !plan.PrimaryTextColorHex.IsUnknown() {
		body["primaryTextColorHex"] = plan.PrimaryTextColorHex.ValueString()
	}
	if !plan.SecondaryTextColorHex.IsNull() && !plan.SecondaryTextColorHex.IsUnknown() {
		body["secondaryTextColorHex"] = plan.SecondaryTextColorHex.ValueString()
	}
	if !plan.CSS.IsNull() && !plan.CSS.IsUnknown() {
		body["css"] = plan.CSS.ValueString()
	}

	return body
}

func (r *ThemeResource) readIntoModel(model *ThemeModel, data map[string]interface{}) {
	if id, ok := data["id"].(string); ok {
		model.ID = types.StringValue(id)
	}
	if v, ok := data["logoUrl"].(string); ok && v != "" {
		model.LogoURL = types.StringValue(v)
	} else if model.LogoURL.IsNull() {
		model.LogoURL = types.StringNull()
	}
	if v, ok := data["logoWidth"]; ok {
		switch n := v.(type) {
		case float64:
			if n > 0 {
				model.LogoWidth = types.Int64Value(int64(n))
			}
		case int64:
			if n > 0 {
				model.LogoWidth = types.Int64Value(n)
			}
		}
	} else if model.LogoWidth.IsNull() {
		model.LogoWidth = types.Int64Null()
	}
	if v, ok := data["faviconUrl"].(string); ok && v != "" {
		model.FaviconURL = types.StringValue(v)
	} else if model.FaviconURL.IsNull() {
		model.FaviconURL = types.StringNull()
	}
	if v, ok := data["primaryButtonColorHex"].(string); ok && v != "" {
		model.PrimaryButtonColorHex = types.StringValue(v)
	} else if model.PrimaryButtonColorHex.IsNull() {
		model.PrimaryButtonColorHex = types.StringNull()
	}
	if v, ok := data["secondaryButtonColorHex"].(string); ok && v != "" {
		model.SecondaryButtonColorHex = types.StringValue(v)
	} else if model.SecondaryButtonColorHex.IsNull() {
		model.SecondaryButtonColorHex = types.StringNull()
	}
	if v, ok := data["primaryTextColorHex"].(string); ok && v != "" {
		model.PrimaryTextColorHex = types.StringValue(v)
	} else if model.PrimaryTextColorHex.IsNull() {
		model.PrimaryTextColorHex = types.StringNull()
	}
	if v, ok := data["secondaryTextColorHex"].(string); ok && v != "" {
		model.SecondaryTextColorHex = types.StringValue(v)
	} else if model.SecondaryTextColorHex.IsNull() {
		model.SecondaryTextColorHex = types.StringNull()
	}
	if v, ok := data["css"].(string); ok && v != "" {
		model.CSS = types.StringValue(v)
	} else if model.CSS.IsNull() {
		model.CSS = types.StringNull()
	}
}
