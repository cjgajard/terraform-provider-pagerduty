package pagerduty

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/PagerDuty/terraform-provider-pagerduty/util"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

type resourceEventOrchestration struct{ client *pagerduty.Client }

var (
	_ resource.ResourceWithConfigure   = (*resourceEventOrchestration)(nil)
	_ resource.ResourceWithImportState = (*resourceEventOrchestration)(nil)
)

func (r *resourceEventOrchestration) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pagerduty_event_orchestration"
}

func (r *resourceEventOrchestration) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	parametersAttr := schema.ListNestedAttribute{
		Computed: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"routing_key": schema.StringAttribute{Computed: true},
				"type":        schema.StringAttribute{Computed: true},
			},
		},
	}

	integrationAttr := schema.ListNestedAttribute{
		Computed: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"id":         schema.StringAttribute{Computed: true},
				"label":      schema.StringAttribute{Computed: true},
				"parameters": parametersAttr,
			},
		},
	}

	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name":         schema.StringAttribute{Required: true},
			"description":  schema.StringAttribute{Optional: true, Computed: true},
			"team":         schema.StringAttribute{Optional: true, Computed: true},
			"routes":       schema.Int64Attribute{Computed: true},
			"integrations": integrationAttr,
		},
	}
}

func (r *resourceEventOrchestration) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceEventOrchestrationModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan := buildPagerdutyEventOrchestration(&model)
	log.Printf("[INFO] Creating PagerDuty event orchestration %s", plan.Name)

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		response, err := r.client.CreateOrchestrationWithContext(ctx, plan)
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
			fmt.Sprintf("Error creating PagerDuty event orchestration %s", plan.Name),
			err.Error(),
		)
		return
	}

	model, err = requestGetEventOrchestration(ctx, r.client, plan.ID, false)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading PagerDuty event orchestration %s", plan.ID),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *resourceEventOrchestration) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var id types.String

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	log.Printf("[INFO] Reading PagerDuty event orchestration %s", id)

	state, err := requestGetEventOrchestration(ctx, r.client, id.ValueString(), false)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading PagerDuty event orchestration %s", id),
			err.Error(),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *resourceEventOrchestration) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceEventOrchestrationModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan := buildPagerdutyEventOrchestration(&model)
	log.Printf("[INFO] Updating PagerDuty event orchestration %s", plan.ID)

	eventOrchestration, err := r.client.UpdateOrchestrationWithContext(ctx, plan.ID, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error updating PagerDuty event orchestration %s", plan.ID),
			err.Error(),
		)
		return
	}
	model = flattenEventOrchestration(eventOrchestration)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *resourceEventOrchestration) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var id types.String

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	log.Printf("[INFO] Deleting PagerDuty event orchestration %s", id)

	err := r.client.DeleteOrchestrationWithContext(ctx, id.ValueString())
	if err != nil && !util.IsNotFoundError(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error deleting PagerDuty event orchestration %s", id),
			err.Error(),
		)
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *resourceEventOrchestration) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(ConfigurePagerdutyClient(&r.client, req.ProviderData)...)
}

func (r *resourceEventOrchestration) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type resourceEventOrchestrationModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Description  types.String `tfsdk:"description"`
	Team         types.String `tfsdk:"team"`
	Routes       types.Int64  `tfsdk:"routes"`
	Integrations types.List   `tfsdk:"integrations"`
}

func requestGetEventOrchestration(ctx context.Context, client *pagerduty.Client, id string, retryNotFound bool) (resourceEventOrchestrationModel, error) {
	var model resourceEventOrchestrationModel

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		opts := &pagerduty.GetOrchestrationOptions{}
		eventOrchestration, err := client.GetOrchestrationWithContext(ctx, id, opts)
		if err != nil {
			if util.IsBadRequestError(err) {
				return retry.NonRetryableError(err)
			}
			if !retryNotFound && util.IsNotFoundError(err) {
				return retry.NonRetryableError(err)
			}
			return retry.RetryableError(err)
		}
		model = flattenEventOrchestration(eventOrchestration)
		return nil
	})

	return model, err
}

func buildPagerdutyEventOrchestration(model *resourceEventOrchestrationModel) pagerduty.Orchestration {
	eventOrchestration := pagerduty.Orchestration{
		Name:        model.Name.ValueString(),
		Description: model.Description.ValueString(),
	}
	if !model.Team.IsNull() && !model.Team.IsUnknown() {
		eventOrchestration.Team = &pagerduty.APIReference{
			ID: model.Team.ValueString(),
		}
	}
	return eventOrchestration
}

func flattenEventOrchestration(response *pagerduty.Orchestration) resourceEventOrchestrationModel {
	model := resourceEventOrchestrationModel{
		ID:           types.StringValue(response.ID),
		Name:         types.StringValue(response.Name),
		Description:  types.StringValue(response.Description),
		Routes:       types.Int64Value(int64(response.Routes)),
		Integrations: flattenEventOrchestrationIntegrations(response.Integrations),
	}
	if response.Team != nil {
		model.Team = types.StringValue(response.Team.ID)
	}
	return model
}

var resourceEventOrchestrationParameterObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"routing_key": types.StringType,
		"type":        types.StringType,
	},
}

func flattenEventOrchestrationIntegrations(list []*pagerduty.OrchestrationIntegration) types.List {
	integrationObjectType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":         types.StringType,
			"label":      types.StringType,
			"parameters": types.ListType{ElemType: resourceEventOrchestrationParameterObjectType},
		},
	}
	elements := make([]attr.Value, 0, len(list))
	for _, integration := range list {
		obj := types.ObjectValueMust(integrationObjectType.AttrTypes, map[string]attr.Value{
			"id":         types.StringValue(integration.ID),
			"label":      types.StringNull(),
			"parameters": flattenEventOrchestrationIntegrationParameters(integration.Parameters),
		})
		elements = append(elements, obj)
	}
	return types.ListValueMust(integrationObjectType, elements)
}

func flattenEventOrchestrationIntegrationParameters(p *pagerduty.OrchestrationIntegrationParameters) types.List {
	obj := types.ObjectValueMust(resourceEventOrchestrationParameterObjectType.AttrTypes, map[string]attr.Value{
		"routing_key": types.StringValue(p.RoutingKey),
		"type":        types.StringValue(p.Type),
	})
	return types.ListValueMust(resourceEventOrchestrationParameterObjectType, []attr.Value{obj})
}
