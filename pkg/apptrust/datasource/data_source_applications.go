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

// singleApplicationResponse matches the API response for each application in GET /v1/applications.
// Labels are returned as an array of key/value objects (not a map).
type singleApplicationResponse struct {
	ApplicationKey  string           `json:"application_key"`
	ApplicationName string           `json:"application_name"`
	ProjectKey      string           `json:"project_key"`
	Description     string           `json:"description,omitempty"`
	MaturityLevel   string           `json:"maturity_level,omitempty"`
	Criticality     string           `json:"criticality,omitempty"`
	Labels          []LabelAPIModel  `json:"labels,omitempty"`
	UserOwners      []string         `json:"user_owners,omitempty"`
	GroupOwners     []string         `json:"group_owners,omitempty"`
}

// paginatedApplicationsAPIResponse matches the paginated wrapper returned by GET /v1/applications.
type paginatedApplicationsAPIResponse struct {
	Applications []singleApplicationResponse `json:"applications"`
	Total        int                         `json:"total"`
	Limit        int                         `json:"limit"`
	Offset       int                         `json:"offset"`
}

type ApplicationListItemAPIModel struct {
	ProjectKey      string          `json:"project_key"`
	ApplicationName string          `json:"application_name"`
	ApplicationKey  string          `json:"application_key"`
	Description     string          `json:"description,omitempty"`
	MaturityLevel   string          `json:"maturity_level,omitempty"`
	Criticality     string          `json:"criticality,omitempty"`
	Labels          []LabelAPIModel `json:"labels,omitempty"`
	UserOwners      []string        `json:"user_owners,omitempty"`
	GroupOwners     []string        `json:"group_owners,omitempty"`
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
	"project_key":      types.StringType,
	"application_name": types.StringType,
	"application_key":  types.StringType,
	"description":      types.StringType,
	"maturity_level":   types.StringType,
	"criticality":      types.StringType,
	"labels":           types.MapType{ElemType: types.StringType},
	"user_owners":      types.ListType{ElemType: types.StringType},
	"group_owners":     types.ListType{ElemType: types.StringType},
}

func (d *ApplicationsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_applications"
}

func (d *ApplicationsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Returns a list of AppTrust applications with their full details. " +
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
						"description": schema.StringAttribute{
							Description: "A free-text description of the application.",
							Computed:    true,
						},
						"maturity_level": schema.StringAttribute{
							Description: "The maturity level of the application.",
							Computed:    true,
						},
						"criticality": schema.StringAttribute{
							Description: "A classification of how critical the application is for your business.",
							Computed:    true,
						},
						"labels": schema.MapAttribute{
							Description: "Key-value pairs that label the application.",
							ElementType: types.StringType,
							Computed:    true,
						},
						"user_owners": schema.ListAttribute{
							Description: "List of users who own the application.",
							ElementType: types.StringType,
							Computed:    true,
						},
						"group_owners": schema.ListAttribute{
							Description: "List of user groups who own the application.",
							ElementType: types.StringType,
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

	// The API returns a paginated wrapper: {"applications": [...], "total": N, "limit": N, "offset": N}
	var apiResp paginatedApplicationsAPIResponse
	response, err := d.ProviderData.Client.R().
		SetContext(ctx).
		SetQueryParamsFromValues(queryValues).
		SetResult(&apiResp).
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
			apiResp = paginatedApplicationsAPIResponse{}
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

	// Convert API response to ApplicationsListAPIModel using actual pagination metadata from API.
	result := ApplicationsListAPIModel{
		Applications: make([]ApplicationListItemAPIModel, len(apiResp.Applications)),
		Total:        apiResp.Total,
		Limit:        apiResp.Limit,
		Offset:       apiResp.Offset,
	}

	// Convert singleApplicationResponse to ApplicationListItemAPIModel.
	for i, app := range apiResp.Applications {
		result.Applications[i] = ApplicationListItemAPIModel{
			ProjectKey:      app.ProjectKey,
			ApplicationKey:  app.ApplicationKey,
			ApplicationName: app.ApplicationName,
			Description:     app.Description,
			MaturityLevel:   app.MaturityLevel,
			Criticality:     app.Criticality,
			Labels:          app.Labels,
			UserOwners:      app.UserOwners,
			GroupOwners:     app.GroupOwners,
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
		// Convert labels slice to map
		labelsMap := types.MapValueMust(types.StringType, map[string]attr.Value{})
		if len(app.Labels) > 0 {
			labelsData := make(map[string]attr.Value, len(app.Labels))
			for _, label := range app.Labels {
				labelsData[label.Key] = types.StringValue(label.Value)
			}
			labelsMap = types.MapValueMust(types.StringType, labelsData)
		}

		// Convert user_owners
		userOwnersList := types.ListValueMust(types.StringType, []attr.Value{})
		if len(app.UserOwners) > 0 {
			userOwnerVals := make([]attr.Value, len(app.UserOwners))
			for i, v := range app.UserOwners {
				userOwnerVals[i] = types.StringValue(v)
			}
			userOwnersList = types.ListValueMust(types.StringType, userOwnerVals)
		}

		// Convert group_owners
		groupOwnersList := types.ListValueMust(types.StringType, []attr.Value{})
		if len(app.GroupOwners) > 0 {
			groupOwnerVals := make([]attr.Value, len(app.GroupOwners))
			for i, v := range app.GroupOwners {
				groupOwnerVals[i] = types.StringValue(v)
			}
			groupOwnersList = types.ListValueMust(types.StringType, groupOwnerVals)
		}

		appObj, d := types.ObjectValue(
			applicationListItemAttrType,
			map[string]attr.Value{
				"project_key":      types.StringValue(app.ProjectKey),
				"application_name": types.StringValue(app.ApplicationName),
				"application_key":  types.StringValue(app.ApplicationKey),
				"description":      types.StringValue(app.Description),
				"maturity_level":   types.StringValue(app.MaturityLevel),
				"criticality":      types.StringValue(app.Criticality),
				"labels":           labelsMap,
				"user_owners":      userOwnersList,
				"group_owners":     groupOwnersList,
			},
		)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
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
