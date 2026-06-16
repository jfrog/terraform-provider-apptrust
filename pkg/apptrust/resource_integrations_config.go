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
	ConfigEndpoint = "https://{jfrog_url}/apptrust/api/v1/integrations/config"
	ConfigEndpoint  = "https://{jfrog_url}/apptrust/api/v1/integrations/config"
)

func NewIntegrationsConfigResource() resource.Resource {
	return &IntegrationsConfigResource{
		TypeName: "apptrust_config",
	}
}

type IntegrationsConfigResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type IntegrationsConfigResourceModel struct {
	JfrogUrl types.String `tfsdk:"jfrog_url"`
}

type ConfigRequestAPIModel struct {
	JfrogUrl string `json:"jfrog_url"`
}

type ConfigAPIModel struct {
	JfrogUrl string `json:"jfrog_url"`
}

func (r *IntegrationsConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *IntegrationsConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"jfrog_url": schema.StringAttribute{
				Required: true,
				Description: "The jfrog_url of the resource.",
			},
		},
		MarkdownDescription: "Manages config in JFrog Apptrust.",
	}
}

func (r *IntegrationsConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}



func (r *IntegrationsConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state IntegrationsConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result IntegrationsConfigAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"jfrog_url": state.JfrogUrl.ValueString(),
		}).
		SetResult(&result).
		Get(ConfigEndpoint)
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


func (r *IntegrationsConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan IntegrationsConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requestBody := IntegrationsConfigRequestAPIModel{
		JfrogUrl: plan.JfrogUrl.ValueString(),
	}

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"jfrog_url": plan.JfrogUrl.ValueString(),
		}).
		SetBody(requestBody).
		Put(ConfigEndpoint)
	if err != nil {
		utilfw.UnableToUpdateResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToUpdateResourceError(resp, response.String())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}



