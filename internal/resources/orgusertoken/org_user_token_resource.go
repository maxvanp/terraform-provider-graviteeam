package orgusertoken

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
	_ resource.Resource                = &OrgUserTokenResource{}
	_ resource.ResourceWithImportState = &OrgUserTokenResource{}
)

type OrgUserTokenResource struct {
	client *client.Client
}

func NewOrgUserTokenResource() resource.Resource {
	return &OrgUserTokenResource{}
}

func (r *OrgUserTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org_user_token"
}

func (r *OrgUserTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM organization user account access token",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The Terraform ID of the organization user token",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the organization user",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"token_id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the generated token",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The token name",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"token": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The generated token value. Gravitee AM returns this only at creation time.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *OrgUserTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrgUserTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrgUserTokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CreateOrgUserToken(ctx, plan.UserID.ValueString(), map[string]interface{}{
		"name": plan.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating organization user token", err.Error())
		return
	}

	readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgUserTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrgUserTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	token, err := r.readToken(ctx, state.UserID.ValueString(), state.TokenID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading organization user token", err.Error())
		return
	}

	savedToken := state.Token
	readIntoModel(&state, token)
	state.Token = savedToken
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OrgUserTokenResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unsupported Organization User Token Update", "Organization user token attributes require replacement.")
}

func (r *OrgUserTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrgUserTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteOrgUserToken(ctx, state.UserID.ValueString(), state.TokenID.ValueString())
	if err != nil && !strings.Contains(err.Error(), "404") {
		resp.Diagnostics.AddError("Error deleting organization user token", err.Error())
	}
}

func (r *OrgUserTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: user_id/token_id, got: %s", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("token_id"), parts[1])...)
}

func (r *OrgUserTokenResource) readToken(ctx context.Context, userID, tokenID string) (map[string]interface{}, error) {
	tokens, err := r.client.ListOrgUserTokens(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, token := range tokens {
		if id, ok := token["tokenId"].(string); ok && id == tokenID {
			return token, nil
		}
	}
	return nil, fmt.Errorf("organization user token not found")
}

func readIntoModel(model *OrgUserTokenModel, data map[string]interface{}) {
	if tokenID, ok := data["tokenId"].(string); ok {
		model.TokenID = types.StringValue(tokenID)
		model.ID = types.StringValue(model.UserID.ValueString() + "/" + tokenID)
	}
	if name, ok := data["name"].(string); ok {
		model.Name = types.StringValue(name)
	}
	if token, ok := data["token"].(string); ok && token != "" {
		model.Token = types.StringValue(token)
	}
}
