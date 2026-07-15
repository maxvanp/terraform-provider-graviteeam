package applications

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

var _ datasource.DataSource = &ApplicationsDataSource{}

type ApplicationsDataSource struct {
	client *client.Client
}

type ApplicationsModel struct {
	DomainID         types.String   `tfsdk:"domain_id"`
	PaginationMode   types.String   `tfsdk:"pagination_mode"`
	Limit            types.Int64    `tfsdk:"limit"`
	Sort             types.String   `tfsdk:"sort"`
	Direction        types.String   `tfsdk:"direction"`
	Page             types.Int64    `tfsdk:"page"`
	Expand           []types.String `tfsdk:"expand"`
	Query            types.String   `tfsdk:"query"`
	Status           types.String   `tfsdk:"status"`
	OwnerEmail       types.String   `tfsdk:"owner_email"`
	ApplicationTypes []types.String `tfsdk:"application_types"`
	Cursor           types.String   `tfsdk:"cursor"`
	ResultJSON       types.String   `tfsdk:"result_json"`
}

func NewApplicationsDataSource() datasource.DataSource {
	return &ApplicationsDataSource{}
}

func (d *ApplicationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_applications"
}

func (d *ApplicationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Searches Gravitee AM applications with page or cursor pagination",
		Attributes: map[string]schema.Attribute{
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain",
			},
			"pagination_mode": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Pagination endpoint to use: page or cursor (default: page)",
				Validators: []validator.String{
					enumStringValidator{attribute: "pagination_mode", allowed: []string{"page", "cursor"}},
				},
			},
			"limit": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Maximum number of applications to return (default: 50)",
			},
			"sort": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Application field used for sorting (default: updatedAt)",
			},
			"direction": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Sort direction: ASC or DESC (default: DESC)",
				Validators: []validator.String{
					enumStringValidator{attribute: "direction", allowed: []string{"ASC", "DESC"}},
				},
			},
			"page": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Zero-based page number (default: 0)",
			},
			"expand": schema.ListAttribute{
				Optional:    true,
				Description: "Application relationships to expand",
				ElementType: types.StringType,
			},
			"query": schema.StringAttribute{
				Optional:    true,
				Description: "Free-text application search query",
			},
			"status": schema.StringAttribute{
				Optional:    true,
				Description: "Application status filter: enabled or disabled",
				Validators: []validator.String{
					enumStringValidator{attribute: "status", allowed: []string{"enabled", "disabled"}},
				},
			},
			"owner_email": schema.StringAttribute{
				Optional:    true,
				Description: "Application owner email filter",
			},
			"application_types": schema.ListAttribute{
				Optional:    true,
				Description: "Application type filters: WEB, NATIVE, BROWSER, SERVICE, RESOURCE_SERVER, or AGENT",
				ElementType: types.StringType,
			},
			"cursor": schema.StringAttribute{
				Optional:    true,
				Description: "Cursor supplied by a previous cursor-mode response",
			},
			"result_json": schema.StringAttribute{
				Computed:    true,
				Description: "JSON string containing the complete application search response",
			},
		},
	}
}

func (d *ApplicationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ApplicationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ApplicationsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applyDefaults(&config)
	query, cursorMode, err := buildQuery(config)
	if err != nil {
		resp.Diagnostics.AddError("Invalid application search configuration", err.Error())
		return
	}
	raw, err := d.client.SearchApplications(ctx, config.DomainID.ValueString(), cursorMode, query)
	if err != nil {
		resp.Diagnostics.AddError("Error searching applications", err.Error())
		return
	}
	config.ResultJSON = types.StringValue(formatJSON(raw))
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func applyDefaults(model *ApplicationsModel) {
	if model.PaginationMode.IsNull() || model.PaginationMode.IsUnknown() || model.PaginationMode.ValueString() == "" {
		model.PaginationMode = types.StringValue("page")
	}
	if model.Limit.IsNull() || model.Limit.IsUnknown() {
		model.Limit = types.Int64Value(50)
	}
	if model.Sort.IsNull() || model.Sort.IsUnknown() || model.Sort.ValueString() == "" {
		model.Sort = types.StringValue("updatedAt")
	}
	if model.Direction.IsNull() || model.Direction.IsUnknown() || model.Direction.ValueString() == "" {
		model.Direction = types.StringValue("DESC")
	}
	if model.Page.IsNull() || model.Page.IsUnknown() {
		model.Page = types.Int64Value(0)
	}
}

func buildQuery(model ApplicationsModel) (url.Values, bool, error) {
	if model.Limit.ValueInt64() <= 0 {
		return nil, false, fmt.Errorf("limit must be greater than zero")
	}
	if model.Page.ValueInt64() < 0 {
		return nil, false, fmt.Errorf("page must be zero or greater")
	}

	cursorMode := model.PaginationMode.ValueString() == "cursor"
	if !cursorMode && !model.Cursor.IsNull() && !model.Cursor.IsUnknown() && model.Cursor.ValueString() != "" {
		return nil, false, fmt.Errorf("cursor requires pagination_mode = cursor")
	}

	query := url.Values{}
	query.Set("limit", strconv.FormatInt(model.Limit.ValueInt64(), 10))
	query.Set("sort", model.Sort.ValueString())
	query.Set("dir", model.Direction.ValueString())
	query.Set("page", strconv.FormatInt(model.Page.ValueInt64(), 10))
	addStringValue(query, "q", model.Query)
	addStringValue(query, "status", model.Status)
	addStringValue(query, "owner.email", model.OwnerEmail)
	if cursorMode {
		addStringValue(query, "cursor", model.Cursor)
	}
	for _, value := range model.Expand {
		query.Add("expand", value.ValueString())
	}
	allowedTypes := map[string]bool{
		"WEB": true, "NATIVE": true, "BROWSER": true, "SERVICE": true,
		"RESOURCE_SERVER": true, "AGENT": true,
	}
	for _, value := range model.ApplicationTypes {
		applicationType := value.ValueString()
		if !allowedTypes[applicationType] {
			return nil, false, fmt.Errorf("unsupported application type %q", applicationType)
		}
		query.Add("type", applicationType)
	}
	return query, cursorMode, nil
}

func addStringValue(query url.Values, key string, value types.String) {
	if !value.IsNull() && !value.IsUnknown() && value.ValueString() != "" {
		query.Set(key, value.ValueString())
	}
}

func formatJSON(raw []byte) string {
	if len(raw) == 0 {
		return "null"
	}
	var parsed interface{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		formatted, _ := json.Marshal(string(raw))
		return string(formatted)
	}
	formatted, _ := json.MarshalIndent(parsed, "", "  ")
	return string(formatted)
}

type enumStringValidator struct {
	attribute string
	allowed   []string
}

func (v enumStringValidator) Description(_ context.Context) string {
	return v.attribute + " must be one of: " + strings.Join(v.allowed, ", ")
}

func (v enumStringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v enumStringValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	for _, allowed := range v.allowed {
		if req.ConfigValue.ValueString() == allowed {
			return
		}
	}
	resp.Diagnostics.AddError("Invalid "+v.attribute, v.Description(context.Background()))
}
