package apptrust

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
)

const (
	StagesEndpoint = "https://{jfrog_url}/apptrust/api/v1/projects/{project_key}/lifecycle/stages"
	StagesEndpoint  = "https://{jfrog_url}/apptrust/api/v1/projects/{project_key}/lifecycle/stages"
)

func NewLifecycleStagesDataSource() datasource.DataSource {
	return &LifecycleStagesDataSource{
		TypeName: "apptrust_stages",
	}
}

type LifecycleStagesDataSource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type LifecycleStagesDataSourceModel struct {
	JfrogUrl types.String `tfsdk:"jfrog_url"`
	ProjectKey types.String `tfsdk:"project_key"`
	ApplicationKey types.String `tfsdk:"application_key"`
	FilterGatesBy types.String `tfsdk:"filter_gates_by"`
}

type StagesAPIModel struct {
	JfrogUrl string `json:"jfrog_url"`
	ProjectKey string `json:"project_key"`
	ApplicationKey string `json:"application_key"`
	FilterGatesBy string `json:"filter_gates_by"`
}

func (d *LifecycleStagesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = d.TypeName
}

func (d *LifecycleStagesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"jfrog_url": schema.StringAttribute{
				Required: true,
				Description: "The jfrog_url of the resource.",
			},
			"project_key": schema.StringAttribute{
				Required: true,
				Description: "The project_key of the resource.",
			},
			"application_key": schema.StringAttribute{
				Computed: true,
				Description: "Application key to scope policies",
			},
			"filter_gates_by": schema.StringAttribute{
				Computed: true,
				Description: "Filter gates - if 'policies', only return gates with policies; otherwise, return category-based gates (promote: entry+exit, others: release)",
			},
		},
		MarkdownDescription: "Fetches stages from JFrog Apptrust.",
	}
}

func (d *LifecycleStagesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (d *LifecycleStagesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, d.ProviderData.Client.R(), d.ProviderData.ProductId, d.TypeName)

	var state LifecycleStagesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result LifecycleStagesAPIModel

	response, err := d.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"jfrog_url": state.JfrogUrl.ValueString(),
			"project_key": state.ProjectKey.ValueString(),
		}).
		SetResult(&result).
		Get(StagesEndpoint)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read apptrust_stages", err.Error())
		return
	}

	if response.IsError() {
		resp.Diagnostics.AddError("Unable to read apptrust_stages", response.String())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
