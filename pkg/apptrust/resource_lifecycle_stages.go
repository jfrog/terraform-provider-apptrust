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
	StagesEndpoint = "https://{jfrog_url}/apptrust/api/v1/projects/{project_key}/lifecycle/stages"
	StagesEndpoint  = "https://{jfrog_url}/apptrust/api/v1/projects/{project_key}/lifecycle/stages"
)

func NewLifecycleStagesResource() resource.Resource {
	return &LifecycleStagesResource{
		TypeName: "apptrust_stages",
	}
}

type LifecycleStagesResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type LifecycleStagesResourceModel struct {
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

func (r *LifecycleStagesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *LifecycleStagesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
				Optional: true,
				Description: "Application key to scope policies",
			},
			"filter_gates_by": schema.StringAttribute{
				Optional: true,
				Description: "Filter gates - if 'policies', only return gates with policies; otherwise, return category-based gates (promote: entry+exit, others: release)",
			},
		},
		MarkdownDescription: "Manages stages in JFrog Apptrust.",
	}
}

func (r *LifecycleStagesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}



func (r *LifecycleStagesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state LifecycleStagesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result LifecycleStagesAPIModel

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




