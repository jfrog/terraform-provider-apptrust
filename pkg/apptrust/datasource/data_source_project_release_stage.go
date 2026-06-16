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
	"github.com/jfrog/terraform-provider-shared/util"
)

const projectReleaseStageEndpoint = "apptrust/api/v1/projects/{project_key}/stages/release"

var _ datasource.DataSource = &ProjectReleaseStageDataSource{}

func NewProjectReleaseStageDataSource() datasource.DataSource {
	return &ProjectReleaseStageDataSource{}
}

type ProjectReleaseStageDataSource struct {
	ProviderData util.ProviderMetadata
}

type ProjectReleaseStageDataSourceModel struct {
	ProjectKey types.String `tfsdk:"project_key"`
	StageName  types.String `tfsdk:"stage_name"`
	Gates      types.List   `tfsdk:"gates"`
}

type projectReleaseStageAPIModel struct {
	StageName string   `json:"stage_name"`
	Gates     []string `json:"gates"`
}

func (d *ProjectReleaseStageDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_release_stage"
}

func (d *ProjectReleaseStageDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Returns the release stage configured for a project (GET /v1/projects/{project_key}/stages/release).",
		Attributes: map[string]schema.Attribute{
			"project_key": schema.StringAttribute{
				Description: "The project key.",
				Required:    true,
			},
			"stage_name": schema.StringAttribute{
				Description: "The name of the project's release stage.",
				Computed:    true,
			},
			"gates": schema.ListAttribute{
				Description: "Gate names configured for the release stage.",
				ElementType: types.StringType,
				Computed:    true,
			},
		},
	}
}

func (d *ProjectReleaseStageDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (d *ProjectReleaseStageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ProjectReleaseStageDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	projectKey := data.ProjectKey.ValueString()
	tflog.Info(ctx, "Reading project release stage", map[string]interface{}{
		"project_key": projectKey,
	})

	var apiResp projectReleaseStageAPIModel
	httpResponse, err := d.ProviderData.Client.R().
		SetContext(ctx).
		SetPathParam("project_key", projectKey).
		SetResult(&apiResp).
		Get(projectReleaseStageEndpoint)

	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Data Source", "Error: "+err.Error())
		return
	}

	if httpResponse.StatusCode() != http.StatusOK {
		diags := apptrust.HandleAPIErrorWithType(httpResponse, "read", "project release stage")
		resp.Diagnostics.Append(diags...)
		return
	}

	data.StageName = types.StringValue(apiResp.StageName)

	gates, diags := types.ListValueFrom(ctx, types.StringType, apiResp.Gates)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Gates = gates

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
