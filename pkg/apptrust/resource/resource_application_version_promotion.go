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

var _ resource.Resource = &ApplicationVersionPromotionResource{}

func NewApplicationVersionPromotionResource() resource.Resource {
	return &ApplicationVersionPromotionResource{
		TypeName: "apptrust_application_version_promotion",
	}
}

type ApplicationVersionPromotionResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type ApplicationVersionPromotionResourceModel struct {
	ID                         types.String `tfsdk:"id"`
	ApplicationKey             types.String `tfsdk:"application_key"`
	Version                    types.String `tfsdk:"version"`
	TargetStage                types.String `tfsdk:"target_stage"`
	PromotionType              types.String `tfsdk:"promotion_type"`
	IncludedRepositoryKeys     types.List   `tfsdk:"included_repository_keys"`
	ExcludedRepositoryKeys     types.List   `tfsdk:"excluded_repository_keys"`
	PromotionAuthorizationType types.String `tfsdk:"promotion_authorization_type"`
}

// PromoteAppVersionRequest per OpenAPI request.PromoteAppVersionRequest
type promoteAppVersionRequestBody struct {
	TargetStage                  string                   `json:"target_stage"`
	PromotionType                string                   `json:"promotion_type,omitempty"`
	IncludedRepositoryKeys       []string                 `json:"included_repository_keys,omitempty"`
	ExcludedRepositoryKeys       []string                 `json:"excluded_repository_keys,omitempty"`
	ArtifactAdditionalProperties []artifactAdditionalProp `json:"artifact_additional_properties,omitempty"`
	PromotionAuthorizationType   string                   `json:"promotion_authorization_type,omitempty"`
}

type artifactAdditionalProp struct {
	Key    string   `json:"key"`
	Values []string `json:"values"`
}

func (r *ApplicationVersionPromotionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *ApplicationVersionPromotionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Promotes an AppTrust application version to a target lifecycle stage (POST /v1/applications/{application_key}/versions/{version}/promote).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Computed ID (application_key:version:target_stage).",
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
				Description: "The application version to promote.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"target_stage": schema.StringAttribute{
				Description: "Target lifecycle stage (e.g. QA, PROD).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"promotion_type": schema.StringAttribute{
				Description: "Promotion type: move, copy, keep, or dry_run. Default is copy.",
				Optional:    true,
			},
			"included_repository_keys": schema.ListAttribute{
				Description: "Repository keys to include in the promotion.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"excluded_repository_keys": schema.ListAttribute{
				Description: "Repository keys to exclude from the promotion.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"promotion_authorization_type": schema.StringAttribute{
				Description: "Promotion authorization type.",
				Optional:    true,
			},
		},
	}
}

func (r *ApplicationVersionPromotionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func promotionID(appKey, version, targetStage string) string {
	return fmt.Sprintf("%s:%s:%s", appKey, version, targetStage)
}

func (r *ApplicationVersionPromotionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan ApplicationVersionPromotionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	promotionType := "copy"
	if !plan.PromotionType.IsNull() && !plan.PromotionType.IsUnknown() {
		promotionType = plan.PromotionType.ValueString()
	}
	body := promoteAppVersionRequestBody{
		TargetStage:   plan.TargetStage.ValueString(),
		PromotionType: promotionType,
	}
	if !plan.PromotionAuthorizationType.IsNull() && !plan.PromotionAuthorizationType.IsUnknown() {
		body.PromotionAuthorizationType = plan.PromotionAuthorizationType.ValueString()
	}
	if !plan.IncludedRepositoryKeys.IsNull() && !plan.IncludedRepositoryKeys.IsUnknown() {
		resp.Diagnostics.Append(plan.IncludedRepositoryKeys.ElementsAs(ctx, &body.IncludedRepositoryKeys, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	if !plan.ExcludedRepositoryKeys.IsNull() && !plan.ExcludedRepositoryKeys.IsUnknown() {
		resp.Diagnostics.Append(plan.ExcludedRepositoryKeys.ElementsAs(ctx, &body.ExcludedRepositoryKeys, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	httpResponse, err := r.ProviderData.Client.R().
		SetContext(ctx).
		SetPathParam("application_key", plan.ApplicationKey.ValueString()).
		SetPathParam("version", plan.Version.ValueString()).
		SetBody(body).
		Post(ApplicationVersionPromoteEP)

	if err != nil {
		tflog.Error(ctx, "Failed to promote application version", map[string]interface{}{
			"application_key": plan.ApplicationKey.ValueString(),
			"version":         plan.Version.ValueString(),
			"target_stage":    plan.TargetStage.ValueString(),
			"error":           err.Error(),
		})
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	if httpResponse.StatusCode() != http.StatusOK && httpResponse.StatusCode() != http.StatusAccepted {
		errorDiags := apptrust.HandleAPIErrorWithType(httpResponse, "promote", "application version")
		resp.Diagnostics.Append(errorDiags...)
		return
	}

	plan.ID = types.StringValue(promotionID(plan.ApplicationKey.ValueString(), plan.Version.ValueString(), plan.TargetStage.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ApplicationVersionPromotionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ApplicationVersionPromotionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Promotion is a one-shot action; we do not refresh from API. State is enough.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ApplicationVersionPromotionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// No-op: changing target_stage etc. requires replace (RequiresReplace on key attrs).
	resp.Diagnostics.Append(resp.State.Set(ctx, req.Plan)...)
}

func (r *ApplicationVersionPromotionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)
	// No API delete for promotion; just remove from state.
}

func (r *ApplicationVersionPromotionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// ID format: application_key:version:target_stage
	parts := splitPromotionID(req.ID)
	if len(parts) != 3 {
		resp.Diagnostics.AddError("Invalid import ID", "Import ID must be application_key:version:target_stage (e.g. my-app:1.0.0:QA)")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_key"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("version"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("target_stage"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func splitPromotionID(id string) []string {
	var parts []string
	start := 0
	for i, c := range id {
		if c == ':' {
			parts = append(parts, id[start:i])
			start = i + 1
		}
	}
	if start <= len(id) {
		parts = append(parts, id[start:])
	}
	return parts
}
