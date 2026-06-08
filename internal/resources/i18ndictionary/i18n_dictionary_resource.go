package i18ndictionary

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

var (
	_ resource.Resource                = &I18nDictionaryResource{}
	_ resource.ResourceWithImportState = &I18nDictionaryResource{}
)

type I18nDictionaryResource struct {
	client *client.Client
}

func NewI18nDictionaryResource() resource.Resource {
	return &I18nDictionaryResource{}
}

func (r *I18nDictionaryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_i18n_dictionary"
}

func (r *I18nDictionaryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM I18n Dictionary for login page translations",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the dictionary",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the dictionary",
			},
			"locale": schema.StringAttribute{
				Required:    true,
				Description: "The locale (e.g. en, fr, de)",
			},
			"entries": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Translation key-value pairs",
			},
		},
	}
}

func (r *I18nDictionaryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *I18nDictionaryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan I18nDictionaryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := buildBody(plan)

	result, err := r.client.CreateI18nDictionary(ctx, plan.DomainID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating i18n dictionary", err.Error())
		return
	}

	plan.ID = types.StringValue(result["id"].(string))

	// If entries are specified, update with entries
	if !plan.Entries.IsNull() && !plan.Entries.IsUnknown() {
		entries := make(map[string]string)
		resp.Diagnostics.Append(plan.Entries.ElementsAs(ctx, &entries, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		_, err := r.client.ReplaceI18nDictionaryEntries(ctx, plan.DomainID.ValueString(), plan.ID.ValueString(), entries)
		if err != nil {
			resp.Diagnostics.AddError("Error updating i18n dictionary entries", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *I18nDictionaryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state I18nDictionaryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetI18nDictionary(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading i18n dictionary", err.Error())
		return
	}

	resp.Diagnostics.Append(readIntoModel(ctx, &state, result)...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *I18nDictionaryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan I18nDictionaryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state I18nDictionaryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	body := buildBody(plan)

	_, err := r.client.UpdateI18nDictionary(ctx, plan.DomainID.ValueString(), plan.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating i18n dictionary", err.Error())
		return
	}

	if !plan.Entries.IsNull() && !plan.Entries.IsUnknown() {
		entries := make(map[string]string)
		resp.Diagnostics.Append(plan.Entries.ElementsAs(ctx, &entries, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		_, err := r.client.ReplaceI18nDictionaryEntries(ctx, plan.DomainID.ValueString(), plan.ID.ValueString(), entries)
		if err != nil {
			resp.Diagnostics.AddError("Error updating i18n dictionary entries", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *I18nDictionaryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state I18nDictionaryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteI18nDictionary(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil && !strings.Contains(err.Error(), "404") {
		resp.Diagnostics.AddError("Error deleting i18n dictionary", err.Error())
	}
}

func (r *I18nDictionaryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	domainID, dictionaryID, ok := parseImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/dictionary_id, got: %s", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), domainID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), dictionaryID)...)
}

func parseImportID(id string) (string, string, bool) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func buildBody(plan I18nDictionaryModel) map[string]interface{} {
	return map[string]interface{}{
		"name":   plan.Name.ValueString(),
		"locale": plan.Locale.ValueString(),
	}
}

func readIntoModel(ctx context.Context, model *I18nDictionaryModel, data map[string]interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	if name, ok := data["name"].(string); ok {
		model.Name = types.StringValue(name)
	}
	if locale, ok := data["locale"].(string); ok {
		model.Locale = types.StringValue(locale)
	}
	if entriesRaw, ok := data["entries"].(map[string]interface{}); ok && len(entriesRaw) > 0 {
		entries := make(map[string]string)
		for k, v := range entriesRaw {
			if s, ok := v.(string); ok {
				entries[k] = s
			}
		}
		mapValue, mapDiags := types.MapValueFrom(ctx, types.StringType, entries)
		diags.Append(mapDiags...)
		model.Entries = mapValue
	}
	return diags
}
