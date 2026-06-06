package permissionsmetadata

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

var _ datasource.DataSource = &PermissionsMetadataDataSource{}

type PermissionsMetadataDataSource struct {
	client *client.Client
}

type PermissionsMetadataModel struct {
	Kind                types.String `tfsdk:"kind"`
	DomainID            types.String `tfsdk:"domain_id"`
	ApplicationID       types.String `tfsdk:"application_id"`
	ProtectedResourceID types.String `tfsdk:"protected_resource_id"`
	ResultJSON          types.String `tfsdk:"result_json"`
}

var permissionsMetadataPaths = map[string]func(PermissionsMetadataModel) (string, error){
	"application_member_permissions": func(config PermissionsMetadataModel) (string, error) {
		if missing(config.DomainID) {
			return "", metadataError("domain_id is required for application_member_permissions")
		}
		if missing(config.ApplicationID) {
			return "", metadataError("application_id is required for application_member_permissions")
		}
		return "/domains/" + config.DomainID.ValueString() + "/applications/" + config.ApplicationID.ValueString() + "/members/permissions", nil
	},
	"domain_member_permissions": func(config PermissionsMetadataModel) (string, error) {
		if missing(config.DomainID) {
			return "", metadataError("domain_id is required for domain_member_permissions")
		}
		return "/domains/" + config.DomainID.ValueString() + "/members/permissions", nil
	},
	"environment_member_permissions": func(_ PermissionsMetadataModel) (string, error) {
		return "/members/permissions", nil
	},
	"protected_resource_member_permissions": func(config PermissionsMetadataModel) (string, error) {
		if missing(config.DomainID) {
			return "", metadataError("domain_id is required for protected_resource_member_permissions")
		}
		if missing(config.ProtectedResourceID) {
			return "", metadataError("protected_resource_id is required for protected_resource_member_permissions")
		}
		return "/domains/" + config.DomainID.ValueString() + "/protected-resources/" + config.ProtectedResourceID.ValueString() + "/members/permissions", nil
	},
}

type metadataError string

func (e metadataError) Error() string {
	return string(e)
}

func NewPermissionsMetadataDataSource() datasource.DataSource {
	return &PermissionsMetadataDataSource{}
}

func (d *PermissionsMetadataDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_permissions_metadata"
}

func (d *PermissionsMetadataDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads Gravitee AM member permission metadata for environments, domains, applications, and protected resources",
		Attributes: map[string]schema.Attribute{
			"kind": schema.StringAttribute{
				Required:    true,
				Description: "Permission metadata kind. Supported values: " + permissionsMetadataKindList(),
			},
			"domain_id": schema.StringAttribute{
				Optional:    true,
				Description: "Domain ID, required for domain, application, and protected resource permission metadata",
			},
			"application_id": schema.StringAttribute{
				Optional:    true,
				Description: "Application ID, required for application_member_permissions",
			},
			"protected_resource_id": schema.StringAttribute{
				Optional:    true,
				Description: "Protected resource ID, required for protected_resource_member_permissions",
			},
			"result_json": schema.StringAttribute{
				Computed:    true,
				Description: "JSON string containing the permission metadata response",
			},
		},
	}
}

func (d *PermissionsMetadataDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected DataSource Configure Type", "Expected *client.Client")
		return
	}
	d.client = c
}

func (d *PermissionsMetadataDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config PermissionsMetadataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pathBuilder, ok := permissionsMetadataPaths[config.Kind.ValueString()]
	if !ok {
		resp.Diagnostics.AddError("Unsupported permission metadata kind", "Expected one of: "+permissionsMetadataKindList())
		return
	}
	path, err := pathBuilder(config)
	if err != nil {
		resp.Diagnostics.AddError("Invalid permission metadata configuration", err.Error())
		return
	}

	raw, err := d.client.GetPermissionsMetadata(ctx, path)
	if err != nil {
		resp.Diagnostics.AddError("Error reading permission metadata", err.Error())
		return
	}
	result, err := formatJSON(raw)
	if err != nil {
		resp.Diagnostics.AddError("Error formatting permission metadata", err.Error())
		return
	}

	config.ResultJSON = types.StringValue(result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func missing(value types.String) bool {
	return value.IsNull() || value.IsUnknown() || value.ValueString() == ""
}

func formatJSON(raw []byte) (string, error) {
	if len(raw) == 0 {
		return "null", nil
	}
	var parsed interface{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		formatted, marshalErr := json.Marshal(string(raw))
		if marshalErr != nil {
			return "", marshalErr
		}
		return string(formatted), nil
	}
	formatted, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return "", err
	}
	return string(formatted), nil
}

func permissionsMetadataKindList() string {
	kinds := make([]string, 0, len(permissionsMetadataPaths))
	for kind := range permissionsMetadataPaths {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return strings.Join(kinds, ", ")
}
