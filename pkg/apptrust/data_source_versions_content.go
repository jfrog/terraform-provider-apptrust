package apptrust

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/jfrog/terraform-provider-shared/util"
)

const (
	ContentEndpoint = "https://{jfrog_url}/apptrust/api/v1/applications/{application_key}/versions/{version}/content"
	ContentEndpoint  = "https://{jfrog_url}/apptrust/api/v1/applications/{application_key}/versions/{version}/content"
)

func NewVersionsContentDataSource() datasource.DataSource {
	return &VersionsContentDataSource{
		TypeName: "apptrust_content",
	}
}

type VersionsContentDataSource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type VersionsContentDataSourceModel struct {
	JfrogUrl types.String `tfsdk:"jfrog_url"`
	ApplicationKey types.String `tfsdk:"application_key"`
	Version types.String `tfsdk:"version"`
	Include types.String `tfsdk:"include"`
	PackageTypes types.String `tfsdk:"package_types"`
	SourceBuilds types.String `tfsdk:"source_builds"`
	SourceReleaseBundles types.String `tfsdk:"source_release_bundles"`
	SourceApplicationVersions types.String `tfsdk:"source_application_versions"`
	Limit types.Int64 `tfsdk:"limit"`
	Offset types.Int64 `tfsdk:"offset"`
}

type ContentAPIModel struct {
	JfrogUrl string `json:"jfrog_url"`
	ApplicationKey string `json:"application_key"`
	Version string `json:"version"`
	Include string `json:"include"`
	PackageTypes string `json:"package_types"`
	SourceBuilds string `json:"source_builds"`
	SourceReleaseBundles string `json:"source_release_bundles"`
	SourceApplicationVersions string `json:"source_application_versions"`
	Limit string `json:"limit"`
	Offset string `json:"offset"`
}

func (d *VersionsContentDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = d.TypeName
}

func (d *VersionsContentDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
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
			"include": schema.StringAttribute{
				Computed: true,
				Description: "The level of detail in the response. Can be one of the following:<br>- sources: The source of the content (application version, release bundle, etc.).<br>- releasables: The package or standalone artifact to be released as part of the version.<br>- releasables_expanded: Includes expanded details about the releasables.",
			},
			"package_types": schema.StringAttribute{
				Computed: true,
				Description: "Filter by package types (comma-separated)",
			},
			"source_builds": schema.StringAttribute{
				Computed: true,
				Description: "Filter by source builds (comma-separated)",
			},
			"source_release_bundles": schema.StringAttribute{
				Computed: true,
				Description: "Filter by source release bundles (comma-separated)",
			},
			"source_application_versions": schema.StringAttribute{
				Computed: true,
				Description: "Filter by source application versions (comma-separated)",
			},
			"limit": schema.StringAttribute{
				Computed: true,
				Description: "The maximum number of records to return (up to 250).<br>The default is 25.",
			},
			"offset": schema.StringAttribute{
				Computed: true,
				Description: "The number of records to skip for pagination.<br>The default is 0.",
			},
		},
		MarkdownDescription: "Fetches content from JFrog Apptrust.",
	}
}

func (d *VersionsContentDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (d *VersionsContentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, d.ProviderData.Client.R(), d.ProviderData.ProductId, d.TypeName)

	var state VersionsContentDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result VersionsContentAPIModel

	response, err := d.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"jfrog_url": state.JfrogUrl.ValueString(),
			"application_key": state.ApplicationKey.ValueString(),
			"version": state.Version.ValueString(),
		}).
		SetResult(&result).
		Get(ContentEndpoint)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read apptrust_content", err.Error())
		return
	}

	if response.IsError() {
		resp.Diagnostics.AddError("Unable to read apptrust_content", response.String())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
