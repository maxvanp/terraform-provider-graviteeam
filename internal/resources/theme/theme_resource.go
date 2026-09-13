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
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/importid"
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

	id, err := client.RequiredString(result, "id")
	if err != nil {
		resp.Diagnostics.AddError("Invalid create response", err.Error())
		return
	}
	plan.ID = types.StringValue(id)

	r.readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ThemeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ThemeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetTheme(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading theme", err.Error())
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

	current, err := r.client.GetTheme(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading current theme before update", err.Error())
		return
	}

	body := r.buildUpdateBody(plan, state, current)

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
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting theme", err.Error())
	}
}

func (r *ThemeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || !importid.Valid(parts) {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/theme_id, got: %s", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *ThemeResource) buildBody(plan ThemeModel) map[string]interface{} {
	body := map[string]interface{}{}
	applyThemePlan(body, plan, nil)
	return body
}

func (r *ThemeResource) buildUpdateBody(plan, state ThemeModel, current map[string]interface{}) map[string]interface{} {
	body := make(map[string]interface{}, len(current))
	for k, v := range current {
		body[k] = v
	}
	applyThemePlan(body, plan, &state)
	return body
}

func applyThemePlan(body map[string]interface{}, plan ThemeModel, state *ThemeModel) {
	if !plan.LogoURL.IsNull() && !plan.LogoURL.IsUnknown() {
		body["logoUrl"] = plan.LogoURL.ValueString()
	} else if state != nil && !state.LogoURL.IsNull() {
		body["logoUrl"] = ""
	}
	if !plan.LogoWidth.IsNull() && !plan.LogoWidth.IsUnknown() {
		body["logoWidth"] = plan.LogoWidth.ValueInt64()
	} else if state != nil && !state.LogoWidth.IsNull() {
		body["logoWidth"] = 0
	}
	if !plan.FaviconURL.IsNull() && !plan.FaviconURL.IsUnknown() {
		body["faviconUrl"] = plan.FaviconURL.ValueString()
	} else if state != nil && !state.FaviconURL.IsNull() {
		body["faviconUrl"] = ""
	}
	if !plan.PrimaryButtonColorHex.IsNull() && !plan.PrimaryButtonColorHex.IsUnknown() {
		body["primaryButtonColorHex"] = plan.PrimaryButtonColorHex.ValueString()
	} else if state != nil && !state.PrimaryButtonColorHex.IsNull() {
		body["primaryButtonColorHex"] = ""
	}
	if !plan.SecondaryButtonColorHex.IsNull() && !plan.SecondaryButtonColorHex.IsUnknown() {
		body["secondaryButtonColorHex"] = plan.SecondaryButtonColorHex.ValueString()
	} else if state != nil && !state.SecondaryButtonColorHex.IsNull() {
		body["secondaryButtonColorHex"] = ""
	}
	if !plan.PrimaryTextColorHex.IsNull() && !plan.PrimaryTextColorHex.IsUnknown() {
		body["primaryTextColorHex"] = plan.PrimaryTextColorHex.ValueString()
	} else if state != nil && !state.PrimaryTextColorHex.IsNull() {
		body["primaryTextColorHex"] = ""
	}
	if !plan.SecondaryTextColorHex.IsNull() && !plan.SecondaryTextColorHex.IsUnknown() {
		body["secondaryTextColorHex"] = plan.SecondaryTextColorHex.ValueString()
	} else if state != nil && !state.SecondaryTextColorHex.IsNull() {
		body["secondaryTextColorHex"] = ""
	}
	if !plan.CSS.IsNull() && !plan.CSS.IsUnknown() {
		body["css"] = plan.CSS.ValueString()
	} else if state != nil && !state.CSS.IsNull() {
		body["css"] = ""
	}
}

func (r *ThemeResource) readIntoModel(model *ThemeModel, data map[string]interface{}) {
	if id, ok := data["id"].(string); ok {
		model.ID = types.StringValue(id)
	}
	if v, ok := data["logoUrl"].(string); ok && v != "" {
		model.LogoURL = types.StringValue(v)
	} else {
		model.LogoURL = types.StringNull()
	}
	if v, ok := data["logoWidth"]; ok {
		model.LogoWidth = types.Int64Null()
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
	} else {
		model.LogoWidth = types.Int64Null()
	}
	if v, ok := data["faviconUrl"].(string); ok && v != "" {
		model.FaviconURL = types.StringValue(v)
	} else {
		model.FaviconURL = types.StringNull()
	}
	if v, ok := data["primaryButtonColorHex"].(string); ok && v != "" {
		model.PrimaryButtonColorHex = types.StringValue(v)
	} else {
		model.PrimaryButtonColorHex = types.StringNull()
	}
	if v, ok := data["secondaryButtonColorHex"].(string); ok && v != "" {
		model.SecondaryButtonColorHex = types.StringValue(v)
	} else {
		model.SecondaryButtonColorHex = types.StringNull()
	}
	if v, ok := data["primaryTextColorHex"].(string); ok && v != "" {
		model.PrimaryTextColorHex = types.StringValue(v)
	} else {
		model.PrimaryTextColorHex = types.StringNull()
	}
	if v, ok := data["secondaryTextColorHex"].(string); ok && v != "" {
		model.SecondaryTextColorHex = types.StringValue(v)
	} else {
		model.SecondaryTextColorHex = types.StringNull()
	}
	if v, ok := data["css"].(string); ok && v != "" {
		model.CSS = types.StringValue(v)
	} else {
		model.CSS = types.StringNull()
	}
}
