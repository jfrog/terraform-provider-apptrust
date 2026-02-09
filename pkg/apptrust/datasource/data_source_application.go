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
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
)

const (
	applicationEndpoint = "apptrust/api/v1/applications"
)

var _ datasource.DataSource = &ApplicationDataSource{}

func NewApplicationDataSource() datasource.DataSource {
	return &ApplicationDataSource{}
}

type ApplicationDataSource struct {
	ProviderData util.ProviderMetadata
}

type ApplicationDataSourceModel struct {
	ApplicationKey  types.String `tfsdk:"application_key"`
	ApplicationName types.String `tfsdk:"application_name"`
	ProjectKey      types.String `tfsdk:"project_key"`
	Description     types.String `tfsdk:"description"`
	MaturityLevel   types.String `tfsdk:"maturity_level"`
	Criticality     types.String `tfsdk:"criticality"`
	Labels          types.Map    `tfsdk:"labels"`
	UserOwners      types.List   `tfsdk:"user_owners"`
	GroupOwners     types.List   `tfsdk:"group_owners"`
}

type ApplicationAPIModel struct {
	ApplicationKey  string            `json:"application_key"`
	ApplicationName string            `json:"application_name"`
	ProjectKey      string            `json:"project_key"`
	Description     string            `json:"description,omitempty"`
	MaturityLevel   string            `json:"maturity_level,omitempty"` // API uses "maturity_level" consistently for all operations (GET/POST/PATCH)
	Criticality     string            `json:"criticality,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	UserOwners      []string          `json:"user_owners,omitempty"`
	GroupOwners     []string          `json:"group_owners,omitempty"`
}

func (d *ApplicationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

func (d *ApplicationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Returns the details of a selected AppTrust application by key (GET /v1/applications/{application_key}), including owners, defined labels, maturity level, and criticality.",
		Attributes: map[string]schema.Attribute{
			"application_key": schema.StringAttribute{
				Description: "The application key to query.",
				Required:    true,
			},
			"application_name": schema.StringAttribute{
				Description: "The application display name.",
				Computed:    true,
			},
			"project_key": schema.StringAttribute{
				Description: "The key of the project associated with the application.",
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "A free-text description of the application.",
				Computed:    true,
			},
			"maturity_level": schema.StringAttribute{
				Description: "The maturity level of the application. Possible values: unspecified, experimental, production, end_of_life.",
				Computed:    true,
			},
			"criticality": schema.StringAttribute{
				Description: "A classification of how critical the application is for your business. Possible values: unspecified, low, medium, high, critical.",
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
	}
}

func (d *ApplicationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (d *ApplicationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ApplicationDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result ApplicationAPIModel
	response, err := d.ProviderData.Client.R().
		SetPathParam("application_key", data.ApplicationKey.ValueString()).
		SetResult(&result).
		Get(applicationEndpoint + "/{application_key}")

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Data Source",
			"An unexpected error occurred while fetching the data source. "+
				"Please report this issue to the provider developers.\n\n"+
				"Error: "+err.Error(),
		)
		return
	}

	if response.StatusCode() != http.StatusOK {
		if response.StatusCode() == http.StatusNotFound {
			resp.Diagnostics.AddError(
				"Application Not Found",
				fmt.Sprintf("Application with key '%s' was not found.", data.ApplicationKey.ValueString()),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Read Data Source",
			"An unexpected error occurred while fetching the data source. "+
				"Please report this issue to the provider developers.\n\n"+
				"Error: "+response.String(),
		)
		return
	}

	diags := data.FromAPIModel(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (m *ApplicationDataSourceModel) FromAPIModel(ctx context.Context, api ApplicationAPIModel) diag.Diagnostics {
	var diags diag.Diagnostics

	m.ApplicationKey = types.StringValue(api.ApplicationKey)
	m.ApplicationName = types.StringValue(api.ApplicationName)
	m.ProjectKey = types.StringValue(api.ProjectKey)

	if api.Description != "" {
		m.Description = types.StringValue(api.Description)
	} else {
		m.Description = types.StringNull()
	}

	// Treat "unspecified" as null since it's the default value when not explicitly set
	if api.MaturityLevel != "" && api.MaturityLevel != "unspecified" {
		m.MaturityLevel = types.StringValue(api.MaturityLevel)
	} else {
		m.MaturityLevel = types.StringNull()
	}

	// Treat "unspecified" as null since it's the default value when not explicitly set
	if api.Criticality != "" && api.Criticality != "unspecified" {
		m.Criticality = types.StringValue(api.Criticality)
	} else {
		m.Criticality = types.StringNull()
	}

	if len(api.Labels) > 0 {
		labels := make(map[string]types.String)
		for k, v := range api.Labels {
			labels[k] = types.StringValue(v)
		}
		labelsMap, d := types.MapValueFrom(ctx, types.StringType, labels)
		diags.Append(d...)
		if !diags.HasError() {
			m.Labels = labelsMap
		}
	} else {
		m.Labels = types.MapNull(types.StringType)
	}

	if len(api.UserOwners) > 0 {
		userOwners := make([]types.String, len(api.UserOwners))
		for i, v := range api.UserOwners {
			userOwners[i] = types.StringValue(v)
		}
		userOwnersList, d := types.ListValueFrom(ctx, types.StringType, userOwners)
		diags.Append(d...)
		if !diags.HasError() {
			m.UserOwners = userOwnersList
		}
	} else {
		m.UserOwners = types.ListNull(types.StringType)
	}

	if len(api.GroupOwners) > 0 {
		groupOwners := make([]types.String, len(api.GroupOwners))
		for i, v := range api.GroupOwners {
			groupOwners[i] = types.StringValue(v)
		}
		groupOwnersList, d := types.ListValueFrom(ctx, types.StringType, groupOwners)
		diags.Append(d...)
		if !diags.HasError() {
			m.GroupOwners = groupOwnersList
		}
	} else {
		m.GroupOwners = types.ListNull(types.StringType)
	}

	return diags
}
