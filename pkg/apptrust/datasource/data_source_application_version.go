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

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jfrog/terraform-provider-apptrust/pkg/apptrust"
	"github.com/jfrog/terraform-provider-apptrust/pkg/apptrust/resource"
	"github.com/jfrog/terraform-provider-shared/util"
)

var _ datasource.DataSource = &ApplicationVersionDataSource{}

func NewApplicationVersionDataSource() datasource.DataSource {
	return &ApplicationVersionDataSource{}
}

type ApplicationVersionDataSource struct {
	ProviderData util.ProviderMetadata
}

type ApplicationVersionDataSourceModel struct {
	ApplicationKey types.String `tfsdk:"application_key"`
	Version        types.String `tfsdk:"version"`
	Tag            types.String `tfsdk:"tag"`
	Status         types.String `tfsdk:"status"`
	ReleaseStatus  types.String `tfsdk:"release_status"`
	CurrentStage   types.String `tfsdk:"current_stage"`
	CreatedBy      types.String `tfsdk:"created_by"`
	Created        types.String `tfsdk:"created"`
}

func (d *ApplicationVersionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_version"
}

func (d *ApplicationVersionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Returns details of a specific application version (GET /v1/applications/{application_key}/versions).",
		Attributes: map[string]schema.Attribute{
			"application_key": schema.StringAttribute{
				Description: "The application key.",
				Required:    true,
			},
			"version": schema.StringAttribute{
				Description: "The version identifier to look up.",
				Required:    true,
			},
			"tag": schema.StringAttribute{
				Description: "Tag associated with the version.",
				Computed:    true,
			},
			"status": schema.StringAttribute{
				Description: "Processing status of the version.",
				Computed:    true,
			},
			"release_status": schema.StringAttribute{
				Description: "Release status: pre_release, released, trusted_release.",
				Computed:    true,
			},
			"current_stage": schema.StringAttribute{
				Description: "Current lifecycle stage.",
				Computed:    true,
			},
			"created_by": schema.StringAttribute{
				Description: "The user who created the version.",
				Computed:    true,
			},
			"created": schema.StringAttribute{
				Description: "Timestamp when the version was created.",
				Computed:    true,
			},
		},
	}
}

func (d *ApplicationVersionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (d *ApplicationVersionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ApplicationVersionDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationKey := data.ApplicationKey.ValueString()
	version := data.Version.ValueString()
	tflog.Info(ctx, "Reading application version", map[string]interface{}{
		"application_key": applicationKey,
		"version":         version,
	})

	var listResp struct {
		Versions []struct {
			Version       string `json:"version"`
			Tag           string `json:"tag"`
			Status        string `json:"status"`
			ReleaseStatus string `json:"release_status"`
			CurrentStage  string `json:"current_stage"`
			CreatedBy     string `json:"created_by"`
			Created       string `json:"created"`
		} `json:"versions"`
	}

	httpResponse, err := d.ProviderData.Client.R().
		SetContext(ctx).
		SetPathParam("application_key", applicationKey).
		SetQueryParam("limit", "1000").
		SetResult(&listResp).
		Get(resource.ApplicationVersionsEndpoint)

	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Data Source", "Error: "+err.Error())
		return
	}

	if httpResponse.StatusCode() != http.StatusOK {
		if httpResponse.StatusCode() == http.StatusNotFound {
			resp.Diagnostics.AddError(
				"Application Not Found",
				fmt.Sprintf("Application with key '%s' was not found.", applicationKey),
			)
			return
		}
		diags := apptrust.HandleAPIErrorWithType(httpResponse, "read", "application version")
		resp.Diagnostics.Append(diags...)
		return
	}

	for _, v := range listResp.Versions {
		if v.Version == version {
			data.Tag = types.StringValue(v.Tag)
			data.Status = types.StringValue(v.Status)
			data.ReleaseStatus = types.StringValue(v.ReleaseStatus)
			data.CurrentStage = types.StringValue(v.CurrentStage)
			data.CreatedBy = types.StringValue(v.CreatedBy)
			data.Created = types.StringValue(v.Created)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
	}

	resp.Diagnostics.AddError(
		"Application Version Not Found",
		fmt.Sprintf("Version '%s' for application '%s' was not found.", version, applicationKey),
	)
}
