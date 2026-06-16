// Copyright (c) JFrog Ltd. (2025)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resource

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jfrog/terraform-provider-apptrust/pkg/apptrust"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
)

const ApplicationUnsyncIntegrationEP = ApplicationEndpoint + "/unsync_integration"

var _ resource.Resource = &ApplicationUnsyncIntegrationResource{}

func NewApplicationUnsyncIntegrationResource() resource.Resource {
	return &ApplicationUnsyncIntegrationResource{
		TypeName: "apptrust_application_unsync_integration",
	}
}

type ApplicationUnsyncIntegrationResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type ApplicationUnsyncIntegrationResourceModel struct {
	ID             types.String `tfsdk:"id"`
	ApplicationKey types.String `tfsdk:"application_key"`
	Message        types.String `tfsdk:"message"`
}

type unsyncIntegrationResponseAPIModel struct {
	Message string `json:"message"`
}

func (r *ApplicationUnsyncIntegrationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *ApplicationUnsyncIntegrationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Unsyncs an integration from an application by removing integration-related labels (POST /v1/applications/{application_key}/unsync_integration). This is an idempotent action: applying it ensures the application's integration is unsynced.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Computed resource ID (equal to application_key).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_key": schema.StringAttribute{
				Description: "The application key whose integration should be unsynced.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"message": schema.StringAttribute{
				Description: "Human-readable message describing the result of the operation.",
				Computed:    true,
			},
		},
	}
}

func (r *ApplicationUnsyncIntegrationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (r *ApplicationUnsyncIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan ApplicationUnsyncIntegrationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationKey := plan.ApplicationKey.ValueString()
	tflog.Info(ctx, "Unsyncing application integration", map[string]interface{}{"application_key": applicationKey})

	var apiResp unsyncIntegrationResponseAPIModel
	httpResponse, err := r.ProviderData.Client.R().
		SetContext(ctx).
		SetPathParam("application_key", applicationKey).
		SetResult(&apiResp).
		Post(ApplicationUnsyncIntegrationEP)

	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	if httpResponse.StatusCode() != http.StatusOK &&
		httpResponse.StatusCode() != http.StatusCreated &&
		httpResponse.StatusCode() != http.StatusAccepted {
		errorDiags := apptrust.HandleAPIErrorWithType(httpResponse, "unsync integration for", "application")
		resp.Diagnostics.Append(errorDiags...)
		return
	}

	plan.ID = types.StringValue(applicationKey)
	plan.Message = types.StringValue(apiResp.Message)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ApplicationUnsyncIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ApplicationUnsyncIntegrationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ApplicationUnsyncIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.Append(resp.State.Set(ctx, req.Plan)...)
}

func (r *ApplicationUnsyncIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)
	// Unsync is idempotent and has no inverse API; just remove from state.
}

func (r *ApplicationUnsyncIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_key"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
