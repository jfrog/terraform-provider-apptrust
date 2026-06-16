package apptrust

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
)

const (
	PromotionsEndpoint = "https://{jfrog_url}/apptrust/api/v1/applications/{application_key}/versions/{version}/promotions/{created_millis}"
	PromotionsEndpoint  = "https://{jfrog_url}/apptrust/api/v1/applications/{application_key}/versions/{version}/promotions/{created_millis}"
)

func NewVersionsPromotionsDataSource() datasource.DataSource {
	return &VersionsPromotionsDataSource{
		TypeName: "apptrust_promotions",
	}
}

type VersionsPromotionsDataSource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type VersionsPromotionsDataSourceModel struct {
	JfrogUrl types.String `tfsdk:"jfrog_url"`
	ApplicationKey types.String `tfsdk:"application_key"`
	Version types.String `tfsdk:"version"`
	CreatedMillis types.String `tfsdk:"created_millis"`
}

type PromotionsAPIModel struct {
	JfrogUrl string `json:"jfrog_url"`
	ApplicationKey string `json:"application_key"`
	Version string `json:"version"`
	CreatedMillis string `json:"created_millis"`
}

func (d *VersionsPromotionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = d.TypeName
}

func (d *VersionsPromotionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"jfrog_url": schema.StringAttribute{
				Required: true,
				Description: "The jfrog_url of the resource.",
			},
			"application_key": schema.StringAttribute{
				Required: true,
				Description: "The application_key of the resource.",
			},
			"version": schema.StringAttribute{
				Required: true,
				Description: "The version of the resource.",
			},
			"created_millis": schema.StringAttribute{
				Required: true,
				Description: "The created_millis of the resource.",
			},
		},
		MarkdownDescription: "Fetches promotions from JFrog Apptrust.",
	}
}

func (d *VersionsPromotionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (d *VersionsPromotionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, d.ProviderData.Client.R(), d.ProviderData.ProductId, d.TypeName)

	var state VersionsPromotionsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result VersionsPromotionsAPIModel

	response, err := d.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"jfrog_url": state.JfrogUrl.ValueString(),
			"application_key": state.ApplicationKey.ValueString(),
			"version": state.Version.ValueString(),
			"created_millis": state.CreatedMillis.ValueString(),
		}).
		SetResult(&result).
		Get(PromotionsEndpoint)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read apptrust_promotions", err.Error())
		return
	}

	if response.IsError() {
		resp.Diagnostics.AddError("Unable to read apptrust_promotions", response.String())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
