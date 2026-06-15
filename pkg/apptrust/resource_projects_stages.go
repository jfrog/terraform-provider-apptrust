package apptrust

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
)

const (
	StagesEndpoint = "https://{jfrog_url}/apptrust/api/v1/projects/{project_key}/stages"
	StagesEndpoint  = "https://{jfrog_url}/apptrust/api/v1/projects/{project_key}/stages"
)

func NewProjectsStagesResource() resource.Resource {
	return &ProjectsStagesResource{
		TypeName: "apptrust_stages",
	}
}

type ProjectsStagesResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type ProjectsStagesResourceModel struct {
	JfrogUrl types.String `tfsdk:"jfrog_url"`
	ProjectKey types.String `tfsdk:"project_key"`
}

type StagesAPIModel struct {
	JfrogUrl string `json:"jfrog_url"`
	ProjectKey string `json:"project_key"`
}

func (r *ProjectsStagesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *ProjectsStagesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
		},
		MarkdownDescription: "Manages stages in JFrog Apptrust.",
	}
}

func (r *ProjectsStagesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}



func (r *ProjectsStagesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ProjectsStagesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result ProjectsStagesAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"jfrog_url": state.JfrogUrl.ValueString(),
			"project_key": state.ProjectKey.ValueString(),
		}).
		SetResult(&result).
		Get(StagesEndpoint)
	if err != nil {
		utilfw.UnableToRefreshResourceError(resp, err.Error())
		return
	}

	if response.StatusCode() == http.StatusNotFound {
		resp.State.RemoveResource(ctx)
		return
	}

	if response.IsError() {
		utilfw.UnableToRefreshResourceError(resp, response.String())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}




