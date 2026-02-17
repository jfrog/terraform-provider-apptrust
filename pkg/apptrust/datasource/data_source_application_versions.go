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

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jfrog/terraform-provider-apptrust/pkg/apptrust"
	"github.com/jfrog/terraform-provider-apptrust/pkg/apptrust/resource"
	"github.com/jfrog/terraform-provider-shared/util"
)

var _ datasource.DataSource = &ApplicationVersionsDataSource{}

func NewApplicationVersionsDataSource() datasource.DataSource {
	return &ApplicationVersionsDataSource{}
}

type ApplicationVersionsDataSource struct {
	ProviderData util.ProviderMetadata
}

type ApplicationVersionsDataSourceModel struct {
	ApplicationKey types.String `tfsdk:"application_key"`
	CreatedBy      types.String `tfsdk:"created_by"`
	ReleaseStatus  types.String `tfsdk:"release_status"`
	Tag            types.String `tfsdk:"tag"`
	Offset         types.Int64  `tfsdk:"offset"`
	Limit          types.Int64  `tfsdk:"limit"`
	OrderAsc       types.Bool   `tfsdk:"order_asc"`
	Versions       types.List   `tfsdk:"versions"`
	Total          types.Int64  `tfsdk:"total"`
}

type applicationVersionItemAPIModel struct {
	Version       string `json:"version"`
	Tag           string `json:"tag"`
	Status        string `json:"status"`
	ReleaseStatus string `json:"release_status"`
	CurrentStage  string `json:"current_stage"`
	CreatedBy     string `json:"created_by"`
	Created       string `json:"created"`
}

type applicationVersionsListAPIModel struct {
	Versions []applicationVersionItemAPIModel `json:"versions"`
	Total    int                              `json:"total"`
	Limit    int                              `json:"limit"`
	Offset   int                              `json:"offset"`
}

var applicationVersionItemAttrType = map[string]attr.Type{
	"version":        types.StringType,
	"tag":            types.StringType,
	"status":         types.StringType,
	"release_status": types.StringType,
	"current_stage":  types.StringType,
	"created_by":     types.StringType,
	"created":        types.StringType,
}

func (d *ApplicationVersionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_versions"
}

func (d *ApplicationVersionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Returns a list of application versions for the specified application.",
		Attributes: map[string]schema.Attribute{
			"application_key": schema.StringAttribute{
				Description: "The application key.",
				Required:    true,
			},
			"created_by": schema.StringAttribute{
				Description: "Filter by the user who created the application version.",
				Optional:    true,
			},
			"release_status": schema.StringAttribute{
				Description: "Filter by release status: released, pre_release, trusted_release. Comma-separated for multiple.",
				Optional:    true,
			},
			"tag": schema.StringAttribute{
				Description: "Filter by tag. Supports trailing wildcard (*) and comma-separated values.",
				Optional:    true,
			},
			"offset": schema.Int64Attribute{
				Description: "Number of records to skip (pagination).",
				Optional:    true,
			},
			"limit": schema.Int64Attribute{
				Description: "Maximum number of versions to return.",
				Optional:    true,
			},
			"order_asc": schema.BoolAttribute{
				Description: "Order ascending (true) or descending (false). Default false.",
				Optional:    true,
			},
			"versions": schema.ListNestedAttribute{
				Description: "List of application versions.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"version":        schema.StringAttribute{Description: "Version identifier.", Computed: true},
						"tag":            schema.StringAttribute{Description: "Tag.", Computed: true},
						"status":         schema.StringAttribute{Description: "Status.", Computed: true},
						"release_status": schema.StringAttribute{Description: "Release status.", Computed: true},
						"current_stage":  schema.StringAttribute{Description: "Current stage.", Computed: true},
						"created_by":     schema.StringAttribute{Description: "Created by.", Computed: true},
						"created":        schema.StringAttribute{Description: "Created timestamp.", Computed: true},
					},
				},
			},
			"total": schema.Int64Attribute{
				Description: "Total number of versions.",
				Computed:    true,
			},
		},
	}
}

func (d *ApplicationVersionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (d *ApplicationVersionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ApplicationVersionsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationKey := data.ApplicationKey.ValueString()
	tflog.Info(ctx, "Reading application versions", map[string]interface{}{"application_key": applicationKey})

	request := d.ProviderData.Client.R().
		SetContext(ctx).
		SetPathParam("application_key", applicationKey)

	if !data.CreatedBy.IsNull() {
		request = request.SetQueryParam("created_by", data.CreatedBy.ValueString())
	}
	if !data.ReleaseStatus.IsNull() {
		request = request.SetQueryParam("release_status", data.ReleaseStatus.ValueString())
	}
	if !data.Tag.IsNull() {
		request = request.SetQueryParam("tag", data.Tag.ValueString())
	}
	if !data.Offset.IsNull() {
		request = request.SetQueryParam("offset", fmt.Sprintf("%d", data.Offset.ValueInt64()))
	}
	if !data.Limit.IsNull() {
		request = request.SetQueryParam("limit", fmt.Sprintf("%d", data.Limit.ValueInt64()))
	}
	if !data.OrderAsc.IsNull() {
		request = request.SetQueryParam("order_asc", fmt.Sprintf("%t", data.OrderAsc.ValueBool()))
	}

	var listResp applicationVersionsListAPIModel
	httpResponse, err := request.SetResult(&listResp).Get(resource.ApplicationVersionsEndpoint)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Data Source", "Error: "+err.Error())
		return
	}

	if httpResponse.StatusCode() != http.StatusOK {
		if httpResponse.StatusCode() == http.StatusNotFound {
			data.Versions = types.ListNull(types.ObjectType{AttrTypes: applicationVersionItemAttrType})
			data.Total = types.Int64Value(0)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		diags := apptrust.HandleAPIErrorWithType(httpResponse, "read", "application versions")
		resp.Diagnostics.Append(diags...)
		return
	}

	diags := data.fromAPIModel(ctx, listResp)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (m *ApplicationVersionsDataSourceModel) fromAPIModel(ctx context.Context, api applicationVersionsListAPIModel) diag.Diagnostics {
	var diags diag.Diagnostics
	m.Total = types.Int64Value(int64(api.Total))

	var items []attr.Value
	for _, v := range api.Versions {
		obj := types.ObjectValueMust(
			applicationVersionItemAttrType,
			map[string]attr.Value{
				"version":        types.StringValue(v.Version),
				"tag":            types.StringValue(v.Tag),
				"status":         types.StringValue(v.Status),
				"release_status": types.StringValue(v.ReleaseStatus),
				"current_stage":  types.StringValue(v.CurrentStage),
				"created_by":     types.StringValue(v.CreatedBy),
				"created":        types.StringValue(v.Created),
			},
		)
		items = append(items, obj)
	}

	list, d := types.ListValue(types.ObjectType{AttrTypes: applicationVersionItemAttrType}, items)
	if d != nil {
		diags.Append(d...)
		return diags
	}
	m.Versions = list
	return diags
}
