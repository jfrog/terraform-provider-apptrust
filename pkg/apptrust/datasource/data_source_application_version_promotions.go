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

package datasource

import (
	"context"
	"net/http"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jfrog/terraform-provider-apptrust/pkg/apptrust"
	"github.com/jfrog/terraform-provider-apptrust/pkg/apptrust/resource"
	"github.com/jfrog/terraform-provider-shared/util"
)

var _ datasource.DataSource = &ApplicationVersionPromotionsDataSource{}

func NewApplicationVersionPromotionsDataSource() datasource.DataSource {
	return &ApplicationVersionPromotionsDataSource{}
}

type ApplicationVersionPromotionsDataSource struct {
	ProviderData util.ProviderMetadata
}

type ApplicationVersionPromotionsDataSourceModel struct {
	ApplicationKey types.String `tfsdk:"application_key"`
	Version        types.String `tfsdk:"version"`
	Include        types.String `tfsdk:"include"`
	Offset         types.Int64  `tfsdk:"offset"`
	Limit          types.Int64  `tfsdk:"limit"`
	FilterBy       types.String `tfsdk:"filter_by"`
	OrderBy        types.String `tfsdk:"order_by"`
	OrderAsc       types.Bool   `tfsdk:"order_asc"`
	Promotions     types.List   `tfsdk:"promotions"`
	Total          types.Int64  `tfsdk:"total"`
}

type promotionMessageAPIModel struct {
	Text string `json:"text"`
}

type promotionRecordAPIModel struct {
	ApplicationKey     string                     `json:"application_key"`
	ApplicationVersion string                     `json:"application_version"`
	Created            string                     `json:"created"`
	CreatedBy          string                     `json:"created_by"`
	CreatedMillis      int64                      `json:"created_millis"`
	Messages           []promotionMessageAPIModel `json:"messages"`
	ProjectKey         string                     `json:"project_key"`
	SourceStage        string                     `json:"source_stage"`
	Status             string                     `json:"status"`
	TargetStage        string                     `json:"target_stage"`
}

type promotionsListAPIModel struct {
	Promotions []promotionRecordAPIModel `json:"promotions"`
	Total      int                       `json:"total"`
	Limit      int                       `json:"limit"`
	Offset     int                       `json:"offset"`
}

var promotionRecordAttrType = map[string]attr.Type{
	"application_key":     types.StringType,
	"application_version": types.StringType,
	"created":             types.StringType,
	"created_by":          types.StringType,
	"created_millis":      types.Int64Type,
	"messages":            types.ListType{ElemType: types.StringType},
	"project_key":         types.StringType,
	"source_stage":        types.StringType,
	"status":              types.StringType,
	"target_stage":        types.StringType,
}

func (d *ApplicationVersionPromotionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_version_promotions"
}

func (d *ApplicationVersionPromotionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Returns the list of promotions for a specific application version (GET /v1/applications/{application_key}/versions/{version}/promotions).",
		Attributes: map[string]schema.Attribute{
			"application_key": schema.StringAttribute{
				Description: "The application key.",
				Required:    true,
			},
			"version": schema.StringAttribute{
				Description: "The application version.",
				Required:    true,
			},
			"include": schema.StringAttribute{
				Description: "When set to message, returns any error messages from the promotion operation.",
				Optional:    true,
			},
			"offset": schema.Int64Attribute{
				Description: "Number of records to skip (pagination).",
				Optional:    true,
			},
			"limit": schema.Int64Attribute{
				Description: "Maximum number of promotions to return.",
				Optional:    true,
			},
			"filter_by": schema.StringAttribute{
				Description: "Filter by application_version, target_stage, promoted_by, or status (success, pending, failed).",
				Optional:    true,
			},
			"order_by": schema.StringAttribute{
				Description: "Order by: created, created_by, version, stage. Default is created.",
				Optional:    true,
			},
			"order_asc": schema.BoolAttribute{
				Description: "Sort ascending (true) or descending (false). Default false.",
				Optional:    true,
			},
			"promotions": schema.ListAttribute{
				Description: "List of promotion records.",
				Computed:    true,
				ElementType: types.ObjectType{AttrTypes: promotionRecordAttrType},
			},
			"total": schema.Int64Attribute{
				Description: "Total number of promotions.",
				Computed:    true,
			},
		},
	}
}

func (d *ApplicationVersionPromotionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (d *ApplicationVersionPromotionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ApplicationVersionPromotionsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationKey := data.ApplicationKey.ValueString()
	version := data.Version.ValueString()
	tflog.Info(ctx, "Reading application version promotions", map[string]interface{}{
		"application_key": applicationKey,
		"version":         version,
	})

	var apiResp promotionsListAPIModel
	httpReq := d.ProviderData.Client.R().
		SetContext(ctx).
		SetPathParam("application_key", applicationKey).
		SetPathParam("version", version).
		SetResult(&apiResp)

	if !data.Include.IsNull() && !data.Include.IsUnknown() {
		httpReq.SetQueryParam("include", data.Include.ValueString())
	}
	if !data.Offset.IsNull() && !data.Offset.IsUnknown() {
		httpReq.SetQueryParam("offset", strconv.FormatInt(data.Offset.ValueInt64(), 10))
	}
	if !data.Limit.IsNull() && !data.Limit.IsUnknown() {
		httpReq.SetQueryParam("limit", strconv.FormatInt(data.Limit.ValueInt64(), 10))
	}
	if !data.FilterBy.IsNull() && !data.FilterBy.IsUnknown() {
		httpReq.SetQueryParam("filter_by", data.FilterBy.ValueString())
	}
	if !data.OrderBy.IsNull() && !data.OrderBy.IsUnknown() {
		httpReq.SetQueryParam("order_by", data.OrderBy.ValueString())
	}
	if !data.OrderAsc.IsNull() && !data.OrderAsc.IsUnknown() {
		httpReq.SetQueryParam("order_asc", strconv.FormatBool(data.OrderAsc.ValueBool()))
	}

	httpResponse, err := httpReq.Get(resource.ApplicationVersionPromotionsEP)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read Data Source", "Error: "+err.Error())
		return
	}

	if httpResponse.StatusCode() != http.StatusOK {
		diags := apptrust.HandleAPIErrorWithType(httpResponse, "read", "application version promotions")
		resp.Diagnostics.Append(diags...)
		return
	}

	elems := make([]attr.Value, 0, len(apiResp.Promotions))
	for _, p := range apiResp.Promotions {
		msgStrs := make([]attr.Value, 0, len(p.Messages))
		for _, m := range p.Messages {
			msgStrs = append(msgStrs, types.StringValue(m.Text))
		}
		obj, diags := types.ObjectValue(promotionRecordAttrType, map[string]attr.Value{
			"application_key":     types.StringValue(p.ApplicationKey),
			"application_version": types.StringValue(p.ApplicationVersion),
			"created":             types.StringValue(p.Created),
			"created_by":          types.StringValue(p.CreatedBy),
			"created_millis":      types.Int64Value(p.CreatedMillis),
			"messages":            types.ListValueMust(types.StringType, msgStrs),
			"project_key":         types.StringValue(p.ProjectKey),
			"source_stage":        types.StringValue(p.SourceStage),
			"status":              types.StringValue(p.Status),
			"target_stage":        types.StringValue(p.TargetStage),
		})
		resp.Diagnostics.Append(diags...)
		if diags.HasError() {
			return
		}
		elems = append(elems, obj)
	}

	data.Promotions = types.ListValueMust(types.ObjectType{AttrTypes: promotionRecordAttrType}, elems)
	data.Total = types.Int64Value(int64(apiResp.Total))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
