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

const (
	ApplicationVersionsEndpoint    = ApplicationEndpoint + "/versions"
	ApplicationVersionEndpoint     = ApplicationVersionsEndpoint + "/{version}"
	ApplicationVersionPromoteEP    = ApplicationVersionEndpoint + "/promote"
	ApplicationVersionReleaseEP    = ApplicationVersionEndpoint + "/release"
	ApplicationVersionRollbackEP   = ApplicationVersionEndpoint + "/rollback"
	ApplicationVersionStatusEP     = ApplicationVersionEndpoint + "/status"
	ApplicationVersionPromotionsEP = ApplicationVersionEndpoint + "/promotions"
)

var _ resource.Resource = &ApplicationVersionResource{}

func NewApplicationVersionResource() resource.Resource {
	return &ApplicationVersionResource{
		TypeName: "apptrust_application_version",
	}
}

type ApplicationVersionResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type ApplicationVersionResourceModel struct {
	ID              types.String `tfsdk:"id"`
	ApplicationKey  types.String `tfsdk:"application_key"`
	Version         types.String `tfsdk:"version"`
	Tag             types.String `tfsdk:"tag"`
	Draft           types.Bool   `tfsdk:"draft"`
	SourceArtifacts types.List   `tfsdk:"source_artifacts"`
	SourceBuilds    types.List   `tfsdk:"source_builds"`
	SourceVersions  types.List   `tfsdk:"source_versions"`
	SourceReleaseBundles         types.List `tfsdk:"source_release_bundles"`
	SourcePackages               types.List `tfsdk:"source_packages"`
	SourceAql                    types.String `tfsdk:"source_aql"`
	SkipDockerManifestResolution types.Bool   `tfsdk:"skip_docker_manifest_resolution"`
	FilterIncluded               types.List `tfsdk:"filter_included"`
	FilterExcluded               types.List `tfsdk:"filter_excluded"`
	// UpdateAppVersionRequest: optional properties and delete_properties
	Properties       types.Map  `tfsdk:"properties"`
	DeleteProperties types.List `tfsdk:"delete_properties"`
	// Computed from API
	ReleaseStatus types.String `tfsdk:"release_status"`
	CurrentStage  types.String `tfsdk:"current_stage"`
}

type applicationVersionSourceArtifact struct {
	Path   string `json:"path"`
	Sha256 string `json:"sha256,omitempty"`
}

type applicationVersionSourceBuild struct {
	Name                string `json:"name"`
	Number              string `json:"number"`
	IncludeDependencies bool   `json:"include_dependencies,omitempty"`
	RepositoryKey       string `json:"repository_key,omitempty"`
	Started             string `json:"started,omitempty"`
}

type createApplicationVersionBody struct {
	Version string                           `json:"version"`
	Sources createApplicationVersionSources  `json:"sources"`
	Tag     string                           `json:"tag,omitempty"`
	Draft   bool                             `json:"draft,omitempty"`
	Filters *createApplicationVersionFilters `json:"filters,omitempty"`
}

type applicationVersionSourceVersion struct {
	ApplicationKey string `json:"application_key"`
	Version        string `json:"version"`
}

type applicationVersionSourceReleaseBundle struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	ProjectKey    string `json:"project_key,omitempty"`
	RepositoryKey string `json:"repository_key,omitempty"`
}

type applicationVersionSourcePackage struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Version string `json:"version,omitempty"`
}

type applicationVersionFilterItem struct {
	Path           string `json:"path,omitempty"`
	Sha256         string `json:"sha256,omitempty"`
	PackageName    string `json:"package_name,omitempty"`
	PackageType    string `json:"package_type,omitempty"`
	PackageVersion string `json:"package_version,omitempty"`
}

type createApplicationVersionFilters struct {
	Included []applicationVersionFilterItem `json:"included,omitempty"`
	Excluded []applicationVersionFilterItem `json:"excluded,omitempty"`
}

type createApplicationVersionSources struct {
	Artifacts                    []applicationVersionSourceArtifact    `json:"artifacts,omitempty"`
	Builds                       []applicationVersionSourceBuild       `json:"builds,omitempty"`
	Versions                     []applicationVersionSourceVersion     `json:"versions,omitempty"`
	ReleaseBundles               []applicationVersionSourceReleaseBundle `json:"release_bundles,omitempty"`
	Packages                     []applicationVersionSourcePackage     `json:"packages,omitempty"`
	Aql                          string                                `json:"aql,omitempty"`
	SkipDockerManifestResolution bool                                  `json:"skip_docker_manifest_resolution,omitempty"`
}

type applicationVersionListItem struct {
	Version       string `json:"version"`
	Tag           string `json:"tag"`
	Status        string `json:"status"`
	ReleaseStatus string `json:"release_status"`
	CurrentStage  string `json:"current_stage"`
	CreatedBy     string `json:"created_by"`
	Created       string `json:"created"`
}

type applicationVersionsListResponse struct {
	Versions []applicationVersionListItem `json:"versions"`
	Total    int                          `json:"total"`
	Limit    int                          `json:"limit"`
	Offset   int                          `json:"offset"`
}

func (r *ApplicationVersionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *ApplicationVersionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Provides an AppTrust application version resource. Creates, updates (tag), and deletes an application version. " +
			"At least one source (artifacts or builds) must be provided on create.",
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
				Description: "The application version (e.g. SemVer 1.0.0).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"tag": schema.StringAttribute{
				Description: "Tag associated with the version (e.g. branch name). Max 128 characters.",
				Optional:    true,
			},
			"draft": schema.BoolAttribute{
				Description: "Whether to create the version as a draft.",
				Optional:    true,
			},
			"source_artifacts": schema.ListNestedAttribute{
				Description: "Artifact paths to include in the version. At least one source (artifacts, builds, or source_versions) required on create.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"path": schema.StringAttribute{
							Description: "Path to the artifact in the repository.",
							Required:    true,
						},
						"sha256": schema.StringAttribute{
							Description: "SHA256 checksum of the artifact (optional).",
							Optional:    true,
						},
					},
				},
			},
			"source_builds": schema.ListNestedAttribute{
				Description: "Builds to include in the version. At least one source (artifacts, builds, or source_versions) required on create.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Build name.",
							Required:    true,
						},
						"number": schema.StringAttribute{
							Description: "Build number.",
							Required:    true,
						},
						"include_dependencies": schema.BoolAttribute{
							Description: "Include build dependencies.",
							Optional:    true,
						},
						"repository_key": schema.StringAttribute{
							Description: "Build-info repository key.",
							Optional:    true,
						},
						"started": schema.StringAttribute{
							Description: "Build timestamp (ISO 8601).",
							Optional:    true,
						},
					},
				},
			},
			"source_versions": schema.ListNestedAttribute{
				Description: "Other application versions to include as sources (CreateAppVersionVersionsSources).",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"application_key": schema.StringAttribute{
							Description: "Application key of the source version.",
							Required:    true,
						},
						"version": schema.StringAttribute{
							Description: "Version of the source application.",
							Required:    true,
						},
					},
				},
			},
			"source_release_bundles": schema.ListNestedAttribute{
				Description: "Release bundles to include as sources.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Release bundle name.",
							Required:    true,
						},
						"version": schema.StringAttribute{
							Description: "Release bundle version.",
							Required:    true,
						},
						"project_key": schema.StringAttribute{
							Description: "Project key for the release bundle.",
							Optional:    true,
						},
						"repository_key": schema.StringAttribute{
							Description: "Repository key for the release bundle.",
							Optional:    true,
						},
					},
				},
			},
			"source_packages": schema.ListNestedAttribute{
				Description: "Packages to include as sources.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Package name.",
							Required:    true,
						},
						"type": schema.StringAttribute{
							Description: "Package type (e.g. docker, npm, maven).",
							Required:    true,
						},
						"version": schema.StringAttribute{
							Description: "Package version.",
							Optional:    true,
						},
					},
				},
			},
			"source_aql": schema.StringAttribute{
				Description: "AQL query to resolve source artifacts.",
				Optional:    true,
			},
			"skip_docker_manifest_resolution": schema.BoolAttribute{
				Description: "Skip Docker manifest resolution for source artifacts.",
				Optional:    true,
			},
			"filter_included": schema.ListNestedAttribute{
				Description: "Artifact filter — only include matching items.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"path": schema.StringAttribute{
							Description: "Artifact path.",
							Optional:    true,
						},
						"sha256": schema.StringAttribute{
							Description: "Artifact SHA256 checksum.",
							Optional:    true,
						},
						"package_name": schema.StringAttribute{
							Description: "Package name.",
							Optional:    true,
						},
						"package_type": schema.StringAttribute{
							Description: "Package type.",
							Optional:    true,
						},
						"package_version": schema.StringAttribute{
							Description: "Package version.",
							Optional:    true,
						},
					},
				},
			},
			"filter_excluded": schema.ListNestedAttribute{
				Description: "Artifact filter — exclude matching items.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"path": schema.StringAttribute{
							Description: "Artifact path.",
							Optional:    true,
						},
						"sha256": schema.StringAttribute{
							Description: "Artifact SHA256 checksum.",
							Optional:    true,
						},
						"package_name": schema.StringAttribute{
							Description: "Package name.",
							Optional:    true,
						},
						"package_type": schema.StringAttribute{
							Description: "Package type.",
							Optional:    true,
						},
						"package_version": schema.StringAttribute{
							Description: "Package version.",
							Optional:    true,
						},
					},
				},
			},
			"properties": schema.MapAttribute{
				Description: "Version properties (key -> list of values). UpdateAppVersionRequest.",
				ElementType: types.ListType{ElemType: types.StringType},
				Optional:    true,
			},
			"delete_properties": schema.ListAttribute{
				Description: "Property keys to remove on update.",
				ElementType: types.StringType,
				Optional:    true,
			},
			"release_status": schema.StringAttribute{
				Description: "Release status: pre_release, released, trusted_release. Computed from API.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"current_stage": schema.StringAttribute{
				Description: "Current lifecycle stage. Computed from API.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ApplicationVersionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (r *ApplicationVersionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan ApplicationVersionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sources := createApplicationVersionSources{}
	hasAnySource := false
	if !plan.SourceArtifacts.IsNull() && !plan.SourceArtifacts.IsUnknown() {
		var list []struct {
			Path   string       `tfsdk:"path"`
			Sha256 types.String `tfsdk:"sha256"`
		}
		resp.Diagnostics.Append(plan.SourceArtifacts.ElementsAs(ctx, &list, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, e := range list {
			sha256 := ""
			if !e.Sha256.IsNull() {
				sha256 = e.Sha256.ValueString()
			}
			sources.Artifacts = append(sources.Artifacts, applicationVersionSourceArtifact{Path: e.Path, Sha256: sha256})
		}
		hasAnySource = len(sources.Artifacts) > 0
	}
	if !plan.SourceBuilds.IsNull() && !plan.SourceBuilds.IsUnknown() {
		var list []struct {
			Name                string       `tfsdk:"name"`
			Number              string       `tfsdk:"number"`
			IncludeDependencies types.Bool   `tfsdk:"include_dependencies"`
			RepositoryKey       types.String `tfsdk:"repository_key"`
			Started             types.String `tfsdk:"started"`
		}
		resp.Diagnostics.Append(plan.SourceBuilds.ElementsAs(ctx, &list, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, e := range list {
			includeDeps := false
			if !e.IncludeDependencies.IsNull() {
				includeDeps = e.IncludeDependencies.ValueBool()
			}
			repoKey := ""
			if !e.RepositoryKey.IsNull() {
				repoKey = e.RepositoryKey.ValueString()
			}
			started := ""
			if !e.Started.IsNull() {
				started = e.Started.ValueString()
			}
			sources.Builds = append(sources.Builds, applicationVersionSourceBuild{
				Name:                e.Name,
				Number:              e.Number,
				IncludeDependencies: includeDeps,
				RepositoryKey:       repoKey,
				Started:             started,
			})
		}
		hasAnySource = hasAnySource || len(sources.Builds) > 0
	}
	if !plan.SourceVersions.IsNull() && !plan.SourceVersions.IsUnknown() {
		var list []struct {
			ApplicationKey string `tfsdk:"application_key"`
			Version        string `tfsdk:"version"`
		}
		resp.Diagnostics.Append(plan.SourceVersions.ElementsAs(ctx, &list, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, e := range list {
			sources.Versions = append(sources.Versions, applicationVersionSourceVersion{ApplicationKey: e.ApplicationKey, Version: e.Version})
		}
		hasAnySource = hasAnySource || len(sources.Versions) > 0
	}
	if !plan.SourceReleaseBundles.IsNull() && !plan.SourceReleaseBundles.IsUnknown() {
		var list []struct {
			Name          string       `tfsdk:"name"`
			Version       string       `tfsdk:"version"`
			ProjectKey    types.String `tfsdk:"project_key"`
			RepositoryKey types.String `tfsdk:"repository_key"`
		}
		resp.Diagnostics.Append(plan.SourceReleaseBundles.ElementsAs(ctx, &list, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, e := range list {
			sources.ReleaseBundles = append(sources.ReleaseBundles, applicationVersionSourceReleaseBundle{
				Name:          e.Name,
				Version:       e.Version,
				ProjectKey:    e.ProjectKey.ValueString(),
				RepositoryKey: e.RepositoryKey.ValueString(),
			})
		}
		hasAnySource = hasAnySource || len(sources.ReleaseBundles) > 0
	}
	if !plan.SourcePackages.IsNull() && !plan.SourcePackages.IsUnknown() {
		var list []struct {
			Name    string       `tfsdk:"name"`
			Type    string       `tfsdk:"type"`
			Version types.String `tfsdk:"version"`
		}
		resp.Diagnostics.Append(plan.SourcePackages.ElementsAs(ctx, &list, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, e := range list {
			sources.Packages = append(sources.Packages, applicationVersionSourcePackage{
				Name:    e.Name,
				Type:    e.Type,
				Version: e.Version.ValueString(),
			})
		}
		hasAnySource = hasAnySource || len(sources.Packages) > 0
	}
	if !plan.SourceAql.IsNull() && !plan.SourceAql.IsUnknown() && plan.SourceAql.ValueString() != "" {
		sources.Aql = plan.SourceAql.ValueString()
		hasAnySource = true
	}
	if !plan.SkipDockerManifestResolution.IsNull() && !plan.SkipDockerManifestResolution.IsUnknown() {
		sources.SkipDockerManifestResolution = plan.SkipDockerManifestResolution.ValueBool()
	}
	if !hasAnySource {
		resp.Diagnostics.AddError(
			"At least one source required",
			"Create application version requires at least one of source_artifacts, source_builds, source_versions, source_release_bundles, source_packages, or source_aql.",
		)
		return
	}

	// Build optional filters
	var filters *createApplicationVersionFilters
	filterItemsFromPlan := func(list types.List) ([]applicationVersionFilterItem, bool) {
		if list.IsNull() || list.IsUnknown() {
			return nil, true
		}
		var items []struct {
			Path           types.String `tfsdk:"path"`
			Sha256         types.String `tfsdk:"sha256"`
			PackageName    types.String `tfsdk:"package_name"`
			PackageType    types.String `tfsdk:"package_type"`
			PackageVersion types.String `tfsdk:"package_version"`
		}
		if diags := list.ElementsAs(ctx, &items, false); diags.HasError() {
			return nil, false
		}
		result := make([]applicationVersionFilterItem, 0, len(items))
		for _, e := range items {
			result = append(result, applicationVersionFilterItem{
				Path:           e.Path.ValueString(),
				Sha256:         e.Sha256.ValueString(),
				PackageName:    e.PackageName.ValueString(),
				PackageType:    e.PackageType.ValueString(),
				PackageVersion: e.PackageVersion.ValueString(),
			})
		}
		return result, true
	}

	if !plan.FilterIncluded.IsNull() || !plan.FilterExcluded.IsNull() {
		filters = &createApplicationVersionFilters{}
		included, ok := filterItemsFromPlan(plan.FilterIncluded)
		if !ok {
			resp.Diagnostics.AddError("Error reading filter_included", "Failed to parse filter_included")
			return
		}
		filters.Included = included
		excluded, ok := filterItemsFromPlan(plan.FilterExcluded)
		if !ok {
			resp.Diagnostics.AddError("Error reading filter_excluded", "Failed to parse filter_excluded")
			return
		}
		filters.Excluded = excluded
	}

	body := createApplicationVersionBody{
		Version: plan.Version.ValueString(),
		Sources: sources,
		Tag:     plan.Tag.ValueString(),
		Filters: filters,
	}
	if !plan.Draft.IsNull() && !plan.Draft.IsUnknown() {
		body.Draft = plan.Draft.ValueBool()
	}

	httpResponse, err := r.ProviderData.Client.R().
		SetContext(ctx).
		SetPathParam("application_key", plan.ApplicationKey.ValueString()).
		SetBody(body).
		Post(ApplicationVersionsEndpoint)

	if err != nil {
		tflog.Error(ctx, "Failed to create application version", map[string]interface{}{
			"application_key": plan.ApplicationKey.ValueString(),
			"version":         plan.Version.ValueString(),
			"error":           err.Error(),
		})
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	if httpResponse.StatusCode() != http.StatusCreated && httpResponse.StatusCode() != http.StatusAccepted {
		errorDiags := apptrust.HandleAPIErrorWithType(httpResponse, "create", "application version")
		resp.Diagnostics.Append(errorDiags...)
		return
	}

	// After create (sync or async), fetch the version list to populate computed fields
	// (release_status, current_stage). These are not returned in the create response body.
	var listResp applicationVersionsListResponse
	readResp, readErr := r.ProviderData.Client.R().
		SetContext(ctx).
		SetPathParam("application_key", plan.ApplicationKey.ValueString()).
		SetQueryParam("limit", "1000").
		SetResult(&listResp).
		Get(ApplicationVersionsEndpoint)

	if readErr == nil && readResp.StatusCode() == http.StatusOK {
		for i := range listResp.Versions {
			if listResp.Versions[i].Version == plan.Version.ValueString() {
				plan.ReleaseStatus = types.StringValue(listResp.Versions[i].ReleaseStatus)
				plan.CurrentStage = types.StringValue(listResp.Versions[i].CurrentStage)
				// Preserve the tag from API response (may differ if plan tag was empty)
				if listResp.Versions[i].Tag != "" {
					plan.Tag = types.StringValue(listResp.Versions[i].Tag)
				}
				break
			}
		}
	} else {
		// If the read-back fails, use empty strings so state is valid (not unknown).
		if plan.ReleaseStatus.IsUnknown() {
			plan.ReleaseStatus = types.StringValue("")
		}
		if plan.CurrentStage.IsUnknown() {
			plan.CurrentStage = types.StringValue("")
		}
	}

	plan.ID = types.StringValue(plan.ApplicationKey.ValueString() + ":" + plan.Version.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ApplicationVersionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ApplicationVersionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationKey := state.ApplicationKey.ValueString()
	version := state.Version.ValueString()
	if applicationKey == "" || version == "" {
		if id := state.ID.ValueString(); id != "" {
			// Import: id is "application_key:version"
			for i, c := range id {
				if c == ':' {
					applicationKey = id[:i]
					version = id[i+1:]
					break
				}
			}
		}
	}
	if applicationKey == "" || version == "" {
		resp.Diagnostics.AddError("Missing application_key or version", "id must be application_key:version or state must have application_key and version")
		return
	}

	var listResp applicationVersionsListResponse
	httpResponse, err := r.ProviderData.Client.R().
		SetContext(ctx).
		SetPathParam("application_key", applicationKey).
		SetQueryParam("limit", "1000").
		SetResult(&listResp).
		Get(ApplicationVersionsEndpoint)

	if err != nil {
		utilfw.UnableToRefreshResourceError(resp, err.Error())
		return
	}

	if httpResponse.StatusCode() != http.StatusOK {
		if httpResponse.StatusCode() == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		errorDiags := apptrust.HandleAPIErrorWithType(httpResponse, "read", "application version")
		resp.Diagnostics.Append(errorDiags...)
		return
	}

	var found *applicationVersionListItem
	for i := range listResp.Versions {
		if listResp.Versions[i].Version == version {
			found = &listResp.Versions[i]
			break
		}
	}
	if found == nil {
		tflog.Warn(ctx, "Application version not found, removing from state", map[string]interface{}{
			"application_key": applicationKey,
			"version":         version,
		})
		resp.State.RemoveResource(ctx)
		return
	}

	state.ApplicationKey = types.StringValue(applicationKey)
	state.Version = types.StringValue(version)
	if found.Tag != "" {
		state.Tag = types.StringValue(found.Tag)
	} else {
		state.Tag = types.StringNull()
	}
	state.ReleaseStatus = types.StringValue(found.ReleaseStatus)
	state.CurrentStage = types.StringValue(found.CurrentStage)
	state.ID = types.StringValue(applicationKey + ":" + version)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ApplicationVersionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	go util.SendUsageResourceUpdate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan ApplicationVersionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"tag": plan.Tag.ValueString(),
	}
	if !plan.Properties.IsNull() && !plan.Properties.IsUnknown() {
		props := make(map[string][]string)
		for k, v := range plan.Properties.Elements() {
			listVal, ok := v.(types.List)
			if !ok {
				continue
			}
			var strs []string
			if diags := listVal.ElementsAs(ctx, &strs, false); diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
			props[k] = strs
		}
		body["properties"] = props
	}
	if !plan.DeleteProperties.IsNull() && !plan.DeleteProperties.IsUnknown() {
		var del []string
		resp.Diagnostics.Append(plan.DeleteProperties.ElementsAs(ctx, &del, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		body["delete_properties"] = del
	}

	httpResponse, err := r.ProviderData.Client.R().
		SetContext(ctx).
		SetPathParam("application_key", plan.ApplicationKey.ValueString()).
		SetPathParam("version", plan.Version.ValueString()).
		SetBody(body).
		Patch(ApplicationVersionEndpoint)

	if err != nil {
		utilfw.UnableToUpdateResourceError(resp, err.Error())
		return
	}

	if httpResponse.StatusCode() != http.StatusOK && httpResponse.StatusCode() != http.StatusAccepted {
		errorDiags := apptrust.HandleAPIErrorWithType(httpResponse, "update", "application version")
		resp.Diagnostics.Append(errorDiags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ApplicationVersionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state ApplicationVersionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationKey := state.ApplicationKey.ValueString()
	version := state.Version.ValueString()
	if applicationKey == "" || version == "" {
		id := state.ID.ValueString()
		for i, c := range id {
			if c == ':' {
				applicationKey = id[:i]
				version = id[i+1:]
				break
			}
		}
	}

	httpResponse, err := r.ProviderData.Client.R().
		SetContext(ctx).
		SetPathParam("application_key", applicationKey).
		SetPathParam("version", version).
		Delete(ApplicationVersionEndpoint)

	if err != nil {
		utilfw.UnableToDeleteResourceError(resp, err.Error())
		return
	}

	if httpResponse.StatusCode() != http.StatusNoContent && httpResponse.StatusCode() != http.StatusOK && httpResponse.StatusCode() != http.StatusAccepted {
		if httpResponse.StatusCode() == http.StatusNotFound {
			return
		}
		errorDiags := apptrust.HandleAPIErrorWithType(httpResponse, "delete", "application version")
		resp.Diagnostics.Append(errorDiags...)
	}
}

func (r *ApplicationVersionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// ID format: application_key:version
	id := req.ID
	for i, c := range id {
		if c == ':' {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_key"), id[:i])...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("version"), id[i+1:])...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
			return
		}
	}
	resp.Diagnostics.AddError("Invalid import ID", "Import ID must be application_key:version (e.g. my-app:1.0.0)")
}
