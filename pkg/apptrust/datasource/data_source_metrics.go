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

const metricsEndpoint = "apptrust/api/v1/metrics"

var _ datasource.DataSource = &MetricsDataSource{}

func NewMetricsDataSource() datasource.DataSource {
	return &MetricsDataSource{}
}

type MetricsDataSource struct {
	ProviderData util.ProviderMetadata
}

type MetricsDataSourceModel struct {
	Metrics types.String `tfsdk:"metrics"`
}

func (d *MetricsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_metrics"
}

func (d *MetricsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Collects the AppTrust service metrics (GET /v1/metrics). The metrics are returned in Prometheus exposition format as a single string.",
		Attributes: map[string]schema.Attribute{
			"metrics": schema.StringAttribute{
				Description: "Raw service metrics payload in Prometheus exposition format.",
				Computed:    true,
			},
		},
	}
}

func (d *MetricsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (d *MetricsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data MetricsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading AppTrust metrics")

	httpResponse, err := d.ProviderData.Client.R().
		SetContext(ctx).
		Get(metricsEndpoint)

	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Data Source", "Error: "+err.Error())
		return
	}

	if httpResponse.StatusCode() != http.StatusOK {
		diags := apptrust.HandleAPIErrorWithType(httpResponse, "read", "metrics")
		resp.Diagnostics.Append(diags...)
		return
	}

	data.Metrics = types.StringValue(httpResponse.String())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
