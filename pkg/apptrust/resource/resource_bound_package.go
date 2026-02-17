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
	"strings"

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
	ApplicationPackagesEndpoint        = ApplicationEndpoint + "/packages"
	ApplicationPackageVersionsEndpoint = ApplicationPackagesEndpoint + "/{type}/{name}"
	ApplicationPackageVersionEndpoint  = ApplicationPackagesEndpoint + "/{type}/{name}/{version}"
)

var _ resource.Resource = &BoundPackageResource{}

func NewBoundPackageResource() resource.Resource {
	return &BoundPackageResource{
		TypeName: "apptrust_bound_package",
	}
}

type BoundPackageResource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type BoundPackageResourceModel struct {
	ID             types.String `tfsdk:"id"`
	ApplicationKey types.String `tfsdk:"application_key"`
	PackageType    types.String `tfsdk:"package_type"`
	PackageName    types.String `tfsdk:"package_name"`
	PackageVersion types.String `tfsdk:"package_version"`
}

type bindPackageRequestBody struct {
	PackageType    string `json:"package_type"`
	PackageName    string `json:"package_name"`
	PackageVersion string `json:"package_version"`
}

func (r *BoundPackageResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = r.TypeName
}

func (r *BoundPackageResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Binds a package version to an AppTrust application. " +
			"A package version can be bound to only one application.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Computed ID (application_key:type:name:version).",
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
			"package_type": schema.StringAttribute{
				Description: "Package type (e.g. maven, docker, npm, generic).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"package_name": schema.StringAttribute{
				Description: "Package name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"package_version": schema.StringAttribute{
				Description: "Package version.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *BoundPackageResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func boundPackageID(appKey, pkgType, name, version string) string {
	return fmt.Sprintf("%s:%s:%s:%s", appKey, pkgType, name, version)
}

func (r *BoundPackageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	go util.SendUsageResourceCreate(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var plan BoundPackageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := bindPackageRequestBody{
		PackageType:    plan.PackageType.ValueString(),
		PackageName:    plan.PackageName.ValueString(),
		PackageVersion: plan.PackageVersion.ValueString(),
	}

	httpResponse, err := r.ProviderData.Client.R().
		SetContext(ctx).
		SetPathParam("application_key", plan.ApplicationKey.ValueString()).
		SetBody(body).
		Post(ApplicationPackagesEndpoint)

	if err != nil {
		tflog.Error(ctx, "Failed to bind package", map[string]interface{}{"error": err.Error()})
		utilfw.UnableToCreateResourceError(resp, err.Error())
		return
	}

	if httpResponse.StatusCode() != http.StatusCreated {
		errorDiags := apptrust.HandleAPIErrorWithType(httpResponse, "create", "bound package")
		resp.Diagnostics.Append(errorDiags...)
		return
	}

	plan.ID = types.StringValue(boundPackageID(
		plan.ApplicationKey.ValueString(),
		plan.PackageType.ValueString(),
		plan.PackageName.ValueString(),
		plan.PackageVersion.ValueString(),
	))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BoundPackageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state BoundPackageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appKey := state.ApplicationKey.ValueString()
	pkgType := state.PackageType.ValueString()
	name := state.PackageName.ValueString()
	version := state.PackageVersion.ValueString()
	if appKey == "" || pkgType == "" || name == "" || version == "" {
		// Parse from id: application_key:type:name:version (name may contain colons e.g. maven group:artifact)
		id := state.ID.ValueString()
		parts := splitBoundPackageID(id)
		if len(parts) >= 4 {
			appKey = parts[0]
			pkgType = parts[1]
			name = parts[2]
			version = parts[3]
		}
	}
	if appKey == "" || pkgType == "" || name == "" || version == "" {
		resp.Diagnostics.AddError("Missing attributes", "application_key, package_type, package_name, package_version or valid id required")
		return
	}

	// Verify binding exists by checking if this version is in the bound package versions list
	var listResp struct {
		Versions []struct {
			Version string `json:"version"`
		} `json:"versions"`
	}
	httpResponse, err := r.ProviderData.Client.R().
		SetContext(ctx).
		SetPathParam("application_key", appKey).
		SetPathParam("type", pkgType).
		SetPathParam("name", name).
		SetQueryParam("package_version", version).
		SetResult(&listResp).
		Get(ApplicationPackageVersionsEndpoint)

	if err != nil {
		utilfw.UnableToRefreshResourceError(resp, err.Error())
		return
	}

	if httpResponse.StatusCode() != http.StatusOK {
		if httpResponse.StatusCode() == http.StatusNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		errorDiags := apptrust.HandleAPIErrorWithType(httpResponse, "read", "bound package")
		resp.Diagnostics.Append(errorDiags...)
		return
	}

	found := false
	for _, v := range listResp.Versions {
		if v.Version == version {
			found = true
			break
		}
	}
	if !found {
		tflog.Warn(ctx, "Bound package not found, removing from state", map[string]interface{}{
			"application_key": appKey, "package_type": pkgType, "package_name": name, "package_version": version,
		})
		resp.State.RemoveResource(ctx)
		return
	}

	state.ApplicationKey = types.StringValue(appKey)
	state.PackageType = types.StringValue(pkgType)
	state.PackageName = types.StringValue(name)
	state.PackageVersion = types.StringValue(version)
	state.ID = types.StringValue(boundPackageID(appKey, pkgType, name, version))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// splitBoundPackageID splits id "appKey:type:name:version" where name may contain colons (e.g. maven group:artifact).
// We split from the end: last part is version, second-to-last is the last segment of name, etc.
// Simplest: split by ":" and take first as appKey, second as type, last as version, and join the rest as name.
func splitBoundPackageID(id string) []string {
	var parts []string
	var cur string
	for _, c := range id {
		if c == ':' {
			if cur != "" {
				parts = append(parts, cur)
				cur = ""
			}
		} else {
			cur += string(c)
		}
	}
	if cur != "" {
		parts = append(parts, cur)
	}
	return parts
}

func (r *BoundPackageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BoundPackageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// No updatable attributes; all require replace.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BoundPackageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	go util.SendUsageResourceDelete(ctx, r.ProviderData.Client.R(), r.ProviderData.ProductId, r.TypeName)

	var state BoundPackageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appKey := state.ApplicationKey.ValueString()
	pkgType := state.PackageType.ValueString()
	name := state.PackageName.ValueString()
	version := state.PackageVersion.ValueString()
	if appKey == "" || pkgType == "" || name == "" || version == "" {
		parts := splitBoundPackageID(state.ID.ValueString())
		if len(parts) >= 4 {
			appKey = parts[0]
			pkgType = parts[1]
			name = parts[2]
			version = parts[3]
		}
	}

	httpResponse, err := r.ProviderData.Client.R().
		SetContext(ctx).
		SetPathParam("application_key", appKey).
		SetPathParam("type", pkgType).
		SetPathParam("name", name).
		SetPathParam("version", version).
		Delete(ApplicationPackageVersionEndpoint)

	if err != nil {
		utilfw.UnableToDeleteResourceError(resp, err.Error())
		return
	}

	if httpResponse.StatusCode() != http.StatusNoContent && httpResponse.StatusCode() != http.StatusOK {
		if httpResponse.StatusCode() == http.StatusNotFound {
			return
		}
		errorDiags := apptrust.HandleAPIErrorWithType(httpResponse, "delete", "bound package")
		resp.Diagnostics.Append(errorDiags...)
	}
}

func (r *BoundPackageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := splitBoundPackageID(req.ID)
	if len(parts) < 4 {
		resp.Diagnostics.AddError("Invalid import ID", "Use application_key:package_type:package_name:package_version (e.g. my-app:maven:com.example:lib:1.0.0)")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_key"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("package_type"), parts[1])...)
	// package_name may contain colons (e.g. maven group:artifact)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("package_name"), strings.Join(parts[2:len(parts)-1], ":"))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("package_version"), parts[len(parts)-1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
