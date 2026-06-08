package adminmetadata

import (
	"context"
	"encoding/json"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

var _ datasource.DataSource = &AdminMetadataDataSource{}

type AdminMetadataDataSource struct {
	client *client.Client
}

type AdminMetadataModel struct {
	Kind       types.String `tfsdk:"kind"`
	DomainID   types.String `tfsdk:"domain_id"`
	UserID     types.String `tfsdk:"user_id"`
	HRID       types.String `tfsdk:"hrid"`
	Size       types.Int64  `tfsdk:"size"`
	ResultJSON types.String `tfsdk:"result_json"`
}

type metadataRequest struct {
	scope string
	path  string
}

var adminMetadataPaths = map[string]func(AdminMetadataModel) (metadataRequest, error){
	"domain_by_hrid": func(config AdminMetadataModel) (metadataRequest, error) {
		if missingString(config.HRID) {
			return metadataRequest{}, metadataError("hrid is required for domain_by_hrid")
		}
		return metadataRequest{
			scope: "environment",
			path:  "/domains/_hrid/" + url.PathEscape(config.HRID.ValueString()),
		}, nil
	},
	"organization_audits": func(config AdminMetadataModel) (metadataRequest, error) {
		return metadataRequest{
			scope: "organization",
			path:  "/audits" + pagingQuery(config),
		}, nil
	},
	"organization_environments": func(_ AdminMetadataModel) (metadataRequest, error) {
		return metadataRequest{
			scope: "organization",
			path:  "/environments",
		}, nil
	},
	"user_audits": func(config AdminMetadataModel) (metadataRequest, error) {
		if missingString(config.DomainID) {
			return metadataRequest{}, metadataError("domain_id is required for user_audits")
		}
		if missingString(config.UserID) {
			return metadataRequest{}, metadataError("user_id is required for user_audits")
		}
		return metadataRequest{
			scope: "environment",
			path: "/domains/" + url.PathEscape(config.DomainID.ValueString()) +
				"/users/" + url.PathEscape(config.UserID.ValueString()) + "/audits" + pagingQuery(config),
		}, nil
	},
}

type metadataError string

func (e metadataError) Error() string {
	return string(e)
}

func NewAdminMetadataDataSource() datasource.DataSource {
	return &AdminMetadataDataSource{}
}

func (d *AdminMetadataDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_admin_metadata"
}

func (d *AdminMetadataDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads Gravitee AM administrative metadata such as organization audits, environments, domain HRID lookups, and user audits",
		Attributes: map[string]schema.Attribute{
			"kind": schema.StringAttribute{
				Required:    true,
				Description: "Administrative metadata kind. Supported values: " + adminMetadataKindList(),
			},
			"domain_id": schema.StringAttribute{
				Optional:    true,
				Description: "Domain ID, required for user_audits",
			},
			"user_id": schema.StringAttribute{
				Optional:    true,
				Description: "User ID, required for user_audits",
			},
			"hrid": schema.StringAttribute{
				Optional:    true,
				Description: "Domain human-readable ID, required for domain_by_hrid",
			},
			"size": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum audit entries to retrieve for audit kinds (default: 10)",
			},
			"result_json": schema.StringAttribute{
				Computed:    true,
				Description: "JSON string containing the administrative metadata response",
			},
		},
	}
}

func (d *AdminMetadataDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AdminMetadataDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config AdminMetadataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requestBuilder, ok := adminMetadataPaths[config.Kind.ValueString()]
	if !ok {
		resp.Diagnostics.AddError("Unsupported administrative metadata kind", "Expected one of: "+adminMetadataKindList())
		return
	}
	metadataReq, err := requestBuilder(config)
	if err != nil {
		resp.Diagnostics.AddError("Invalid administrative metadata configuration", err.Error())
		return
	}

	var raw []byte
	switch metadataReq.scope {
	case "organization":
		raw, err = d.client.GetOrganizationMetadata(ctx, metadataReq.path)
	default:
		raw, err = d.client.GetAdminMetadata(ctx, metadataReq.path)
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading administrative metadata", err.Error())
		return
	}
	result, err := formatJSON(raw)
	if err != nil {
		resp.Diagnostics.AddError("Error formatting administrative metadata", err.Error())
		return
	}

	config.ResultJSON = types.StringValue(result)
	if config.Size.IsNull() || config.Size.IsUnknown() {
		config.Size = types.Int64Value(10)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func missingString(value types.String) bool {
	return value.IsNull() || value.IsUnknown() || value.ValueString() == ""
}

func pagingQuery(config AdminMetadataModel) string {
	size := int64(10)
	if !config.Size.IsNull() && !config.Size.IsUnknown() {
		size = config.Size.ValueInt64()
	}
	query := url.Values{}
	query.Set("page", "0")
	query.Set("size", strconv.FormatInt(size, 10))
	return "?" + query.Encode()
}

func formatJSON(raw []byte) (string, error) {
	if len(raw) == 0 {
		return "null", nil
	}
	var parsed interface{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		formatted, _ := json.Marshal(string(raw))
		return string(formatted), nil
	}
	formatted, _ := json.MarshalIndent(parsed, "", "  ")
	return string(formatted), nil
}

func adminMetadataKindList() string {
	kinds := make([]string, 0, len(adminMetadataPaths))
	for kind := range adminMetadataPaths {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return strings.Join(kinds, ", ")
}
