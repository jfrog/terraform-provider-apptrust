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
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jfrog/terraform-provider-apptrust/pkg/apptrust"
	"github.com/jfrog/terraform-provider-shared/util"
	utilfw "github.com/jfrog/terraform-provider-shared/util/fw"
)

// Application API endpoints (used by this resource and application datasources)
const (
	ApplicationsEndpoint = "apptrust/api/v1/applications"
	ApplicationEndpoint  = ApplicationsEndpoint + "/{application_key}"
)

var _ resource.Resource = &ApplicationResource{}

func NewApplicationResource() resource.Resource {
	return &ApplicationResource{
		TypeName: "apptrust_application",
	}
}

type ApplicationResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type ApplicationResourceModel struct {
	ID              types.String `tfsdk:"id"` // Computed ID that maps to application_key for Terraform compatibility
	ApplicationKey  types.String `tfsdk:"application_key"`
	ApplicationName types.String `tfsdk:"application_name"`
	ProjectKey      types.String `tfsdk:"project_key"`
	Description     types.String `tfsdk:"description"`
	MaturityLevel   types.String `tfsdk:"maturity_level"`
	Criticality     types.String `tfsdk:"criticality"`
	Labels          types.Map    `tfsdk:"labels"`
	UserOwners      types.List   `tfsdk:"user_owners"`
	GroupOwners     types.List   `tfsdk:"group_owners"`
}

type ApplicationAPIModel struct {
	ApplicationKey  string            `json:"application_key"`
	ApplicationName string            `json:"application_name"`
	ProjectKey      string            `json:"project_key"`
	Description     string            `json:"description,omitempty"`
	MaturityLevel   string            `json:"maturity_level,omitempty"`
	Criticality     string            `json:"criticality,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	UserOwners      []string          `json:"user_owners,omitempty"`
	GroupOwners     []string          `json:"group_owners,omitempty"`
}

type UpdateApplicationAPIModel struct {
	ApplicationName *string           `json:"application_name,omitempty"`
	Description     *string           `json:"description,omitempty"`
	MaturityLevel   *string           `json:"maturity_level,omitempty"`
	Criticality     *string           `json:"criticality,omitempty"`
	Labels          map[string]string `json:"labels"`       // No omitempty - empty map must be sent to clear
	UserOwners      []string          `json:"user_owners"`  // No omitempty - empty array must be sent to clear
	GroupOwners     []string          `json:"group_owners"` // No omitempty - empty array must be sent to clear
}

var (
	maturityLevels    = []string{"unspecified", "experimental", "production", "end_of_life"}
	criticalityLevels = []string{"unspecified", "low", "medium", "high", "critical"}
)

func (r *ApplicationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *ApplicationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provides an AppTrust application resource. This resource allows you to create, update, and delete AppTrust applications. " +
			"Applications are business-aware entities that serve as a definitive, centralized system of record for all software assets throughout their lifecycle.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of this resource. This is computed and always equals the application_key.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_key": schema.StringAttribute{
				Description: "The application key. Must be 2-64 lowercase alphanumeric characters, beginning with a letter (hyphens are supported). " +
					"The key must be unique and immutable. Cannot be changed after creation. Changing this field will force replacement of the resource.",
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(2, 64),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z][a-z0-9\-]*[a-z0-9]$|^[a-z]$`),
						"application_key must be 2-64 lowercase alphanumeric characters and hyphens, beginning with a letter",
					),
				},
			},
			"application_name": schema.StringAttribute{
				Description: "The application display name. Must be a unique string within the scope of the project, " +
					"1-255 alphanumeric characters in length, including underscores, hyphens, and spaces.",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"project_key": schema.StringAttribute{
				Description: "The key of the project associated with the application. Cannot be changed after creation. Changing this field will force replacement of the resource.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description: "A free-text description of the application.",
				Optional:    true,
			},
			"maturity_level": schema.StringAttribute{
				Description: fmt.Sprintf("The maturity level of the application. Allowed values: %s. Defaults to 'unspecified' if not set.", strings.Join(maturityLevels, ", ")),
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("unspecified"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(maturityLevels...),
				},
			},
			"criticality": schema.StringAttribute{
				Description: fmt.Sprintf("A classification of how critical the application is for your business. Allowed values: %s. Defaults to 'unspecified' if not set.", strings.Join(criticalityLevels, ", ")),
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("unspecified"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(criticalityLevels...),
				},
			},
			"labels": schema.MapAttribute{
				Description: "Key-value pairs for labeling the application. Each key and value is free text, limited to 255 characters, " +
					"beginning and ending with an alphanumeric character ([a-z0-9A-Z]) with dashes (-), underscores (_), dots (.), and alphanumerics in between.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"user_owners": schema.ListAttribute{
				Description: "List of users defined in the project who own the application. Each user must be at least 1 character in length.",
				ElementType: types.StringType,
				Optional:    true,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						stringvalidator.LengthAtLeast(1),
					),
				},
			},
			"group_owners": schema.ListAttribute{
				Description: "List of user groups defined in the project who own the application. Each group must be at least 1 character in length.",
				ElementType: types.StringType,
				Optional:    true,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						stringvalidator.LengthAtLeast(1),
					),
				},
			},
		},
	}
}

func (r *ApplicationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (r *ApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan ApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiModel, diags := plan.toAPIModel(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// All fields are optional except application_key, application_name, and project_key
	createBody := map[string]interface{}{
		"application_key":  apiModel.ApplicationKey,
		"application_name": apiModel.ApplicationName,
		"project_key":      apiModel.ProjectKey,
	}
	if apiModel.Description != "" {
		createBody["description"] = apiModel.Description
	}
	if apiModel.MaturityLevel != "" {
		createBody["maturity_level"] = apiModel.MaturityLevel
	}
	if apiModel.Criticality != "" {
		createBody["criticality"] = apiModel.Criticality
	}
	if len(apiModel.Labels) > 0 {
		createBody["labels"] = apiModel.Labels
	}
	// Only send user_owners/group_owners when there are items. Null or empty list [] = don't send (API treats as no owners).
	if len(apiModel.UserOwners) > 0 {
		createBody["user_owners"] = apiModel.UserOwners
	}
	if len(apiModel.GroupOwners) > 0 {
		createBody["group_owners"] = apiModel.GroupOwners
	}

	var result ApplicationAPIModel
	httpResponse, err := r.ProviderData.Client.R().
		SetContext(ctx).
		SetBody(createBody).
		SetResult(&result).
		Post(ApplicationsEndpoint)

	if err != nil {
		tflog.Error(ctx, "Failed to send create request", map[string]interface{}{
			"application_key": plan.ApplicationKey.ValueString(),
			"error":           err.Error(),
		})
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	if httpResponse.StatusCode() != http.StatusCreated {
		if httpResponse.StatusCode() == http.StatusConflict {
			tflog.Warn(ctx, "Application already exists", map[string]interface{}{
				"application_key": plan.ApplicationKey.ValueString(),
			})
			resp.Diagnostics.AddError(
				"Application Already Exists",
				fmt.Sprintf("An application with key '%s' already exists. Please use a different application_key.", plan.ApplicationKey.ValueString()),
			)
			return
		}
		errorDiags := apptrust.HandleAPIError(httpResponse, "create")
		resp.Diagnostics.Append(errorDiags...)
		return
	}

	// Record if plan had explicit empty values before fromAPIModel overwrites (API may omit or return empty).
	planHadEmptyLabels := false
	if !plan.Labels.IsNull() && !plan.Labels.IsUnknown() && len(plan.Labels.Elements()) == 0 {
		planHadEmptyLabels = true
	}
	planHadEmptyUserOwners := false
	if !plan.UserOwners.IsNull() && !plan.UserOwners.IsUnknown() {
		var planOwners []string
		if diags := plan.UserOwners.ElementsAs(ctx, &planOwners, false); !diags.HasError() && len(planOwners) == 0 {
			planHadEmptyUserOwners = true
		}
	}
	planHadEmptyGroupOwners := false
	if !plan.GroupOwners.IsNull() && !plan.GroupOwners.IsUnknown() {
		var planOwners []string
		if diags := plan.GroupOwners.ElementsAs(ctx, &planOwners, false); !diags.HasError() && len(planOwners) == 0 {
			planHadEmptyGroupOwners = true
		}
	}

	diags = plan.fromAPIModel(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// When plan had empty value and API returned empty/nothing, preserve in state so state matches plan.
	planHadEmptyDescription := !plan.Description.IsNull() && !plan.Description.IsUnknown() && plan.Description.ValueString() == ""
	if planHadEmptyDescription && result.Description == "" {
		plan.Description = types.StringValue("")
	}
	if planHadEmptyLabels && len(result.Labels) == 0 {
		plan.Labels = types.MapValueMust(types.StringType, map[string]attr.Value{})
	}
	if planHadEmptyUserOwners && (result.UserOwners == nil || len(result.UserOwners) == 0) {
		plan.UserOwners = types.ListValueMust(types.StringType, []attr.Value{})
	}
	if planHadEmptyGroupOwners && (result.GroupOwners == nil || len(result.GroupOwners) == 0) {
		plan.GroupOwners = types.ListValueMust(types.StringType, []attr.Value{})
	}

	// Ensure ID is always set to application_key (computed field)
	plan.ID = types.StringValue(plan.ApplicationKey.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Support import: after ImportStatePassthroughID only id is set (id = application_key)
	applicationKey := state.ApplicationKey.ValueString()
	if applicationKey == "" {
		applicationKey = state.ID.ValueString()
	}
	tflog.Info(ctx, "Reading application", map[string]interface{}{
		"application_key": applicationKey,
	})

	var result ApplicationAPIModel
	httpResponse, err := r.ProviderData.Client.R().
		SetContext(ctx).
		SetPathParam("application_key", applicationKey).
		SetResult(&result).
		Get(ApplicationEndpoint)

	if err != nil {
		tflog.Error(ctx, "Failed to send read request", map[string]interface{}{
			"application_key": applicationKey,
			"error":           err.Error(),
		})
		utilfw.UnableToRefreshResourceError(resp, err.Error())
		return
	}

	if httpResponse.StatusCode() != http.StatusOK {
		if httpResponse.StatusCode() == http.StatusNotFound {
			tflog.Warn(ctx, "Application not found, removing from state", map[string]interface{}{
				"application_key": applicationKey,
			})
			resp.State.RemoveResource(ctx)
			return
		}
		errorDiags := apptrust.HandleAPIError(httpResponse, "read")
		resp.Diagnostics.Append(errorDiags...)
		return
	}

	// Record if state had explicit empty values before fromAPIModel overwrites.
	stateHadEmptyDescription := !state.Description.IsNull() && !state.Description.IsUnknown() && state.Description.ValueString() == ""
	stateHadEmptyLabels := false
	if !state.Labels.IsNull() && !state.Labels.IsUnknown() && len(state.Labels.Elements()) == 0 {
		stateHadEmptyLabels = true
	}
	stateHadEmptyUserOwners := false
	if !state.UserOwners.IsNull() && !state.UserOwners.IsUnknown() {
		var stateOwners []string
		if diags := state.UserOwners.ElementsAs(ctx, &stateOwners, false); !diags.HasError() && len(stateOwners) == 0 {
			stateHadEmptyUserOwners = true
		}
	}
	stateHadEmptyGroupOwners := false
	if !state.GroupOwners.IsNull() && !state.GroupOwners.IsUnknown() {
		var stateOwners []string
		if diags := state.GroupOwners.ElementsAs(ctx, &stateOwners, false); !diags.HasError() && len(stateOwners) == 0 {
			stateHadEmptyGroupOwners = true
		}
	}

	diags := state.fromAPIModel(ctx, result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// When state had empty value and API returns nothing, preserve in state so state matches.
	if stateHadEmptyDescription && result.Description == "" {
		state.Description = types.StringValue("")
	}
	if stateHadEmptyLabels && len(result.Labels) == 0 {
		state.Labels = types.MapValueMust(types.StringType, map[string]attr.Value{})
	}
	if stateHadEmptyUserOwners && (result.UserOwners == nil || len(result.UserOwners) == 0) {
		state.UserOwners = types.ListValueMust(types.StringType, []attr.Value{})
	}
	if stateHadEmptyGroupOwners && (result.GroupOwners == nil || len(result.GroupOwners) == 0) {
		state.GroupOwners = types.ListValueMust(types.StringType, []attr.Value{})
	}

	// Ensure ID is always set to application_key (computed field)
	state.ID = types.StringValue(state.ApplicationKey.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan ApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiModel, diags := plan.toAPIModelForUpdate(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If plan has null but state had values, send empty strings/maps/arrays to clear them
	// Use pointers: nil = don't update, &"" = clear field
	if plan.Description.IsNull() && !state.Description.IsNull() {
		emptyStr := ""
		apiModel.Description = &emptyStr
	}
	if plan.Labels.IsNull() && !state.Labels.IsNull() {
		apiModel.Labels = make(map[string]string)
	}
	if plan.UserOwners.IsNull() && !state.UserOwners.IsNull() {
		apiModel.UserOwners = []string{}
	}
	if plan.GroupOwners.IsNull() && !state.GroupOwners.IsNull() {
		apiModel.GroupOwners = []string{}
	}

	var result ApplicationAPIModel
	// NOTE: The provider sends "project" query parameter for context/authorization purposes.
	response, err := r.ProviderData.Client.R().
		SetContext(ctx).
		SetPathParam("application_key", plan.ApplicationKey.ValueString()).
		SetQueryParam("project", plan.ProjectKey.ValueString()).
		SetBody(apiModel).
		SetResult(&result).
		Patch(ApplicationEndpoint)

	if err != nil {
		tflog.Error(ctx, "Failed to send update request", map[string]interface{}{
			"application_key": plan.ApplicationKey.ValueString(),
			"error":           err.Error(),
		})
		utilfw.UnableToUpdateResourceError(resp, err.Error())
		return
	}

	if response.StatusCode() != http.StatusOK {
		errorDiags := apptrust.HandleAPIError(response, "update")
		resp.Diagnostics.Append(errorDiags...)
		return
	}

	// Track what the plan originally wanted before fromAPIModel modifies it
	planWantedDescriptionNull := plan.Description.IsNull() && !state.Description.IsNull()
	planWantedLabelsNull := plan.Labels.IsNull() && !state.Labels.IsNull()
	planWantedUserOwnersNull := plan.UserOwners.IsNull() && !state.UserOwners.IsNull()
	planWantedGroupOwnersNull := plan.GroupOwners.IsNull() && !state.GroupOwners.IsNull()
	planHadEmptyDescription := !plan.Description.IsNull() && !plan.Description.IsUnknown() && plan.Description.ValueString() == ""
	planHadEmptyLabels := false
	if !plan.Labels.IsNull() && !plan.Labels.IsUnknown() && len(plan.Labels.Elements()) == 0 {
		planHadEmptyLabels = true
	}
	planHadEmptyUserOwners := false
	if !plan.UserOwners.IsNull() && !plan.UserOwners.IsUnknown() {
		var planOwners []string
		if diags := plan.UserOwners.ElementsAs(ctx, &planOwners, false); !diags.HasError() && len(planOwners) == 0 {
			planHadEmptyUserOwners = true
		}
	}
	planHadEmptyGroupOwners := false
	if !plan.GroupOwners.IsNull() && !plan.GroupOwners.IsUnknown() {
		var planOwners []string
		if diags := plan.GroupOwners.ElementsAs(ctx, &planOwners, false); !diags.HasError() && len(planOwners) == 0 {
			planHadEmptyGroupOwners = true
		}
	}

	resp.Diagnostics.Append(plan.fromAPIModel(ctx, result)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// When plan wanted to clear (null) and API returned empty, set state to null. When plan had empty value, preserve it.
	if planWantedDescriptionNull && result.Description == "" {
		plan.Description = types.StringNull()
	} else if planHadEmptyDescription && result.Description == "" {
		plan.Description = types.StringValue("")
	}
	if planWantedLabelsNull && len(result.Labels) == 0 {
		plan.Labels = types.MapNull(types.StringType)
	} else if planHadEmptyLabels && len(result.Labels) == 0 {
		plan.Labels = types.MapValueMust(types.StringType, map[string]attr.Value{})
	}
	// When plan had null and API returned nothing, set state to null. When plan had [] and API returned nothing, preserve empty list.
	if planWantedUserOwnersNull && (result.UserOwners == nil || len(result.UserOwners) == 0) {
		plan.UserOwners = types.ListNull(types.StringType)
	} else if planHadEmptyUserOwners && (result.UserOwners == nil || len(result.UserOwners) == 0) {
		plan.UserOwners = types.ListValueMust(types.StringType, []attr.Value{})
	}
	if planWantedGroupOwnersNull && (result.GroupOwners == nil || len(result.GroupOwners) == 0) {
		plan.GroupOwners = types.ListNull(types.StringType)
	} else if planHadEmptyGroupOwners && (result.GroupOwners == nil || len(result.GroupOwners) == 0) {
		plan.GroupOwners = types.ListValueMust(types.StringType, []attr.Value{})
	}
	// maturity_level and criticality are already set by fromAPIModel (with "" normalized to "unspecified").

	// Always set ID to application_key (computed field)
	plan.ID = types.StringValue(plan.ApplicationKey.ValueString())

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationKey := state.ApplicationKey.ValueString()
	tflog.Info(ctx, "Deleting application", map[string]interface{}{
		"application_key": applicationKey,
	})

	httpResponse, err := r.ProviderData.Client.R().
		SetContext(ctx).
		SetPathParam("application_key", applicationKey).
		Delete(ApplicationEndpoint)

	if err != nil {
		tflog.Error(ctx, "Failed to send delete request", map[string]interface{}{
			"application_key": applicationKey,
			"error":           err.Error(),
		})
		utilfw.UnableToDeleteResourceError(resp, err.Error())
		return
	}

	if httpResponse.StatusCode() == http.StatusNoContent || httpResponse.StatusCode() == http.StatusOK {
		return
	}
	if httpResponse.StatusCode() == http.StatusNotFound {
		tflog.Warn(ctx, "Application not found during delete, assuming already deleted", map[string]interface{}{
			"application_key": applicationKey,
		})
		return
	}
	errorDiags := apptrust.HandleAPIError(httpResponse, "delete")
	resp.Diagnostics.Append(errorDiags...)
}

func (m *ApplicationResourceModel) toAPIModel(ctx context.Context) (ApplicationAPIModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	apiModel := ApplicationAPIModel{
		ApplicationKey:  m.ApplicationKey.ValueString(),
		ApplicationName: m.ApplicationName.ValueString(),
		ProjectKey:      m.ProjectKey.ValueString(),
	}

	if !m.Description.IsNull() {
		apiModel.Description = m.Description.ValueString()
	}

	if !m.MaturityLevel.IsNull() {
		apiModel.MaturityLevel = m.MaturityLevel.ValueString()
	}

	if !m.Criticality.IsNull() {
		apiModel.Criticality = m.Criticality.ValueString()
	}

	if !m.Labels.IsNull() {
		labels := make(map[string]string)
		diags.Append(m.Labels.ElementsAs(ctx, &labels, false)...)
		if !diags.HasError() {
			apiModel.Labels = labels
		}
	}

	if !m.UserOwners.IsNull() {
		var userOwners []string
		diags.Append(m.UserOwners.ElementsAs(ctx, &userOwners, false)...)
		if !diags.HasError() {
			apiModel.UserOwners = userOwners
		}
	}

	if !m.GroupOwners.IsNull() {
		var groupOwners []string
		diags.Append(m.GroupOwners.ElementsAs(ctx, &groupOwners, false)...)
		if !diags.HasError() {
			apiModel.GroupOwners = groupOwners
		}
	}

	return apiModel, diags
}

func (m *ApplicationResourceModel) toAPIModelForUpdate(ctx context.Context) (UpdateApplicationAPIModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	apiModel := UpdateApplicationAPIModel{}

	// Use pointers to differentiate between null (don't update) and empty string (clear field)
	// Optional string fields use *string
	if !m.ApplicationName.IsNull() {
		val := m.ApplicationName.ValueString()
		apiModel.ApplicationName = &val
	}

	if !m.Description.IsNull() {
		val := m.Description.ValueString()
		apiModel.Description = &val
	}

	if !m.MaturityLevel.IsNull() {
		val := m.MaturityLevel.ValueString()
		apiModel.MaturityLevel = &val
	}

	if !m.Criticality.IsNull() {
		val := m.Criticality.ValueString()
		apiModel.Criticality = &val
	}

	if !m.Labels.IsNull() {
		labels := make(map[string]string)
		diags.Append(m.Labels.ElementsAs(ctx, &labels, false)...)
		if !diags.HasError() {
			apiModel.Labels = labels
		}
	}

	if !m.UserOwners.IsNull() {
		var userOwners []string
		diags.Append(m.UserOwners.ElementsAs(ctx, &userOwners, false)...)
		if !diags.HasError() {
			apiModel.UserOwners = userOwners
		}
	}

	if !m.GroupOwners.IsNull() {
		var groupOwners []string
		diags.Append(m.GroupOwners.ElementsAs(ctx, &groupOwners, false)...)
		if !diags.HasError() {
			apiModel.GroupOwners = groupOwners
		}
	}

	return apiModel, diags
}

func (m *ApplicationResourceModel) fromAPIModel(ctx context.Context, api ApplicationAPIModel) diag.Diagnostics {
	var diags diag.Diagnostics

	// Set ID to application_key for Terraform compatibility
	m.ID = types.StringValue(api.ApplicationKey)
	m.ApplicationKey = types.StringValue(api.ApplicationKey)
	m.ApplicationName = types.StringValue(api.ApplicationName)
	m.ProjectKey = types.StringValue(api.ProjectKey)

	if api.Description != "" {
		m.Description = types.StringValue(api.Description)
	} else {
		m.Description = types.StringNull()
	}

	// Normalize empty to default so state never has null (schema default is "unspecified")
	if api.MaturityLevel != "" {
		m.MaturityLevel = types.StringValue(api.MaturityLevel)
	} else {
		m.MaturityLevel = types.StringValue("unspecified")
	}
	if api.Criticality != "" {
		m.Criticality = types.StringValue(api.Criticality)
	} else {
		m.Criticality = types.StringValue("unspecified")
	}

	if len(api.Labels) > 0 {
		labels := make(map[string]types.String)
		for k, v := range api.Labels {
			labels[k] = types.StringValue(v)
		}
		labelsMap, d := types.MapValueFrom(ctx, types.StringType, labels)
		diags.Append(d...)
		if !diags.HasError() {
			m.Labels = labelsMap
		}
	} else {
		m.Labels = types.MapNull(types.StringType)
	}

	// API BEHAVIOR: No owners is represented as null in state (API omits or returns []).
	// Empty list [] is preserved when plan/state had [] and API returns nothing (see Create/Read/Update).
	if api.UserOwners != nil && len(api.UserOwners) > 0 {
		userOwners := make([]types.String, len(api.UserOwners))
		for i, v := range api.UserOwners {
			userOwners[i] = types.StringValue(v)
		}
		userOwnersList, d := types.ListValueFrom(ctx, types.StringType, userOwners)
		diags.Append(d...)
		if !diags.HasError() {
			m.UserOwners = userOwnersList
		}
	} else {
		m.UserOwners = types.ListNull(types.StringType)
	}

	if api.GroupOwners != nil && len(api.GroupOwners) > 0 {
		groupOwners := make([]types.String, len(api.GroupOwners))
		for i, v := range api.GroupOwners {
			groupOwners[i] = types.StringValue(v)
		}
		groupOwnersList, d := types.ListValueFrom(ctx, types.StringType, groupOwners)
		diags.Append(d...)
		if !diags.HasError() {
			m.GroupOwners = groupOwnersList
		}
	} else {
		m.GroupOwners = types.ListNull(types.StringType)
	}

	return diags
}

// ImportState imports an existing application using the application_key as the import ID.
// Example: terraform import apptrust_application.example my-application-key
func (r *ApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
