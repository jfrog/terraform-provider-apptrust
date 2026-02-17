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
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jfrog/terraform-provider-apptrust/pkg/apptrust"
	"github.com/jfrog/terraform-provider-apptrust/pkg/apptrust/resource"
	"github.com/jfrog/terraform-provider-shared/util"
)

var _ datasource.DataSource = &ApplicationVersionStatusDataSource{}

func NewApplicationVersionStatusDataSource() datasource.DataSource {
	return &ApplicationVersionStatusDataSource{}
}

type ApplicationVersionStatusDataSource struct {
	ProviderData util.ProviderMetadata
}

type ApplicationVersionStatusDataSourceModel struct {
	ApplicationKey       types.String `tfsdk:"application_key"`
	Version              types.String `tfsdk:"version"`
	VersionReleaseStatus types.String `tfsdk:"version_release_status"`
}

type versionStatusResponseAPIModel struct {
	VersionReleaseStatus string `json:"version_release_status"`
}

func (d *ApplicationVersionStatusDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_version_status"
}

func (d *ApplicationVersionStatusDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Returns the release status for a specific application version (GET /v1/applications/{application_key}/versions/{version}/status).",
		Attributes: map[string]schema.Attribute{
			"application_key": schema.StringAttribute{
				Description: "The application key.",
				Required:    true,
			},
			"version": schema.StringAttribute{
				Description: "The application version.",
				Required:    true,
			},
			"version_release_status": schema.StringAttribute{
				Description: "Release status: pre_release, released, or trusted_release.",
				Computed:    true,
			},
		},
	}
}

func (d *ApplicationVersionStatusDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (d *ApplicationVersionStatusDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ApplicationVersionStatusDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationKey := data.ApplicationKey.ValueString()
	version := data.Version.ValueString()
	tflog.Info(ctx, "Reading application version status", map[string]interface{}{
		"application_key": applicationKey,
		"version":         version,
	})

	var apiResp versionStatusResponseAPIModel
	httpResponse, err := d.ProviderData.Client.R().
		SetContext(ctx).
		SetPathParam("application_key", applicationKey).
		SetPathParam("version", version).
		SetResult(&apiResp).
		Get(resource.ApplicationVersionStatusEP)

	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Data Source", "Error: "+err.Error())
		return
	}

	if httpResponse.StatusCode() != http.StatusOK {
		if httpResponse.StatusCode() == http.StatusNotFound {
			data.VersionReleaseStatus = types.StringNull()
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		diags := apptrust.HandleAPIErrorWithType(httpResponse, "read", "application version status")
		resp.Diagnostics.Append(diags...)
		return
	}

	data.VersionReleaseStatus = types.StringValue(apiResp.VersionReleaseStatus)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
