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

var _ resource.Resource = &ApplicationVersionReleaseResource{}

func NewApplicationVersionReleaseResource() resource.Resource {
	return &ApplicationVersionReleaseResource{
		TypeName: "apptrust_application_version_release",
	}
}

type ApplicationVersionReleaseResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type ApplicationVersionReleaseResourceModel struct {
	ID                         types.String `tfsdk:"id"`
	ApplicationKey             types.String `tfsdk:"application_key"`
	Version                    types.String `tfsdk:"version"`
	PromotionType              types.String `tfsdk:"promotion_type"`
	IncludedRepositoryKeys     types.List   `tfsdk:"included_repository_keys"`
	ExcludedRepositoryKeys     types.List   `tfsdk:"excluded_repository_keys"`
	PromotionAuthorizationType types.String `tfsdk:"promotion_authorization_type"`
}

type releaseAppVersionRequestBody struct {
	PromotionType              string   `json:"promotion_type,omitempty"`
	IncludedRepositoryKeys     []string `json:"included_repository_keys,omitempty"`
	ExcludedRepositoryKeys     []string `json:"excluded_repository_keys,omitempty"`
	PromotionAuthorizationType string   `json:"promotion_authorization_type,omitempty"`
}

func (r *ApplicationVersionReleaseResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *ApplicationVersionReleaseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Releases an AppTrust application version to the PROD stage (POST /v1/applications/{application_key}/versions/{version}/release).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Computed ID (application_key:version).",
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
				Description: "The application version to release.",
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
				Description: "Repository keys to include.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"excluded_repository_keys": schema.ListAttribute{
				Description: "Repository keys to exclude.",
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

func (r *ApplicationVersionReleaseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (r *ApplicationVersionReleaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan ApplicationVersionReleaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	promotionType := "copy"
	if !plan.PromotionType.IsNull() && !plan.PromotionType.IsUnknown() {
		promotionType = plan.PromotionType.ValueString()
	}
	body := releaseAppVersionRequestBody{PromotionType: promotionType}
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
		Post(ApplicationVersionReleaseEP)

	if err != nil {
		tflog.Error(ctx, "Failed to release application version", map[string]interface{}{
			"application_key": plan.ApplicationKey.ValueString(),
			"version":         plan.Version.ValueString(),
			"error":           err.Error(),
		})
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	if httpResponse.StatusCode() != http.StatusOK && httpResponse.StatusCode() != http.StatusAccepted {
		errorDiags := apptrust.HandleAPIErrorWithType(httpResponse, "release", "application version")
		resp.Diagnostics.Append(errorDiags...)
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s:%s", plan.ApplicationKey.ValueString(), plan.Version.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ApplicationVersionReleaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ApplicationVersionReleaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ApplicationVersionReleaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.Append(resp.State.Set(ctx, req.Plan)...)
}

func (r *ApplicationVersionReleaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)
	// No API delete for release; just remove from state.
}

func (r *ApplicationVersionReleaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// ID format: application_key:version
	for i, c := range req.ID {
		if c == ':' {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_key"), req.ID[:i])...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("version"), req.ID[i+1:])...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
			return
		}
	}
	resp.Diagnostics.AddError("Invalid import ID", "Import ID must be application_key:version")
}
