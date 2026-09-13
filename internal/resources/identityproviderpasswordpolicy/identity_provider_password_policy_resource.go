package identityproviderpasswordpolicy

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
	_ resource.Resource                = &IdentityProviderPasswordPolicyResource{}
	_ resource.ResourceWithImportState = &IdentityProviderPasswordPolicyResource{}
)

type IdentityProviderPasswordPolicyResource struct {
	client *client.Client
}

func NewIdentityProviderPasswordPolicyResource() resource.Resource {
	return &IdentityProviderPasswordPolicyResource{}
}

func (r *IdentityProviderPasswordPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_provider_password_policy"
}

func (r *IdentityProviderPasswordPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Assigns a password policy to a Gravitee AM identity provider",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The Terraform ID of the identity provider password policy assignment",
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
			"identity_provider_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the identity provider",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"password_policy_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the password policy to assign",
			},
		},
	}
}

func (r *IdentityProviderPasswordPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *IdentityProviderPasswordPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan IdentityProviderPasswordPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.AssignIdentityProviderPasswordPolicy(ctx, plan.DomainID.ValueString(), plan.IdentityProviderID.ValueString(), plan.PasswordPolicyID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error assigning identity provider password policy", err.Error())
		return
	}

	plan.ID = assignmentID(plan.DomainID.ValueString(), plan.IdentityProviderID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IdentityProviderPasswordPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state IdentityProviderPasswordPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetIdentityProvider(ctx, state.DomainID.ValueString(), state.IdentityProviderID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading identity provider password policy", err.Error())
		return
	}

	passwordPolicyID, ok := result["passwordPolicy"].(string)
	if !ok || passwordPolicyID == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	readIntoModel(&state, passwordPolicyID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *IdentityProviderPasswordPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan IdentityProviderPasswordPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.AssignIdentityProviderPasswordPolicy(ctx, plan.DomainID.ValueString(), plan.IdentityProviderID.ValueString(), plan.PasswordPolicyID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error updating identity provider password policy", err.Error())
		return
	}

	plan.ID = assignmentID(plan.DomainID.ValueString(), plan.IdentityProviderID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IdentityProviderPasswordPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state IdentityProviderPasswordPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.ClearIdentityProviderPasswordPolicy(ctx, state.DomainID.ValueString(), state.IdentityProviderID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error clearing identity provider password policy", err.Error())
	}
}

func (r *IdentityProviderPasswordPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	domainID, identityProviderID, ok := parseAssignmentImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/identity_provider_id, got: %s", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), domainID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("identity_provider_id"), identityProviderID)...)
}

func assignmentID(domainID, identityProviderID string) types.String {
	return types.StringValue(domainID + "/" + identityProviderID)
}

func parseAssignmentImportID(id string) (string, string, bool) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func readIntoModel(model *IdentityProviderPasswordPolicyModel, passwordPolicyID string) {
	model.ID = assignmentID(model.DomainID.ValueString(), model.IdentityProviderID.ValueString())
	model.PasswordPolicyID = types.StringValue(passwordPolicyID)
}
