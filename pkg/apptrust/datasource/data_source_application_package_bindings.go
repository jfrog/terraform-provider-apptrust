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

var _ datasource.DataSource = &ApplicationPackageBindingsDataSource{}

func NewApplicationPackageBindingsDataSource() datasource.DataSource {
	return &ApplicationPackageBindingsDataSource{}
}

type ApplicationPackageBindingsDataSource struct {
	ProviderData util.ProviderMetadata
}

type ApplicationPackageBindingsDataSourceModel struct {
	ApplicationKey types.String `tfsdk:"application_key"`
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	Offset         types.Int64  `tfsdk:"offset"`
	Limit          types.Int64  `tfsdk:"limit"`
	Packages       types.List   `tfsdk:"packages"`
	Pagination     types.Object `tfsdk:"pagination"`
}

type packageBindingAPIModel struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	NumVersions   int    `json:"num_versions"`
	LatestVersion string `json:"latest_version"`
}

type packageBindingsResponseAPIModel struct {
	Packages   []packageBindingAPIModel `json:"packages"`
	Pagination *struct {
		Offset     int `json:"offset"`
		Limit      int `json:"limit"`
		TotalItems int `json:"total_items"`
	} `json:"pagination,omitempty"`
}

var packageBindingAttrType = map[string]attr.Type{
	"name":           types.StringType,
	"type":           types.StringType,
	"num_versions":   types.Int64Type,
	"latest_version": types.StringType,
}

var paginationAttrType = map[string]attr.Type{
	"offset":      types.Int64Type,
	"limit":       types.Int64Type,
	"total_items": types.Int64Type,
}

func (d *ApplicationPackageBindingsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_package_bindings"
}

func (d *ApplicationPackageBindingsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Returns a list of packages bound to the specified application.",
		Attributes: map[string]schema.Attribute{
			"application_key": schema.StringAttribute{
				Description: "The application key.",
				Required:    true,
			},
			"name": schema.StringAttribute{
				Description: "Filter by package name.",
				Optional:    true,
			},
			"type": schema.StringAttribute{
				Description: "Filter by package type (e.g. maven, docker, npm).",
				Optional:    true,
			},
			"offset": schema.Int64Attribute{
				Description: "Pagination offset.",
				Optional:    true,
			},
			"limit": schema.Int64Attribute{
				Description: "Pagination limit.",
				Optional:    true,
			},
			"packages": schema.ListNestedAttribute{
				Description: "List of bound packages.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":           schema.StringAttribute{Description: "Package name.", Computed: true},
						"type":           schema.StringAttribute{Description: "Package type.", Computed: true},
						"num_versions":   schema.Int64Attribute{Description: "Number of versions bound.", Computed: true},
						"latest_version": schema.StringAttribute{Description: "Latest version.", Computed: true},
					},
				},
			},
			"pagination": schema.SingleNestedAttribute{
				Description: "Pagination info.",
				Computed:    true,
				Attributes: map[string]schema.Attribute{
					"offset":      schema.Int64Attribute{Description: "Offset used.", Computed: true},
					"limit":       schema.Int64Attribute{Description: "Limit used.", Computed: true},
					"total_items": schema.Int64Attribute{Description: "Total items.", Computed: true},
				},
			},
		},
	}
}

func (d *ApplicationPackageBindingsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (d *ApplicationPackageBindingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ApplicationPackageBindingsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationKey := data.ApplicationKey.ValueString()
	tflog.Info(ctx, "Reading application package bindings", map[string]interface{}{"application_key": applicationKey})

	request := d.ProviderData.Client.R().
		SetContext(ctx).
		SetPathParam("application_key", applicationKey)
	if !data.Name.IsNull() {
		request = request.SetQueryParam("name", data.Name.ValueString())
	}
	if !data.Type.IsNull() {
		request = request.SetQueryParam("type", data.Type.ValueString())
	}
	if !data.Offset.IsNull() {
		request = request.SetQueryParam("offset", fmt.Sprintf("%d", data.Offset.ValueInt64()))
	}
	if !data.Limit.IsNull() {
		request = request.SetQueryParam("limit", fmt.Sprintf("%d", data.Limit.ValueInt64()))
	}

	var apiResp packageBindingsResponseAPIModel
	httpResponse, err := request.SetResult(&apiResp).Get(resource.ApplicationPackagesEndpoint)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Data Source", "Error: "+err.Error())
		return
	}

	if httpResponse.StatusCode() != http.StatusOK {
		if httpResponse.StatusCode() == http.StatusNotFound {
			data.Packages = types.ListNull(types.ObjectType{AttrTypes: packageBindingAttrType})
			data.Pagination = types.ObjectNull(paginationAttrType)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		diags := apptrust.HandleAPIErrorWithType(httpResponse, "read", "application package bindings")
		resp.Diagnostics.Append(diags...)
		return
	}

	diags := data.fromAPIModel(ctx, apiResp)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (m *ApplicationPackageBindingsDataSourceModel) fromAPIModel(ctx context.Context, api packageBindingsResponseAPIModel) diag.Diagnostics {
	var diags diag.Diagnostics
	var items []attr.Value
	for _, p := range api.Packages {
		items = append(items, types.ObjectValueMust(packageBindingAttrType, map[string]attr.Value{
			"name":           types.StringValue(p.Name),
			"type":           types.StringValue(p.Type),
			"num_versions":   types.Int64Value(int64(p.NumVersions)),
			"latest_version": types.StringValue(p.LatestVersion),
		}))
	}
	list, d := types.ListValue(types.ObjectType{AttrTypes: packageBindingAttrType}, items)
	if d != nil {
		diags.Append(d...)
		return diags
	}
	m.Packages = list
	offset, limit, total := 0, 0, len(api.Packages)
	if api.Pagination != nil {
		offset, limit, total = api.Pagination.Offset, api.Pagination.Limit, api.Pagination.TotalItems
	}
	m.Pagination = types.ObjectValueMust(paginationAttrType, map[string]attr.Value{
		"offset":      types.Int64Value(int64(offset)),
		"limit":       types.Int64Value(int64(limit)),
		"total_items": types.Int64Value(int64(total)),
	})
	return diags
}
