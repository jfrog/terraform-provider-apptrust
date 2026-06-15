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
	IntegrationEventEndpoint = "https://{jfrog_url}/apptrust/api/v1/applications/{application_key}/versions/{version}/integration_event"
	IntegrationEventEndpoint  = "https://{jfrog_url}/apptrust/api/v1/applications/{application_key}/versions/{version}/integration_event"
)

func NewVersionsIntegrationEventResource() resource.Resource {
	return &VersionsIntegrationEventResource{
		TypeName: "apptrust_integration_event",
	}
}

type VersionsIntegrationEventResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type VersionsIntegrationEventResourceModel struct {
	JfrogUrl types.String `tfsdk:"jfrog_url"`
	ApplicationKey types.String `tfsdk:"application_key"`
	Version types.String `tfsdk:"version"`
}

type IntegrationEventRequestAPIModel struct {
	JfrogUrl string `json:"jfrog_url"`
	ApplicationKey string `json:"application_key"`
	Version string `json:"version"`
}

type IntegrationEventAPIModel struct {
	JfrogUrl string `json:"jfrog_url"`
	ApplicationKey string `json:"application_key"`
	Version string `json:"version"`
}

func (r *VersionsIntegrationEventResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *VersionsIntegrationEventResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
		},
		MarkdownDescription: "Manages integration_event in JFrog Apptrust.",
	}
}

func (r *VersionsIntegrationEventResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}


func (r *VersionsIntegrationEventResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan VersionsIntegrationEventResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requestBody := VersionsIntegrationEventRequestAPIModel{
		JfrogUrl: plan.JfrogUrl.ValueString(),
		ApplicationKey: plan.ApplicationKey.ValueString(),
		Version: plan.Version.ValueString(),
	}

	var result VersionsIntegrationEventAPIModel

	response, err := r.ProviderData.Client.R().
		SetBody(requestBody).
		SetResult(&result).
		Post(IntegrationEventEndpoint)
	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	if response.IsError() {
		utilfw.UnableToCreateResourceError(resp, response.String())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}


func (r *VersionsIntegrationEventResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state VersionsIntegrationEventResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result VersionsIntegrationEventAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"jfrog_url": state.JfrogUrl.ValueString(),
			"application_key": state.ApplicationKey.ValueString(),
			"version": state.Version.ValueString(),
		}).
		SetResult(&result).
		Get(IntegrationEventEndpoint)
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




