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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/jfrog/terraform-provider-shared/util"
)

const (
	activityLogEndpoint = "apptrust/api/v1/activity/log"
)

var (
	activityLogSortByOptions = []string{"timestamp", "event_id", "subject_type", "subject_name", "event_type", "event_category", "project_key", "created_by"}
	activityLogSortOptions   = []string{"asc", "desc"}
	activityLogResultOptions = []string{"success", "failure", "warning"}
)

var _ datasource.DataSource = &ActivityLogDataSource{}

func NewActivityLogDataSource() datasource.DataSource {
	return &ActivityLogDataSource{}
}

type ActivityLogDataSource struct {
	ProviderData util.ProviderMetadata
}

type ActivityLogDataSourceModel struct {
	ApplicationKey  types.List   `tfsdk:"application_key"`
	ProjectKey      types.List   `tfsdk:"project_key"`
	ApplicationName types.List   `tfsdk:"application_name"`
	ProjectName     types.List   `tfsdk:"project_name"`
	TimestampFrom   types.Int64  `tfsdk:"timestamp_from"`
	TimestampTo     types.Int64  `tfsdk:"timestamp_to"`
	SubjectType     types.List   `tfsdk:"subject_type"`
	SubjectId       types.String `tfsdk:"subject_id"`
	SubjectName     types.String `tfsdk:"subject_name"`
	EventType       types.List   `tfsdk:"event_type"`
	EventCategory   types.List   `tfsdk:"event_category"`
	Result          types.List   `tfsdk:"result"`
	Prefix          types.String `tfsdk:"prefix"`
	CreatedBy       types.List   `tfsdk:"created_by"`
	Limit           types.Int64  `tfsdk:"limit"`
	Offset          types.Int64  `tfsdk:"offset"`
	SortBy          types.String `tfsdk:"sort_by"`
	Sort            types.String `tfsdk:"sort"`
	ActivityLogs    types.List   `tfsdk:"activity_logs"`
	Total           types.Int64  `tfsdk:"total"`
}

// activityLogEntryAPIResponse matches a single entry in the "activity_logs" array
// returned by GET /v1/activity/log.
type activityLogEntryAPIResponse struct {
	EventId          string          `json:"event_id"`
	Timestamp        int64           `json:"timestamp"`
	SubjectType      string          `json:"subject_type"`
	SubjectId        string          `json:"subject_id,omitempty"`
	SubjectName      string          `json:"subject_name"`
	EventDescription string          `json:"event_description"`
	EventType        string          `json:"event_type"`
	EventCategory    string          `json:"event_category"`
	Result           string          `json:"result"`
	AdditionalData   json.RawMessage `json:"additional_data,omitempty"`
	CreatedBy        string          `json:"created_by"`
	ApplicationKey   string          `json:"application_key,omitempty"`
	ApplicationName  string          `json:"application_name,omitempty"`
	ProjectKey       string          `json:"project_key,omitempty"`
	ProjectName      string          `json:"project_name,omitempty"`
}

// activityLogAPIResponse matches the paginated wrapper returned by GET /v1/activity/log.
type activityLogAPIResponse struct {
	ActivityLogs []activityLogEntryAPIResponse `json:"activity_logs"`
	Total        int                           `json:"total"`
	Limit        int64                         `json:"limit"`
	Offset       int64                         `json:"offset"`
}

var activityLogEntryAttrType = map[string]attr.Type{
	"event_id":          types.StringType,
	"timestamp":         types.Int64Type,
	"subject_type":      types.StringType,
	"subject_id":        types.StringType,
	"subject_name":      types.StringType,
	"event_description": types.StringType,
	"event_type":        types.StringType,
	"event_category":    types.StringType,
	"result":            types.StringType,
	"additional_data":   types.StringType,
	"created_by":        types.StringType,
	"application_key":   types.StringType,
	"application_name":  types.StringType,
	"project_key":       types.StringType,
	"project_name":      types.StringType,
}

func (d *ActivityLogDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_activity_log"
}

func (d *ActivityLogDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	// multiStringFilter builds an optional list-of-strings filter attribute.
	multiStringFilter := func(description string) schema.ListAttribute {
		return schema.ListAttribute{
			Description: description,
			ElementType: types.StringType,
			Optional:    true,
			Validators: []validator.List{
				listvalidator.ValueStringsAre(
					stringvalidator.LengthAtLeast(1),
				),
			},
		}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Returns AppTrust activity logs, with optional sorting and filtering. " +
			"Wraps the `GET /v1/activity/log` API.\n\n" +
			"## API Notes\n\n" +
			"- All list-type filters (`application_key`, `project_key`, `application_name`, `project_name`, " +
			"`subject_type`, `event_type`, `event_category`, `result`, `created_by`) are sent as a single " +
			"comma-separated query parameter and may match any of the supplied values.\n" +
			"- Pagination is supported via `limit` (default 100, max 10000) and `offset` (default 0).\n" +
			"- Ordering is via `sort_by` (default `timestamp`) and `sort` (`asc`/`desc`, default `desc`).",
		Attributes: map[string]schema.Attribute{
			// Filters
			"application_key":  multiStringFilter("Filters by application key."),
			"project_key":      multiStringFilter("Filters by project key."),
			"application_name": multiStringFilter("Filters by application name."),
			"project_name":     multiStringFilter("Filters by project name."),
			"timestamp_from": schema.Int64Attribute{
				Description: "Filters by timestamp from (UNIX timestamp, milliseconds).",
				Optional:    true,
			},
			"timestamp_to": schema.Int64Attribute{
				Description: "Filters by timestamp to (UNIX timestamp, milliseconds).",
				Optional:    true,
			},
			"subject_type": multiStringFilter("Filters by subject type."),
			"subject_id": schema.StringAttribute{
				Description: "Filters by subject id.",
				Optional:    true,
			},
			"subject_name": schema.StringAttribute{
				Description: "Filters by subject name.",
				Optional:    true,
			},
			"event_type":     multiStringFilter("Filters by event type."),
			"event_category": multiStringFilter("Filters by event category."),
			"result": schema.ListAttribute{
				Description: fmt.Sprintf("Filters by result. Allowed values: %s.", strings.Join(activityLogResultOptions, ", ")),
				ElementType: types.StringType,
				Optional:    true,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						stringvalidator.OneOf(activityLogResultOptions...),
					),
				},
			},
			"prefix": schema.StringAttribute{
				Description: "Filters by prefix.",
				Optional:    true,
			},
			"created_by": multiStringFilter("Filters by the user who initiated the event."),
			"limit": schema.Int64Attribute{
				Description: "Maximum number of logs to return. API default is 100, maximum is 10000.",
				Optional:    true,
			},
			"offset": schema.Int64Attribute{
				Description: "Number of records to skip before returning the response. Used for pagination. API default is 0.",
				Optional:    true,
			},
			"sort_by": schema.StringAttribute{
				Description: fmt.Sprintf("Field by which to sort results. Allowed values: %s. API default is 'timestamp'.", strings.Join(activityLogSortByOptions, ", ")),
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(activityLogSortByOptions...),
				},
			},
			"sort": schema.StringAttribute{
				Description: fmt.Sprintf("Sort direction. Allowed values: %s. API default is 'desc'.", strings.Join(activityLogSortOptions, ", ")),
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(activityLogSortOptions...),
				},
			},
			// Computed results
			"activity_logs": schema.ListNestedAttribute{
				Description: "List of activity log entries matching the query.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"event_id": schema.StringAttribute{
							Description: "The unique identifier for the activity log entry.",
							Computed:    true,
						},
						"timestamp": schema.Int64Attribute{
							Description: "The UNIX timestamp (milliseconds) when the event occurred.",
							Computed:    true,
						},
						"subject_type": schema.StringAttribute{
							Description: "The type of subject that triggered the event.",
							Computed:    true,
						},
						"subject_id": schema.StringAttribute{
							Description: "The ID of the subject that triggered the event.",
							Computed:    true,
						},
						"subject_name": schema.StringAttribute{
							Description: "The name of the subject that triggered the event.",
							Computed:    true,
						},
						"event_description": schema.StringAttribute{
							Description: "The event description.",
							Computed:    true,
						},
						"event_type": schema.StringAttribute{
							Description: "The event type.",
							Computed:    true,
						},
						"event_category": schema.StringAttribute{
							Description: "The event category.",
							Computed:    true,
						},
						"result": schema.StringAttribute{
							Description: "The result of the event (success, failure, warning).",
							Computed:    true,
						},
						"additional_data": schema.StringAttribute{
							Description: "Additional structured data related to the event, as a raw JSON string.",
							Computed:    true,
						},
						"created_by": schema.StringAttribute{
							Description: "The user who initiated the event.",
							Computed:    true,
						},
						"application_key": schema.StringAttribute{
							Description: "The key of the associated application, if applicable.",
							Computed:    true,
						},
						"application_name": schema.StringAttribute{
							Description: "The display name of the associated application.",
							Computed:    true,
						},
						"project_key": schema.StringAttribute{
							Description: "The key of the associated project.",
							Computed:    true,
						},
						"project_name": schema.StringAttribute{
							Description: "The display name of the associated project.",
							Computed:    true,
						},
					},
				},
			},
			"total": schema.Int64Attribute{
				Description: "Total number of activity logs matching the query.",
				Computed:    true,
			},
		},
	}
}

func (d *ActivityLogDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (d *ActivityLogDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ActivityLogDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading activity_log datasource")

	queryValues := url.Values{}

	// List filters are sent as a single comma-separated value (OpenAPI style=form, explode=false).
	addCommaSeparated := func(key string, list types.List) {
		if list.IsNull() || list.IsUnknown() {
			return
		}
		var values []string
		resp.Diagnostics.Append(list.ElementsAs(ctx, &values, false)...)
		if resp.Diagnostics.HasError() || len(values) == 0 {
			return
		}
		queryValues.Set(key, strings.Join(values, ","))
	}

	addCommaSeparated("application_key", data.ApplicationKey)
	addCommaSeparated("project_key", data.ProjectKey)
	addCommaSeparated("application_name", data.ApplicationName)
	addCommaSeparated("project_name", data.ProjectName)
	addCommaSeparated("subject_type", data.SubjectType)
	addCommaSeparated("event_type", data.EventType)
	addCommaSeparated("event_category", data.EventCategory)
	addCommaSeparated("result", data.Result)
	addCommaSeparated("created_by", data.CreatedBy)

	if !data.TimestampFrom.IsNull() {
		queryValues.Set("timestamp_from", strconv.FormatInt(data.TimestampFrom.ValueInt64(), 10))
	}
	if !data.TimestampTo.IsNull() {
		queryValues.Set("timestamp_to", strconv.FormatInt(data.TimestampTo.ValueInt64(), 10))
	}
	if !data.SubjectId.IsNull() {
		queryValues.Set("subject_id", data.SubjectId.ValueString())
	}
	if !data.SubjectName.IsNull() {
		queryValues.Set("subject_name", data.SubjectName.ValueString())
	}
	if !data.Prefix.IsNull() {
		queryValues.Set("prefix", data.Prefix.ValueString())
	}
	if !data.Limit.IsNull() {
		queryValues.Set("limit", strconv.FormatInt(data.Limit.ValueInt64(), 10))
	}
	if !data.Offset.IsNull() {
		queryValues.Set("offset", strconv.FormatInt(data.Offset.ValueInt64(), 10))
	}
	if !data.SortBy.IsNull() {
		queryValues.Set("sort_by", data.SortBy.ValueString())
	}
	if !data.Sort.IsNull() {
		queryValues.Set("sort", data.Sort.ValueString())
	}
	if resp.Diagnostics.HasError() {
		return
	}

	var apiResp activityLogAPIResponse
	response, err := d.ProviderData.Client.R().
		SetContext(ctx).
		SetQueryParamsFromValues(queryValues).
		SetResult(&apiResp).
		Get(activityLogEndpoint)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Data Source",
			"An unexpected error occurred while fetching the activity log. "+
				"Please report this issue to the provider developers.\n\n"+
				"Error: "+err.Error(),
		)
		return
	}

	if response.IsError() {
		if response.StatusCode() == http.StatusNotFound {
			// No logs found, return empty list.
			apiResp = activityLogAPIResponse{}
		} else {
			resp.Diagnostics.AddError(
				"Unable to Read Data Source",
				"An unexpected error occurred while fetching the activity log. "+
					"Please report this issue to the provider developers.\n\n"+
					"Error: "+response.String(),
			)
			return
		}
	}

	resp.Diagnostics.Append(data.FromAPIModel(ctx, apiResp)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (m *ActivityLogDataSourceModel) FromAPIModel(ctx context.Context, data activityLogAPIResponse) diag.Diagnostics {
	var diags diag.Diagnostics

	m.Total = types.Int64Value(int64(data.Total))

	entries := make([]attr.Value, 0, len(data.ActivityLogs))
	for _, e := range data.ActivityLogs {
		additionalData := ""
		if len(e.AdditionalData) > 0 && string(e.AdditionalData) != "null" {
			additionalData = string(e.AdditionalData)
		}

		entryObj, d := types.ObjectValue(
			activityLogEntryAttrType,
			map[string]attr.Value{
				"event_id":          types.StringValue(e.EventId),
				"timestamp":         types.Int64Value(e.Timestamp),
				"subject_type":      types.StringValue(e.SubjectType),
				"subject_id":        types.StringValue(e.SubjectId),
				"subject_name":      types.StringValue(e.SubjectName),
				"event_description": types.StringValue(e.EventDescription),
				"event_type":        types.StringValue(e.EventType),
				"event_category":    types.StringValue(e.EventCategory),
				"result":            types.StringValue(e.Result),
				"additional_data":   types.StringValue(additionalData),
				"created_by":        types.StringValue(e.CreatedBy),
				"application_key":   types.StringValue(e.ApplicationKey),
				"application_name":  types.StringValue(e.ApplicationName),
				"project_key":       types.StringValue(e.ProjectKey),
				"project_name":      types.StringValue(e.ProjectName),
			},
		)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
		entries = append(entries, entryObj)
	}

	activityLogsList, d := types.ListValue(
		types.ObjectType{AttrTypes: activityLogEntryAttrType},
		entries,
	)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	m.ActivityLogs = activityLogsList
	return diags
}
