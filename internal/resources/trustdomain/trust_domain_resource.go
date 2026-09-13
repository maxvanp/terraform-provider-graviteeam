package trustdomain

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

var (
	_ resource.Resource                = &TrustDomainResource{}
	_ resource.ResourceWithImportState = &TrustDomainResource{}
)

type TrustDomainResource struct {
	client *client.Client
}

func NewTrustDomainResource() resource.Resource {
	return &TrustDomainResource{}
}

func (r *TrustDomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_trust_domain"
}

func (r *TrustDomainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM workload identity trust domain",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the trust domain",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain this trust domain belongs to",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The SPIFFE trust domain name",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "A description of the trust domain",
			},
			"bundle_source": schema.StringAttribute{
				Required:    true,
				Description: "The trust bundle source. Gravitee AM 4.12 supports JWKS_URL",
				Validators:  []validator.String{bundleSourceValidator{}},
			},
			"jwks_url": schema.StringAttribute{
				Required:    true,
				Description: "The URL from which Gravitee AM refreshes the JWKS trust bundle",
			},
			"refresh_interval_seconds": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The JWKS refresh interval in seconds",
				Validators:  []validator.Int64{positiveInt64Validator{}},
			},
			"allowed_algorithms": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				Description: "JWT signature algorithms accepted from this trust domain",
				ElementType: types.StringType,
			},
			"created_at": schema.Int64Attribute{
				Computed:    true,
				Description: "Creation time as an epoch timestamp in milliseconds",
			},
			"updated_at": schema.Int64Attribute{
				Computed:    true,
				Description: "Last update time as an epoch timestamp in milliseconds",
			},
		},
	}
}

func (r *TrustDomainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TrustDomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TrustDomainModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CreateTrustDomain(ctx, plan.DomainID.ValueString(), buildBody(plan, true))
	if err != nil {
		resp.Diagnostics.AddError("Error creating trust domain", err.Error())
		return
	}
	r.readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TrustDomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TrustDomainModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetTrustDomain(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading trust domain", err.Error())
		return
	}
	r.readIntoModel(&state, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TrustDomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TrustDomainModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.UpdateTrustDomain(ctx, plan.DomainID.ValueString(), plan.ID.ValueString(), buildBody(plan, false))
	if err != nil {
		resp.Diagnostics.AddError("Error updating trust domain", err.Error())
		return
	}
	r.readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TrustDomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TrustDomainModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteTrustDomain(ctx, state.DomainID.ValueString(), state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting trust domain", err.Error())
	}
}

func (r *TrustDomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	domainID, trustDomainID, ok := parseImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/trust_domain_id, got: %s", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), domainID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), trustDomainID)...)
}

func parseImportID(id string) (string, string, bool) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func buildBody(model TrustDomainModel, includeName bool) map[string]interface{} {
	body := map[string]interface{}{
		"bundleSource": model.BundleSource.ValueString(),
		"jwksUrl":      model.JWKsURL.ValueString(),
	}
	if includeName {
		body["name"] = model.Name.ValueString()
	}
	if !model.Description.IsNull() && !model.Description.IsUnknown() {
		body["description"] = model.Description.ValueString()
	}
	if !model.RefreshIntervalSeconds.IsNull() && !model.RefreshIntervalSeconds.IsUnknown() {
		body["refreshIntervalSeconds"] = model.RefreshIntervalSeconds.ValueInt64()
	}
	if model.AllowedAlgorithms != nil {
		algorithms := make([]string, len(model.AllowedAlgorithms))
		for i, algorithm := range model.AllowedAlgorithms {
			algorithms[i] = algorithm.ValueString()
		}
		body["allowedAlgorithms"] = algorithms
	}
	return body
}

func (r *TrustDomainResource) readIntoModel(model *TrustDomainModel, data map[string]interface{}) {
	model.ID = readString(data, "id")
	model.Name = readString(data, "name")
	model.Description = readString(data, "description")
	model.BundleSource = readStringUpper(data, "bundleSource")
	model.JWKsURL = readString(data, "jwksUrl")
	model.RefreshIntervalSeconds = readInt64(data, "refreshIntervalSeconds")
	model.CreatedAt = readInt64(data, "createdAt")
	model.UpdatedAt = readInt64(data, "updatedAt")
	model.AllowedAlgorithms = readStringList(data, "allowedAlgorithms")
}

func readString(data map[string]interface{}, key string) types.String {
	if value, ok := data[key].(string); ok {
		return types.StringValue(value)
	}
	return types.StringNull()
}

func readStringUpper(data map[string]interface{}, key string) types.String {
	value := readString(data, key)
	if value.IsNull() {
		return value
	}
	return types.StringValue(strings.ToUpper(value.ValueString()))
}

func readInt64(data map[string]interface{}, key string) types.Int64 {
	switch value := data[key].(type) {
	case float64:
		return types.Int64Value(int64(value))
	case int64:
		return types.Int64Value(value)
	default:
		return types.Int64Null()
	}
}

func readStringList(data map[string]interface{}, key string) []types.String {
	values, ok := data[key].([]interface{})
	if !ok {
		return nil
	}
	result := make([]types.String, 0, len(values))
	for _, value := range values {
		if text, ok := value.(string); ok {
			result = append(result, types.StringValue(text))
		}
	}
	return result
}

type bundleSourceValidator struct{}

func (bundleSourceValidator) Description(_ context.Context) string {
	return "bundle_source must be JWKS_URL"
}

func (bundleSourceValidator) MarkdownDescription(_ context.Context) string {
	return "`bundle_source` must be `JWKS_URL`"
}

func (bundleSourceValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if req.ConfigValue.ValueString() != "JWKS_URL" {
		resp.Diagnostics.AddError("Unsupported trust bundle source", "Gravitee AM 4.12 supports only JWKS_URL trust domains")
	}
}

type positiveInt64Validator struct{}

func (positiveInt64Validator) Description(_ context.Context) string {
	return "value must be greater than zero"
}

func (positiveInt64Validator) MarkdownDescription(_ context.Context) string {
	return "Value must be greater than zero."
}

func (positiveInt64Validator) ValidateInt64(_ context.Context, req validator.Int64Request, resp *validator.Int64Response) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if req.ConfigValue.ValueInt64() <= 0 {
		resp.Diagnostics.AddError("Invalid refresh interval", "refresh_interval_seconds must be greater than zero")
	}
}
