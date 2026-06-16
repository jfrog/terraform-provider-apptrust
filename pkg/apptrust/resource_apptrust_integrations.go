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
	IntegrationsEndpoint = "https://{jfrog_url}/apptrust/api/v1/integrations/{integration_key}"
	IntegrationsEndpoint  = "https://{jfrog_url}/apptrust/api/v1/integrations/{integration_key}"
)

func NewApptrustIntegrationsResource() resource.Resource {
	return &ApptrustIntegrationsResource{
		TypeName: "apptrust_integrations",
	}
}

type ApptrustIntegrationsResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type ApptrustIntegrationsResourceModel struct {
	JfrogUrl types.String `tfsdk:"jfrog_url"`
	IntegrationKey types.String `tfsdk:"integration_key"`
}

type IntegrationsAPIModel struct {
	JfrogUrl string `json:"jfrog_url"`
	IntegrationKey string `json:"integration_key"`
}

func (r *ApptrustIntegrationsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *ApptrustIntegrationsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"jfrog_url": schema.StringAttribute{
				Required: true,
				Description: "The jfrog_url of the resource.",
			},
			"integration_key": schema.StringAttribute{
				Required: true,
				Description: "The integration_key of the resource.",
			},
		},
		MarkdownDescription: "Manages integrations in JFrog Apptrust.",
	}
}

func (r *ApptrustIntegrationsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}



func (r *ApptrustIntegrationsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ApptrustIntegrationsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result ApptrustIntegrationsAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"jfrog_url": state.JfrogUrl.ValueString(),
			"integration_key": state.IntegrationKey.ValueString(),
		}).
		SetResult(&result).
		Get(IntegrationsEndpoint)
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




func (r *ApptrustIntegrationsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ApptrustIntegrationsResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"jfrog_url": state.JfrogUrl.ValueString(),
			"integration_key": state.IntegrationKey.ValueString(),
		}).
		Delete(IntegrationsEndpoint)
	if err != nil {
		utilfw.UnableToDeleteResourceError(resp, err.Error())
		return
	}

	if response.StatusCode() == http.StatusNotFound {
		return
	}

	if response.IsError() {
		utilfw.UnableToDeleteResourceError(resp, response.String())
		return
	}
}

