// Copyright (c) JFrog Ltd. (2025)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package datasource

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jfrog/terraform-provider-shared/util"
)

const (
	applicationsEndpoint = "apptrust/api/v1/applications"
)

var _ datasource.DataSource = &ApplicationsDataSource{}

func NewApplicationsDataSource() datasource.DataSource {
	return &ApplicationsDataSource{}
}

type ApplicationsDataSource struct {
	ProviderData util.ProviderMetadata
}

type ApplicationsDataSourceModel struct {
	ProjectKey    types.String `tfsdk:"project_key"`
	Name          types.String `tfsdk:"name"`
	Owners        types.List   `tfsdk:"owners"`
	MaturityLevel types.String `tfsdk:"maturity"`
	Criticality   types.String `tfsdk:"criticality"`
	Labels        types.List   `tfsdk:"labels"`
	OrderBy       types.String `tfsdk:"order_by"`
	OrderAsc      types.Bool   `tfsdk:"order_asc"`
	Offset        types.Int64  `tfsdk:"offset"`
	Limit         types.Int64  `tfsdk:"limit"`
	Applications  types.List   `tfsdk:"applications"`
	Total         types.Int64  `tfsdk:"total"`
}

// SingleApplicationResponse matches the API response structure for GET /v1/applications
// The API returns an array of these objects directly
type SingleApplicationResponse struct {
	ApplicationKey  string            `json:"application_key"`
	ApplicationName string            `json:"application_name"`
	ProjectKey      string            `json:"project_key"`
	Description     string            `json:"description,omitempty"`
	MaturityLevel   string            `json:"maturity_level,omitempty"`
	Criticality     string            `json:"criticality,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	UserOwners      []string          `json:"user_owners,omitempty"`
	GroupOwners     []string          `json:"group_owners,omitempty"`
}

type ApplicationListItemAPIModel struct {
	ProjectKey               string `json:"project_key"`
	ApplicationName          string `json:"application_name"`
	ApplicationKey           string `json:"application_key"`
	ApplicationVersionLatest string `json:"application_version_latest,omitempty"`
	ApplicationVersionTag    string `json:"application_version_tag,omitempty"`
	ApplicationVersionsCount int    `json:"application_versions_count,omitempty"`
}

type ApplicationsListAPIModel struct {
	Applications []ApplicationListItemAPIModel
	Total        int
	Limit        int
	Offset       int
}

var (
	maturityLevels    = []string{"unspecified", "experimental", "production", "end_of_life"}
	criticalityLevels = []string{"unspecified", "low", "medium", "high", "critical"}
	orderByOptions    = []string{"name", "created"}
)

var applicationListItemAttrType = map[string]attr.Type{
	"project_key":                types.StringType,
	"application_name":           types.StringType,
	"application_key":            types.StringType,
	"application_version_latest": types.StringType,
	"application_version_tag":    types.StringType,
	"application_versions_count": types.Int64Type,
}

func (d *ApplicationsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_applications"
}

func (d *ApplicationsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Returns a list of AppTrust applications, including the latest version and the total number of versions. " +
			"Supports filtering, pagination, and sorting.\n\n" +
			"## API Notes\n\n" +
			"- The API endpoint `GET /v1/applications` supports filtering by project_key, name, criticality, maturity, label, and owner (each filter can be specified multiple times where applicable).\n" +
			"- The `maturity` query parameter is used for filtering (not `maturity_level`); the response uses `maturity_level` in application objects.\n" +
			"- Pagination is supported via `limit` (default 100) and `offset` (default 0).\n" +
			"- Ordering is via `order_by` (name or created; default created) and `order_asc` (default false).",
		Attributes: map[string]schema.Attribute{
			"project_key": schema.StringAttribute{
				Description: "The key of the project associated with the application. If not specified, applications from all projects will be returned.",
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "Filters results by the application name.",
				Optional:    true,
			},
			"owners": schema.ListAttribute{
				Description: "Filters results by application owners (user or group). This filter can be used multiple times.",
				ElementType: types.StringType,
				Optional:    true,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						stringvalidator.LengthAtLeast(1),
					),
				},
			},
			"maturity": schema.StringAttribute{
				Description: fmt.Sprintf("Filters results by application maturity. Allowed values: %s", strings.Join(maturityLevels, ", ")),
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(maturityLevels...),
				},
			},
			"criticality": schema.StringAttribute{
				Description: fmt.Sprintf("Filters results by application criticality. Allowed values: %s", strings.Join(criticalityLevels, ", ")),
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(criticalityLevels...),
				},
			},
			"labels": schema.ListAttribute{
				Description: "Filters by application labels in the format 'key:value'. Can be specified multiple times (once per label). " +
					"Example: [\"environment:production\", \"region:us-east\"]",
				ElementType: types.StringType,
				Optional:    true,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^[^:]+:[^:]+$`),
							"label must be in format 'key:value'",
						),
					),
				},
			},
			"order_by": schema.StringAttribute{
				Description: fmt.Sprintf("Defines whether to order the applications by name or created. Allowed values: %s. API default is 'created'.", strings.Join(orderByOptions, ", ")),
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(orderByOptions...),
				},
			},
			"order_asc": schema.BoolAttribute{
				Description: "Defines whether to list the applications in ascending (true) or descending (false) order. API default is false.",
				Optional:    true,
			},
			"offset": schema.Int64Attribute{
				Description: "Sets the number of records to skip before returning the query response. Used for pagination. API default is 0.",
				Optional:    true,
			},
			"limit": schema.Int64Attribute{
				Description: "Sets the maximum number of applications to return at one time. Used for pagination. API default is 100.",
				Optional:    true,
			},
			"applications": schema.ListNestedAttribute{
				Description: "List of applications.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project_key": schema.StringAttribute{
							Description: "The key of the project associated with the application.",
							Computed:    true,
						},
						"application_name": schema.StringAttribute{
							Description: "The application display name.",
							Computed:    true,
						},
						"application_key": schema.StringAttribute{
							Description: "The application key.",
							Computed:    true,
						},
						"application_version_latest": schema.StringAttribute{
							Description: "The latest version of the application.",
							Computed:    true,
						},
						"application_version_tag": schema.StringAttribute{
							Description: "The tag associated with the latest application version.",
							Computed:    true,
						},
						"application_versions_count": schema.Int64Attribute{
							Description: "The total number of versions for this application.",
							Computed:    true,
						},
					},
				},
			},
			"total": schema.Int64Attribute{
				Description: "Total number of applications matching the filter criteria.",
				Computed:    true,
			},
		},
	}
}

func (d *ApplicationsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (d *ApplicationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ApplicationsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading applications datasource", map[string]interface{}{
		"project_key": data.ProjectKey.ValueString(),
	})

	// Build query params like unifiedpolicy: url.Values with Set for single, Add for multi. List API uses "maturity" (not "maturity_level").
	queryValues := url.Values{}
	if !data.ProjectKey.IsNull() {
		queryValues.Set("project_key", data.ProjectKey.ValueString())
	}
	if !data.Name.IsNull() {
		queryValues.Set("name", data.Name.ValueString())
	}
	if !data.MaturityLevel.IsNull() {
		queryValues.Set("maturity", data.MaturityLevel.ValueString())
	}
	if !data.Criticality.IsNull() {
		queryValues.Set("criticality", data.Criticality.ValueString())
	}
	if !data.OrderBy.IsNull() {
		queryValues.Set("order_by", data.OrderBy.ValueString())
	}
	if !data.OrderAsc.IsNull() {
		queryValues.Set("order_asc", strconv.FormatBool(data.OrderAsc.ValueBool()))
	}
	if !data.Offset.IsNull() {
		queryValues.Set("offset", strconv.FormatInt(data.Offset.ValueInt64(), 10))
	}
	if !data.Limit.IsNull() {
		queryValues.Set("limit", strconv.FormatInt(data.Limit.ValueInt64(), 10))
	}

	// Handle owners - can be multiple (need to add each one separately)
	// NOTE: The API supports multiple "owner" query parameters for filtering
	// Resty will append multiple query params with the same key, which is the expected API behavior
	if !data.Owners.IsNull() {
		var owners []string
		resp.Diagnostics.Append(data.Owners.ElementsAs(ctx, &owners, false)...)
		if !resp.Diagnostics.HasError() {
			for _, owner := range owners {
				queryValues.Add("owner", owner)
			}
		}
	}
	if !data.Labels.IsNull() {
		var labels []string
		resp.Diagnostics.Append(data.Labels.ElementsAs(ctx, &labels, false)...)
		if !resp.Diagnostics.HasError() {
			for _, label := range labels {
				queryValues.Add("label", label)
			}
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// API returns an array of SingleApplicationResponse directly, not wrapped in an object
	var apiApplications []SingleApplicationResponse
	response, err := d.ProviderData.Client.R().
		SetContext(ctx).
		SetQueryParamsFromValues(queryValues).
		SetResult(&apiApplications).
		Get(applicationsEndpoint)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Data Source",
			"An unexpected error occurred while fetching the data source. "+
				"Please report this issue to the provider developers.\n\n"+
				"Error: "+err.Error(),
		)
		return
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			// No applications found, return empty list
			apiApplications = []SingleApplicationResponse{}
		} else {
			resp.Diagnostics.AddError(
				"Unable to Read Data Source",
				"An unexpected error occurred while fetching the data source. "+
					"Please report this issue to the provider developers.\n\n"+
					"Error: "+response.String(),
			)
			return
		}
	}

	// Convert API response (array of SingleApplicationResponse) to ApplicationsListAPIModel
	// Note: The API doesn't return pagination metadata, so we calculate total from array length
	// and use the requested limit/offset values
	limit := 0
	if !data.Limit.IsNull() {
		limit = int(data.Limit.ValueInt64())
	}
	offset := 0
	if !data.Offset.IsNull() {
		offset = int(data.Offset.ValueInt64())
	}

	result := ApplicationsListAPIModel{
		Applications: make([]ApplicationListItemAPIModel, len(apiApplications)),
		Total:        len(apiApplications),
		Limit:        limit,
		Offset:       offset,
	}

	// Convert SingleApplicationResponse to ApplicationListItemAPIModel
	// Note: API response doesn't include version info in list endpoint
	for i, app := range apiApplications {
		result.Applications[i] = ApplicationListItemAPIModel{
			ProjectKey:      app.ProjectKey,
			ApplicationKey:  app.ApplicationKey,
			ApplicationName: app.ApplicationName,
			// These fields are not returned by the list endpoint, set to empty/default values
			ApplicationVersionLatest: "",
			ApplicationVersionTag:    "",
			ApplicationVersionsCount: 0,
		}
	}

	diags := data.FromAPIModel(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (m *ApplicationsDataSourceModel) FromAPIModel(ctx context.Context, data ApplicationsListAPIModel) diag.Diagnostics {
	var diags diag.Diagnostics

	m.Total = types.Int64Value(int64(data.Total))

	var applications []attr.Value
	for _, app := range data.Applications {
		appObj := types.ObjectValueMust(
			applicationListItemAttrType,
			map[string]attr.Value{
				"project_key":                types.StringValue(app.ProjectKey),
				"application_name":           types.StringValue(app.ApplicationName),
				"application_key":            types.StringValue(app.ApplicationKey),
				"application_version_latest": types.StringValue(app.ApplicationVersionLatest),
				"application_version_tag":    types.StringValue(app.ApplicationVersionTag),
				"application_versions_count": types.Int64Value(int64(app.ApplicationVersionsCount)),
			},
		)
		applications = append(applications, appObj)
	}

	applicationsList, d := types.ListValue(
		types.ObjectType{AttrTypes: applicationListItemAttrType},
		applications,
	)
	if d != nil {
		diags.Append(d...)
		return diags
	}

	m.Applications = applicationsList
	return diags
}
