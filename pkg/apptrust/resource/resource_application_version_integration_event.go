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
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jfrog/terraform-provider-apptrust/pkg/apptrust"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
)

const ApplicationVersionIntegrationEventEP = ApplicationVersionEndpoint + "/integration_event"

var _ resource.Resource = &ApplicationVersionIntegrationEventResource{}

func NewApplicationVersionIntegrationEventResource() resource.Resource {
	return &ApplicationVersionIntegrationEventResource{
		TypeName: "apptrust_application_version_integration_event",
	}
}

type ApplicationVersionIntegrationEventResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type ApplicationVersionIntegrationEventResourceModel struct {
	ID             types.String `tfsdk:"id"`
	ApplicationKey types.String `tfsdk:"application_key"`
	Version        types.String `tfsdk:"version"`
	Reference      types.String `tfsdk:"reference"`
	Status         types.String `tfsdk:"status"`
	Type           types.String `tfsdk:"type"`
	EventMessage   types.String `tfsdk:"event_message"`
	Properties     types.Map    `tfsdk:"properties"`
	Message        types.String `tfsdk:"message"`
}

type integrationEventProperty struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type integrationEventRequestBody struct {
	Reference    string                     `json:"reference"`
	Status       string                     `json:"status"`
	Type         string                     `json:"type"`
	EventMessage string                     `json:"event_message,omitempty"`
	Properties   []integrationEventProperty `json:"properties,omitempty"`
}

type integrationEventResponseAPIModel struct {
	Message string `json:"message"`
}

func (r *ApplicationVersionIntegrationEventResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *ApplicationVersionIntegrationEventResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	requiresReplaceString := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Posts an integration event for a specific application version (POST /v1/applications/{application_key}/versions/{version}/integration_event). Each apply records a single event; changing any attribute records a new event.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Computed resource ID (application_key:version:reference).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_key": schema.StringAttribute{
				Description:   "The application key.",
				Required:      true,
				PlanModifiers: requiresReplaceString,
			},
			"version": schema.StringAttribute{
				Description:   "The application version.",
				Required:      true,
				PlanModifiers: requiresReplaceString,
			},
			"reference": schema.StringAttribute{
				Description:   "The external reference identifier.",
				Required:      true,
				PlanModifiers: requiresReplaceString,
			},
			"status": schema.StringAttribute{
				Description:   "The integration event status.",
				Required:      true,
				PlanModifiers: requiresReplaceString,
			},
			"type": schema.StringAttribute{
				Description:   "The integration event type.",
				Required:      true,
				PlanModifiers: requiresReplaceString,
			},
			"event_message": schema.StringAttribute{
				Description:   "An optional message describing the event.",
				Optional:      true,
				PlanModifiers: requiresReplaceString,
			},
			"properties": schema.MapAttribute{
				Description: "Optional key-value properties associated with the event.",
				ElementType: types.StringType,
				Optional:    true,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"message": schema.StringAttribute{
				Description: "Human-readable summary of the outcome.",
				Computed:    true,
			},
		},
	}
}

func (r *ApplicationVersionIntegrationEventResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (r *ApplicationVersionIntegrationEventResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan ApplicationVersionIntegrationEventResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := integrationEventRequestBody{
		Reference: plan.Reference.ValueString(),
		Status:    plan.Status.ValueString(),
		Type:      plan.Type.ValueString(),
	}
	if !plan.EventMessage.IsNull() && !plan.EventMessage.IsUnknown() {
		body.EventMessage = plan.EventMessage.ValueString()
	}
	if !plan.Properties.IsNull() && !plan.Properties.IsUnknown() {
		elements := map[string]string{}
		resp.Diagnostics.Append(plan.Properties.ElementsAs(ctx, &elements, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for k, v := range elements {
			body.Properties = append(body.Properties, integrationEventProperty{Key: k, Value: v})
		}
	}

	tflog.Info(ctx, "Posting application version integration event", map[string]interface{}{
		"application_key": plan.ApplicationKey.ValueString(),
		"version":         plan.Version.ValueString(),
		"reference":       plan.Reference.ValueString(),
	})

	var apiResp integrationEventResponseAPIModel
	httpResponse, err := r.ProviderData.Client.R().
		SetContext(ctx).
		SetPathParam("application_key", plan.ApplicationKey.ValueString()).
		SetPathParam("version", plan.Version.ValueString()).
		SetBody(body).
		SetResult(&apiResp).
		Post(ApplicationVersionIntegrationEventEP)

	if err != nil {
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	if httpResponse.StatusCode() != http.StatusOK &&
		httpResponse.StatusCode() != http.StatusCreated &&
		httpResponse.StatusCode() != http.StatusAccepted {
		errorDiags := apptrust.HandleAPIErrorWithType(httpResponse, "post integration event for", "application version")
		resp.Diagnostics.Append(errorDiags...)
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s:%s:%s",
		plan.ApplicationKey.ValueString(), plan.Version.ValueString(), plan.Reference.ValueString()))
	plan.Message = types.StringValue(apiResp.Message)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ApplicationVersionIntegrationEventResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ApplicationVersionIntegrationEventResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ApplicationVersionIntegrationEventResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.Append(resp.State.Set(ctx, req.Plan)...)
}

func (r *ApplicationVersionIntegrationEventResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)
	// Integration events are append-only; there is no delete API, so just remove from state.
}
