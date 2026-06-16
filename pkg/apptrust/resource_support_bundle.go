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
	BundleEndpoint = "https://{jfrog_url}/apptrust/api/v1/system/support/bundle/{id}"
	BundleEndpoint  = "https://{jfrog_url}/apptrust/api/v1/system/support/bundle/{id}"
)

func NewSupportBundleResource() resource.Resource {
	return &SupportBundleResource{
		TypeName: "apptrust_bundle",
	}
}

type SupportBundleResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type SupportBundleResourceModel struct {
	JfrogUrl types.String `tfsdk:"jfrog_url"`
	Id types.String `tfsdk:"id"`
}

type BundleRequestAPIModel struct {
	JfrogUrl string `json:"jfrog_url"`
	Id string `json:"id"`
}

type BundleAPIModel struct {
	JfrogUrl string `json:"jfrog_url"`
	Id string `json:"id"`
}

func (r *SupportBundleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *SupportBundleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"jfrog_url": schema.StringAttribute{
				Required: true,
				Description: "The jfrog_url of the resource.",
			},
			"id": schema.StringAttribute{
				Required: true,
				Description: "The id of the resource.",
			},
		},
		MarkdownDescription: "Manages bundle in JFrog Apptrust.",
	}
}

func (r *SupportBundleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}



func (r *SupportBundleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state SupportBundleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result SupportBundleAPIModel

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"jfrog_url": state.JfrogUrl.ValueString(),
			"id": state.Id.ValueString(),
		}).
		SetResult(&result).
		Get(BundleEndpoint)
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


func (r *SupportBundleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan SupportBundleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	requestBody := SupportBundleRequestAPIModel{
		JfrogUrl: plan.JfrogUrl.ValueString(),
		Id: plan.Id.ValueString(),
	}

	response, err := r.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"jfrog_url": plan.JfrogUrl.ValueString(),
			"id": plan.Id.ValueString(),
		}).
		SetBody(requestBody).
		Put(BundleEndpoint)
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



