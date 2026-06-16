package apptrust

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/jfrog/terraform-provider-shared/util"
)

const (
	LogEndpoint = "https://{jfrog_url}/apptrust/api/v1/activity/log"
	LogEndpoint  = "https://{jfrog_url}/apptrust/api/v1/activity/log"
)

func NewActivityLogDataSource() datasource.DataSource {
	return &ActivityLogDataSource{
		TypeName: "apptrust_log",
	}
}

type ActivityLogDataSource struct {
	ProviderData util.ProviderMetadata
	TypeName     string
}

type ActivityLogDataSourceModel struct {
	JfrogUrl types.String `tfsdk:"jfrog_url"`
	ApplicationKey types.List `tfsdk:"application_key"`
	ProjectKey types.List `tfsdk:"project_key"`
	ApplicationName types.List `tfsdk:"application_name"`
	ProjectName types.List `tfsdk:"project_name"`
	TimestampFrom types.Int64 `tfsdk:"timestamp_from"`
	TimestampTo types.Int64 `tfsdk:"timestamp_to"`
	SubjectType types.List `tfsdk:"subject_type"`
	SubjectId types.String `tfsdk:"subject_id"`
	SubjectName types.String `tfsdk:"subject_name"`
	EventType types.List `tfsdk:"event_type"`
	EventCategory types.List `tfsdk:"event_category"`
	Result types.List `tfsdk:"result"`
	Prefix types.String `tfsdk:"prefix"`
	CreatedBy types.List `tfsdk:"created_by"`
	Limit types.Int64 `tfsdk:"limit"`
	Offset types.Int64 `tfsdk:"offset"`
	SortBy types.String `tfsdk:"sort_by"`
	Sort types.String `tfsdk:"sort"`
}

type LogAPIModel struct {
	JfrogUrl string `json:"jfrog_url"`
	ApplicationKey string `json:"application_key"`
	ProjectKey string `json:"project_key"`
	ApplicationName string `json:"application_name"`
	ProjectName string `json:"project_name"`
	TimestampFrom string `json:"timestamp_from"`
	TimestampTo string `json:"timestamp_to"`
	SubjectType string `json:"subject_type"`
	SubjectId string `json:"subject_id"`
	SubjectName string `json:"subject_name"`
	EventType string `json:"event_type"`
	EventCategory string `json:"event_category"`
	Result string `json:"result"`
	Prefix string `json:"prefix"`
	CreatedBy string `json:"created_by"`
	Limit string `json:"limit"`
	Offset string `json:"offset"`
	SortBy string `json:"sort_by"`
	Sort string `json:"sort"`
}

func (d *ActivityLogDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = d.TypeName
}

func (d *ActivityLogDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"jfrog_url": schema.StringAttribute{
				Required: true,
				Description: "The jfrog_url of the resource.",
			},
			"application_key": schema.StringAttribute{
				Computed: true,
				Description: "Filters by application key.",
			},
			"project_key": schema.StringAttribute{
				Computed: true,
				Description: "Filters by project key.",
			},
			"application_name": schema.StringAttribute{
				Computed: true,
				Description: "Filter by application name (can be specified multiple times).",
			},
			"project_name": schema.StringAttribute{
				Computed: true,
				Description: "Filter by project name (can be specified multiple times).",
			},
			"timestamp_from": schema.StringAttribute{
				Computed: true,
				Description: "Filter by timestamp from (UNIX timestamp).",
			},
			"timestamp_to": schema.StringAttribute{
				Computed: true,
				Description: "Filter by timestamp to (UNIX timestamp).",
			},
			"subject_type": schema.StringAttribute{
				Computed: true,
				Description: "Filters by subject type.",
			},
			"subject_id": schema.StringAttribute{
				Computed: true,
				Description: "Filter by subject id.",
			},
			"subject_name": schema.StringAttribute{
				Computed: true,
				Description: "Filters by subject name.",
			},
			"event_type": schema.StringAttribute{
				Computed: true,
				Description: "Filters by event type.",
			},
			"event_category": schema.StringAttribute{
				Computed: true,
				Description: "Filters by event category.",
			},
			"result": schema.StringAttribute{
				Computed: true,
				Description: "Filters by result (success, failure, warning). Can be specified multiple times.",
			},
			"prefix": schema.StringAttribute{
				Computed: true,
				Description: "Filters by prefix.",
			},
			"created_by": schema.StringAttribute{
				Computed: true,
				Description: "Filter by created by (can be specified multiple times)",
			},
			"limit": schema.StringAttribute{
				Computed: true,
				Description: "Limit",
			},
			"offset": schema.StringAttribute{
				Computed: true,
				Description: "Offset",
			},
			"sort_by": schema.StringAttribute{
				Computed: true,
				Description: "Defines the field by which to sort results (timestamp, event_id, subject_type, subject_name, event_type, event_category, project_key, created_by). The default is timestamp.",
			},
			"sort": schema.StringAttribute{
				Computed: true,
				Description: "Defines the sort direction (asc, desc). The default is desc.",
			},
		},
		MarkdownDescription: "Fetches log from JFrog Apptrust.",
	}
}

func (d *ActivityLogDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.ProviderData = req.ProviderData.(util.ProviderMetadata)
}

func (d *ActivityLogDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	go util.SendUsageResourceRead(ctx, d.ProviderData.Client.R(), d.ProviderData.ProductId, d.TypeName)

	var state ActivityLogDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var result ActivityLogAPIModel

	response, err := d.ProviderData.Client.R().
		SetPathParams(map[string]string{
			"jfrog_url": state.JfrogUrl.ValueString(),
		}).
		SetResult(&result).
		Get(LogEndpoint)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read apptrust_log", err.Error())
		return
	}

	if response.IsError() {
		resp.Diagnostics.AddError("Unable to read apptrust_log", response.String())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
