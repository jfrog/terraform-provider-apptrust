package apptrust

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
)

const (
	ServiceIdEndpoint = "https://{jfrog_url}/apptrust/api/v1/system/service_id"
	ServiceIdEndpoint  = "https://{jfrog_url}/apptrust/api/v1/system/service_id"
)

func NewSystemServiceIdDataSource() datasource.DataSource {
	return &SystemServiceIdDataSource{
		TypeName: "apptrust_service_id",
	}
}

type SystemServiceIdDataSource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type SystemServiceIdDataSourceModel struct {
	JfrogUrl types.String `tfsdk:"jfrog_url"`
}

type ServiceIdAPIModel struct {
	JfrogUrl string `json:"jfrog_url"`
}

func (d *SystemServiceIdDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = d.TypeName
}

func (d *SystemServiceIdDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"jfrog_url": schema.StringAttribute{
				Required: true,
				Description: "The jfrog_url of the resource.",
			},
		},
		MarkdownDescription: "Fetches service_id from JFrog Apptrust.",
	}
}

func (d *SystemServiceIdDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (d *SystemServiceIdDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, d.ProviderData.Client.R(), d.ProviderData.ProductId, d.TypeName)

	var state SystemServiceIdDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result SystemServiceIdAPIModel

	response, err := d.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"jfrog_url": state.JfrogUrl.ValueString(),
		}).
		SetResult(&result).
		Get(ServiceIdEndpoint)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read apptrust_service_id", err.Error())
		return
	}

	if response.IsError() {
		resp.Diagnostics.AddError("Unable to read apptrust_service_id", response.String())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
