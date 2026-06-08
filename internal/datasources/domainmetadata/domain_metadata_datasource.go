package domainmetadata

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

var _ datasource.DataSource = &DomainMetadataDataSource{}

type DomainMetadataDataSource struct {
	client *client.Client
}

type DomainMetadataModel struct {
	DomainID      types.String `tfsdk:"domain_id"`
	Kind          types.String `tfsdk:"kind"`
	CertificateID types.String `tfsdk:"certificate_id"`
	ResultJSON    types.String `tfsdk:"result_json"`
}

var domainMetadataPaths = map[string]func(DomainMetadataModel) (string, error){
	"active_password_policy": func(_ DomainMetadataModel) (string, error) {
		return "password-policies/activePolicy", nil
	},
	"certificate_key": func(config DomainMetadataModel) (string, error) {
		if config.CertificateID.IsNull() || config.CertificateID.IsUnknown() || config.CertificateID.ValueString() == "" {
			return "", errMissingCertificateID
		}
		return "certificates/" + config.CertificateID.ValueString() + "/key", nil
	},
	"certificate_keys": func(config DomainMetadataModel) (string, error) {
		if config.CertificateID.IsNull() || config.CertificateID.IsUnknown() || config.CertificateID.ValueString() == "" {
			return "", errMissingCertificateID
		}
		return "certificates/" + config.CertificateID.ValueString() + "/keys", nil
	},
}

var errMissingCertificateID = &metadataError{message: "certificate_id is required for certificate metadata kinds"}

type metadataError struct {
	message string
}

func (e *metadataError) Error() string {
	return e.message
}

func NewDomainMetadataDataSource() datasource.DataSource {
	return &DomainMetadataDataSource{}
}

func (d *DomainMetadataDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_metadata"
}

func (d *DomainMetadataDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads Gravitee AM domain metadata such as active password policy and certificate public keys",
		Attributes: map[string]schema.Attribute{
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain",
			},
			"kind": schema.StringAttribute{
				Required:    true,
				Description: "Metadata kind. Supported values: " + domainMetadataKindList(),
			},
			"certificate_id": schema.StringAttribute{
				Optional:    true,
				Description: "Certificate ID, required for certificate_key and certificate_keys",
			},
			"result_json": schema.StringAttribute{
				Computed:    true,
				Description: "JSON string containing the domain metadata response",
			},
		},
	}
}

func (d *DomainMetadataDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DomainMetadataDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DomainMetadataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pathBuilder, ok := domainMetadataPaths[config.Kind.ValueString()]
	if !ok {
		resp.Diagnostics.AddError("Unsupported domain metadata kind", "Expected one of: "+domainMetadataKindList())
		return
	}
	path, err := pathBuilder(config)
	if err != nil {
		resp.Diagnostics.AddError("Invalid domain metadata configuration", err.Error())
		return
	}

	raw, err := d.client.GetDomainMetadata(ctx, config.DomainID.ValueString(), path)
	if err != nil {
		resp.Diagnostics.AddError("Error reading domain metadata", err.Error())
		return
	}
	result, err := formatJSON(raw)
	if err != nil {
		resp.Diagnostics.AddError("Error formatting domain metadata", err.Error())
		return
	}

	config.ResultJSON = types.StringValue(result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
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

func domainMetadataKindList() string {
	kinds := make([]string, 0, len(domainMetadataPaths))
	for kind := range domainMetadataPaths {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return strings.Join(kinds, ", ")
}
