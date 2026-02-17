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

var _ resource.Resource = &ApplicationVersionRollbackResource{}

func NewApplicationVersionRollbackResource() resource.Resource {
	return &ApplicationVersionRollbackResource{
		TypeName: "apptrust_application_version_rollback",
	}
}

type ApplicationVersionRollbackResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type ApplicationVersionRollbackResourceModel struct {
	ID             types.String `tfsdk:"id"`
	ApplicationKey types.String `tfsdk:"application_key"`
	Version        types.String `tfsdk:"version"`
	FromStage      types.String `tfsdk:"from_stage"`
}

type rollbackAppVersionRequestBody struct {
	FromStage string `json:"from_stage"`
}

func (r *ApplicationVersionRollbackResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *ApplicationVersionRollbackResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Rolls back the latest promotion of an AppTrust application version (POST /v1/applications/{application_key}/versions/{version}/rollback).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Computed ID (application_key:version:from_stage).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_key": schema.StringAttribute{
				Description: "The application key.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.StringAttribute{
				Description: "The application version to roll back.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"from_stage": schema.StringAttribute{
				Description: "Stage from which to roll back (e.g. qa, PROD).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *ApplicationVersionRollbackResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (r *ApplicationVersionRollbackResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan ApplicationVersionRollbackResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := rollbackAppVersionRequestBody{FromStage: plan.FromStage.ValueString()}

	httpResponse, err := r.ProviderData.Client.R().
		SetContext(ctx).
		SetPathParam("application_key", plan.ApplicationKey.ValueString()).
		SetPathParam("version", plan.Version.ValueString()).
		SetBody(body).
		Post(ApplicationVersionRollbackEP)

	if err != nil {
		tflog.Error(ctx, "Failed to roll back application version", map[string]interface{}{
			"application_key": plan.ApplicationKey.ValueString(),
			"version":         plan.Version.ValueString(),
			"from_stage":      plan.FromStage.ValueString(),
			"error":           err.Error(),
		})
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	if httpResponse.StatusCode() != http.StatusOK && httpResponse.StatusCode() != http.StatusAccepted {
		errorDiags := apptrust.HandleAPIErrorWithType(httpResponse, "rollback", "application version")
		resp.Diagnostics.Append(errorDiags...)
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s:%s:%s", plan.ApplicationKey.ValueString(), plan.Version.ValueString(), plan.FromStage.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ApplicationVersionRollbackResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ApplicationVersionRollbackResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ApplicationVersionRollbackResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.Append(resp.State.Set(ctx, req.Plan)...)
}

func (r *ApplicationVersionRollbackResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)
	// No API delete for rollback; just remove from state.
}

func (r *ApplicationVersionRollbackResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := splitPromotionID(req.ID)
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID", "Import ID must be application_key:version:from_stage")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_key"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("version"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("from_stage"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
