package pagerduty

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/PagerDuty/terraform-provider-pagerduty/util"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

type resourceSlackConnection struct {
	client *pagerduty.Client
}

var (
	_ resource.ResourceWithConfigure      = (*resourceSlackConnection)(nil)
	_ resource.ResourceWithImportState    = (*resourceSlackConnection)(nil)
	_ resource.ResourceWithValidateConfig = (*resourceSlackConnection)(nil)
)

func (r *resourceSlackConnection) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pagerduty_slack_connection"
}

func (r *resourceSlackConnection) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_id":   schema.StringAttribute{Required: true},
			"source_name": schema.StringAttribute{Computed: true},
			"source_type": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf("service_reference", "team_reference"),
				},
			},
			"channel_id":   schema.StringAttribute{Required: true},
			"channel_name": schema.StringAttribute{Computed: true},
			"workspace_id": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  util.DefaultGetenv("SLACK_CONNECTION_WORKSPACE_ID"),
			},
			"notification_type": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf("responder", "stakeholder"),
				},
			},
			"config": schema.ListAttribute{
				Required: true,
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"events":     types.ListType{ElemType: types.StringType}, // required
						"priorities": types.ListType{ElemType: types.StringType}, // optional
						"urgency":    types.StringType,                           // optional, high, low
					},
				},
			},
		},
	}
}

func (r *resourceSlackConnection) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	// req.Config.GetAttibute()
}

func (r *resourceSlackConnection) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceSlackConnectionModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	workspaceID := model.WorkspaceID.ValueString()
	plan := buildPagerdutySlackConnection(ctx, &model, &resp.Diagnostics)
	log.Printf("[INFO] Creating PagerDuty slack connection for source %s and slack channel %s", plan.SourceID, plan.ChannelID)

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		response, err := r.client.CreateSlackConnectionWithContext(ctx, workspaceID, plan)
		if err != nil {
			if util.IsBadRequestError(err) {
				return retry.NonRetryableError(err)
			}
			return retry.RetryableError(err)
		}
		plan.ID = response.ID
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error creating PagerDuty slack connection for source %s and slack channel %s", plan.SourceID, plan.ChannelID),
			err.Error(),
		)
		return
	}

	model, err = requestGetSlackConnection(ctx, r.client, workspaceID, plan.ID, true, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading PagerDuty slack connection %s", plan.ID),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *resourceSlackConnection) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var id types.String
	var workspaceID types.String

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("workspace_id"), &workspaceID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	log.Printf("[INFO] Reading PagerDuty slack connection %s", id)

	state, err := requestGetSlackConnection(ctx, r.client, workspaceID.ValueString(), id.ValueString(), false, &resp.Diagnostics)
	if err != nil {
		if util.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading PagerDuty slack connection %s", id),
			err.Error(),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *resourceSlackConnection) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceSlackConnectionModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan := buildPagerdutySlackConnection(ctx, &model, &resp.Diagnostics)
	log.Printf("[INFO] Updating PagerDuty slack connection %s", plan.ID)

	slackConnection, err := r.client.UpdateSlackConnectionWithContext(ctx, plan.ID, plan)
	if err != nil {
		if util.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error updating PagerDuty slack connection %s", plan.ID),
			err.Error(),
		)
		return
	}
	model = flattenSlackConnection(slackConnection, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *resourceSlackConnection) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var id types.String
	var workspaceID types.String

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("workspace_id"), &workspaceID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	log.Printf("[INFO] Deleting PagerDuty slack connection %s", id)

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		err := r.client.DeleteSlackConnectionWithContext(ctx, workspaceID.ValueString(), id.ValueString())
		if err != nil {
			if util.IsBadRequestError(err) {
				return retry.NonRetryableError(err)
			}
			if util.IsNotFoundError(err) {
				resp.State.RemoveResource(ctx)
				return nil
			}
			return retry.RetryableError(err)
		}
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error deleting PagerDuty slack connection %s", id),
			err.Error(),
		)
		return
	}
}

func (r *resourceSlackConnection) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	//                                        ↓↓↓↓↓
	resp.Diagnostics.Append(ConfigurePagerdutySlackClient(&r.client, req.ProviderData)...)
}

func (r *resourceSlackConnection) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ids := strings.Split(req.ID, ".")
	if len(ids) != 2 {
		resp.Diagnostics.AddError(
			"Error importing pagerduty_slack_connection",
			"Expecting an importation ID formed as '<workspace_id>.<slack_connection_id>'",
		)
	}
	workspaceID, connectionID := ids[0], ids[1]
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), connectionID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_id"), workspaceID)...)
}

type resourceSlackConnectionModel struct {
	ID               types.String `tfsdk:"id"`
	SourceID         types.String `tfsdk:"source_id"`
	SourceName       types.String `tfsdk:"source_name"`
	SourceType       types.String `tfsdk:"source_type"`
	ChannelID        types.String `tfsdk:"channel_id"`
	ChannelName      types.String `tfsdk:"channel_name"`
	WorkspaceID      types.String `tfsdk:"workspace_id"`
	NotificationType types.String `tfsdk:"notification_type"`
	Config           types.List   `tfsdk:"config"`
}

func requestGetSlackConnection(ctx context.Context, client *pagerduty.Client, workspaceID, id string, retryNotFound bool, diags *diag.Diagnostics) (resourceSlackConnectionModel, error) {
	var model resourceSlackConnectionModel

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		slackConnection, err := client.GetSlackConnectionWithContext(ctx, workspaceID, id)
		if err != nil {
			if util.IsBadRequestError(err) {
				return retry.NonRetryableError(err)
			}
			if !retryNotFound && util.IsNotFoundError(err) {
				return retry.NonRetryableError(err)
			}
			return retry.RetryableError(err)
		}
		model = flattenSlackConnection(slackConnection, diags)
		return nil
	})

	return model, err
}

func buildPagerdutySlackConnection(ctx context.Context, model *resourceSlackConnectionModel, diags *diag.Diagnostics) pagerduty.SlackConnection {
	return pagerduty.SlackConnection{
		ID:               model.ID.ValueString(),
		SourceID:         model.SourceID.ValueString(),
		SourceName:       model.SourceName.ValueString(),
		SourceType:       model.SourceType.ValueString(),
		ChannelID:        model.ChannelID.ValueString(),
		ChannelName:      model.ChannelName.ValueString(),
		NotificationType: model.NotificationType.ValueString(),
		Config:           buildPagerdutySlackConnectionConfig(ctx, model.Config, diags),
	}
}

func buildPagerdutySlackConnectionConfig(ctx context.Context, list types.List, diags *diag.Diagnostics) *pagerduty.SlackConnectionConfig {
	var target []struct {
		Events     types.List   `tfsdk:"events"`
		Priorities types.List   `tfsdk:"priorities"`
		Urgency    types.String `tfsdk:"urgency"`
	}
	if d := list.ElementsAs(ctx, target, false); d.HasError() {
		return nil
	}
	obj := target[0]

	events := []string{}
	d := obj.Events.ElementsAs(ctx, &events, false)
	if diags.Append(d...); d.HasError() {
		return nil
	}

	priorities := []string{}
	d = obj.Priorities.ElementsAs(ctx, &priorities, false)
	if diags.Append(d...); d.HasError() {
		return nil
	}

	// Expands the use of star wildcard ("*") configuration for an attribute
	// to its matching expected value by PagerDuty's API, which is nil. This
	// is necessary when the API accepts and interprets nil and empty
	// configurations as valid settings.
	if len(priorities) == 1 && priorities[0] == StarWildcardConfig {
		priorities = nil
	}

	var urgency *string
	if !obj.Urgency.IsNull() && !obj.Urgency.IsUnknown() {
		urgency = obj.Urgency.ValueStringPointer()
	}

	return &pagerduty.SlackConnectionConfig{
		Events:     events,
		Priorities: priorities,
		Urgency:    urgency,
	}
}

func flattenSlackConnection(response *pagerduty.SlackConnection, diags *diag.Diagnostics) resourceSlackConnectionModel {
	model := resourceSlackConnectionModel{
		ID:               types.StringValue(response.ID),
		SourceID:         types.StringValue(response.SourceID),
		SourceName:       types.StringValue(response.SourceName),
		ChannelID:        types.StringValue(response.ChannelID),
		ChannelName:      types.StringValue(response.ChannelName),
		NotificationType: types.StringValue(response.NotificationType),
		Config:           flattenSlackConnectionConfig(response.Config, diags),
	}
	return model
}

func flattenSlackConnectionConfig(cfg *pagerduty.SlackConnectionConfig, diags *diag.Diagnostics) types.List {
	configObjectType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"events":     types.ListType{ElemType: types.StringType},
			"priorities": types.ListType{ElemType: types.StringType},
			"urgency":    types.StringType,
		},
	}

	urgency := types.StringNull()
	if cfg.Urgency != nil {
		urgency = types.StringValue(*cfg.Urgency)
	}

	// Flattens a `nil` configuration to its corresponding star wildcard
	// configuration value for an attribute which is meant to be
	// accepting this kind of configuration, with the only purpose to match
	// the config stored in the Terraform's state.
	priorities := cfg.Priorities
	if util.IsNilFunc(priorities) {
		priorities = []string{StarWildcardConfig}
	}

	configObj := types.ObjectValueMust(configObjectType.AttrTypes, map[string]attr.Value{
		"events":     flattenStringList(cfg.Events, diags),
		"priorities": flattenStringList(priorities, diags),
		"urgency":    urgency,
	})
	return types.ListValueMust(configObjectType, []attr.Value{configObj})
}

func flattenStringList(values []string, diags *diag.Diagnostics) types.List {
	elements := make([]attr.Value, 0, len(values))
	for _, v := range values {
		elements = append(elements, types.StringValue(v))
	}
	list, d := types.ListValue(types.StringType, elements)
	diags.Append(d...)
	return list
}

const StarWildcardConfig = "*"
