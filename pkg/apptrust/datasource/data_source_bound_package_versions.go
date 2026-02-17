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

var _ datasource.DataSource = &BoundPackageVersionsDataSource{}

func NewBoundPackageVersionsDataSource() datasource.DataSource {
	return &BoundPackageVersionsDataSource{}
}

type BoundPackageVersionsDataSource struct {
	ProviderData util.ProviderMetadata
}

type BoundPackageVersionsDataSourceModel struct {
	ApplicationKey types.String `tfsdk:"application_key"`
	PackageType    types.String `tfsdk:"package_type"`
	PackageName    types.String `tfsdk:"package_name"`
	PackageVersion types.String `tfsdk:"package_version"`
	Offset         types.Int64  `tfsdk:"offset"`
	Limit          types.Int64  `tfsdk:"limit"`
	Versions       types.List   `tfsdk:"versions"`
	Total          types.Int64  `tfsdk:"total"`
}

type boundPackageVersionAPIModel struct {
	Version     string `json:"version"`
	VcsURL      string `json:"vcs_url"`
	VcsBranch   string `json:"vcs_branch"`
	VcsRevision string `json:"vcs_revision"`
	Branch      string `json:"branch"`
	Revision    string `json:"revision"`
}

type boundPackageVersionsResponseAPIModel struct {
	Versions []boundPackageVersionAPIModel `json:"versions"`
	Total    int                           `json:"total"`
	Offset   int                           `json:"offset"`
	Limit    int                           `json:"limit"`
}

var boundPackageVersionAttrType = map[string]attr.Type{
	"version":      types.StringType,
	"vcs_url":      types.StringType,
	"vcs_branch":   types.StringType,
	"vcs_revision": types.StringType,
}

func (d *BoundPackageVersionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bound_package_versions"
}

func (d *BoundPackageVersionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Returns bound package versions for a given application and package.",
		Attributes: map[string]schema.Attribute{
			"application_key": schema.StringAttribute{
				Description: "The application key.",
				Required:    true,
			},
			"package_type": schema.StringAttribute{
				Description: "Package type (e.g. maven, docker).",
				Required:    true,
			},
			"package_name": schema.StringAttribute{
				Description: "Package name.",
				Required:    true,
			},
			"package_version": schema.StringAttribute{
				Description: "Filter by package version. If not set, all bound versions are returned.",
				Optional:    true,
			},
			"offset": schema.Int64Attribute{
				Description: "Pagination offset. Default 0.",
				Optional:    true,
			},
			"limit": schema.Int64Attribute{
				Description: "Max versions to return (up to 250). Default 25.",
				Optional:    true,
			},
			"versions": schema.ListNestedAttribute{
				Description: "List of bound package versions.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"version":      schema.StringAttribute{Description: "Version.", Computed: true},
						"vcs_url":      schema.StringAttribute{Description: "VCS URL.", Computed: true},
						"vcs_branch":   schema.StringAttribute{Description: "VCS branch.", Computed: true},
						"vcs_revision": schema.StringAttribute{Description: "VCS revision.", Computed: true},
					},
				},
			},
			"total": schema.Int64Attribute{
				Description: "Total bound versions for this package.",
				Computed:    true,
			},
		},
	}
}

func (d *BoundPackageVersionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (d *BoundPackageVersionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BoundPackageVersionsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationKey := data.ApplicationKey.ValueString()
	pkgType := data.PackageType.ValueString()
	pkgName := data.PackageName.ValueString()
	tflog.Info(ctx, "Reading bound package versions", map[string]interface{}{
		"application_key": applicationKey, "package_type": pkgType, "package_name": pkgName,
	})

	request := d.ProviderData.Client.R().
		SetContext(ctx).
		SetPathParam("application_key", applicationKey).
		SetPathParam("type", pkgType).
		SetPathParam("name", pkgName)
	if !data.PackageVersion.IsNull() {
		request = request.SetQueryParam("package_version", data.PackageVersion.ValueString())
	}
	if !data.Offset.IsNull() {
		request = request.SetQueryParam("offset", fmt.Sprintf("%d", data.Offset.ValueInt64()))
	}
	if !data.Limit.IsNull() {
		request = request.SetQueryParam("limit", fmt.Sprintf("%d", data.Limit.ValueInt64()))
	}

	var apiResp boundPackageVersionsResponseAPIModel
	httpResponse, err := request.SetResult(&apiResp).Get(resource.ApplicationPackageVersionsEndpoint)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Data Source", "Error: "+err.Error())
		return
	}

	if httpResponse.StatusCode() != http.StatusOK {
		if httpResponse.StatusCode() == http.StatusNotFound {
			data.Versions = types.ListNull(types.ObjectType{AttrTypes: boundPackageVersionAttrType})
			data.Total = types.Int64Value(0)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		diags := apptrust.HandleAPIErrorWithType(httpResponse, "read", "bound package versions")
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

func (m *BoundPackageVersionsDataSourceModel) fromAPIModel(ctx context.Context, api boundPackageVersionsResponseAPIModel) diag.Diagnostics {
	var diags diag.Diagnostics
	m.Total = types.Int64Value(int64(api.Total))
	var items []attr.Value
	for _, v := range api.Versions {
		branch := v.VcsBranch
		if branch == "" {
			branch = v.Branch
		}
		revision := v.VcsRevision
		if revision == "" {
			revision = v.Revision
		}
		items = append(items, types.ObjectValueMust(boundPackageVersionAttrType, map[string]attr.Value{
			"version":      types.StringValue(v.Version),
			"vcs_url":      types.StringValue(v.VcsURL),
			"vcs_branch":   types.StringValue(branch),
			"vcs_revision": types.StringValue(revision),
		}))
	}
	list, d := types.ListValue(types.ObjectType{AttrTypes: boundPackageVersionAttrType}, items)
	if d != nil {
		diags.Append(d...)
		return diags
	}
	m.Versions = list
	return diags
}
