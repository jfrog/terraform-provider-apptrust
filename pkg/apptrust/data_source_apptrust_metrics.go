package apptrust

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
)

const (
	MetricsEndpoint = "https://{jfrog_url}/apptrust/api/v1/metrics"
	MetricsEndpoint  = "https://{jfrog_url}/apptrust/api/v1/metrics"
)

func NewApptrustMetricsDataSource() datasource.DataSource {
	return &ApptrustMetricsDataSource{
		TypeName: "apptrust_metrics",
	}
}

type ApptrustMetricsDataSource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type ApptrustMetricsDataSourceModel struct {
	JfrogUrl types.String `tfsdk:"jfrog_url"`
}

type MetricsAPIModel struct {
	JfrogUrl string `json:"jfrog_url"`
}

func (d *ApptrustMetricsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = d.TypeName
}

func (d *ApptrustMetricsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"jfrog_url": schema.StringAttribute{
				Required: true,
				Description: "The jfrog_url of the resource.",
			},
		},
		MarkdownDescription: "Fetches metrics from JFrog Apptrust.",
	}
}

func (d *ApptrustMetricsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (d *ApptrustMetricsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, d.ProviderData.Client.R(), d.ProviderData.ProductId, d.TypeName)

	var state ApptrustMetricsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result ApptrustMetricsAPIModel

	response, err := d.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"jfrog_url": state.JfrogUrl.ValueString(),
		}).
		SetResult(&result).
		Get(MetricsEndpoint)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read apptrust_metrics", err.Error())
		return
	}

	if response.IsError() {
		resp.Diagnostics.AddError("Unable to read apptrust_metrics", response.String())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
