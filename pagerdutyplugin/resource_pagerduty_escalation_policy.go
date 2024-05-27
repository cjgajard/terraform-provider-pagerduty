package pagerduty

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/PagerDuty/terraform-provider-pagerduty/util"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

type resourceEscalationPolicy struct{ client *pagerduty.Client }

var (
	_ resource.ResourceWithConfigure   = (*resourceEscalationPolicy)(nil)
	_ resource.ResourceWithImportState = (*resourceEscalationPolicy)(nil)
)

func (r *resourceEscalationPolicy) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pagerduty_escalation_policy"
}

func (r *resourceEscalationPolicy) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
				// Validator: validateIsAllowedString(NoNonPrintableCharsOrSpecialChars),
			},
			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("Managed by Terraform"),
			},
			"num_loops": schema.Int64Attribute{
				Optional: true,
				Validators: []validator.Int64{
					int64validator.Between(0, 9),
				},
			},
			"teams": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
			"rule": schema.ListAttribute{
				Required:    true,
				ElementType: escalationPolicyRuleObjectType,
			},
		},
	}
}

func (r *resourceEscalationPolicy) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceEscalationPolicyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan := buildPagerdutyEscalationPolicy(&model)
	log.Printf("[INFO] Creating PagerDuty escalation policy %s", plan.Name)

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		response, err := r.client.CreateEscalationPolicyWithContext(ctx, plan)
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
			fmt.Sprintf("Error creating PagerDuty escalation policy %s", plan.Name),
			err.Error(),
		)
		return
	}

	model, err = requestGetEscalationPolicy(ctx, r.client, plan.ID, true, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading PagerDuty escalation policy %s", plan.ID),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *resourceEscalationPolicy) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var id types.String

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	log.Printf("[INFO] Reading PagerDuty escalation policy %s", id)

	state, err := requestGetEscalationPolicy(ctx, r.client, id.ValueString(), false, &resp.Diagnostics)
	if err != nil {
		if util.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading PagerDuty escalation policy %s", id),
			err.Error(),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *resourceEscalationPolicy) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceEscalationPolicyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan := buildPagerdutyEscalationPolicy(&model)
	if plan.ID == "" {
		var id string
		req.State.GetAttribute(ctx, path.Root("id"), &id)
		plan.ID = id
	}
	log.Printf("[INFO] Updating PagerDuty escalation policy %s", plan.ID)

	escalationPolicy, err := r.client.UpdateEscalationPolicyWithContext(ctx, plan.ID, plan)
	if err != nil {
		if util.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error updating PagerDuty escalation policy %s", plan.ID),
			err.Error(),
		)
		return
	}
	model = flattenEscalationPolicy(escalationPolicy)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *resourceEscalationPolicy) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var id types.String

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	log.Printf("[INFO] Deleting PagerDuty escalation policy %s", id)

	err := r.client.DeleteEscalationPolicyWithContext(ctx, id.ValueString())
	if err != nil && !util.IsNotFoundError(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error deleting PagerDuty escalation policy %s", id),
			err.Error(),
		)
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *resourceEscalationPolicy) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(ConfigurePagerdutyClient(&r.client, req.ProviderData)...)
}

func (r *resourceEscalationPolicy) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type resourceEscalationPolicyModel struct {
	ID          types.String
	Name        types.String
	Description types.String
	NumLoops    types.Int64
	Teams       types.List
	Rule        types.List
}

var escalationPolicyRuleObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"id":                          types.StringType,
		"escalation_delay_in_minutes": types.Int64Type, // required, at least 1
		"escalation_rule_assignment_strategy": types.ListType{
			// required, computed, max items 1
			ElemType: escalationRuleAssignmentStategyObjectType,
		},
		"target": types.ListType{
			ElemType: types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"type": types.StringType, // default "user_reference"
					// OneOf: []string{"user_reference", "schedule_reference"}
					"id": types.StringType, // required
				},
			},
		}, // required
	},
}

var escalationRuleAssignmentStategyObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"type": types.StringType,
		// "assign_to_everyone",
		// "round_robin",
	},
}

func requestGetEscalationPolicy(ctx context.Context, client *pagerduty.Client, id string, retryNotFound bool, diags *diag.Diagnostics) (resourceEscalationPolicyModel, error) {
	var model resourceEscalationPolicyModel

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		escalationPolicy, err := client.GetEscalationPolicyWithContext(ctx, id, &pagerduty.GetEscalationPolicyOptions{})
		if err != nil {
			if util.IsBadRequestError(err) {
				return retry.NonRetryableError(err)
			}
			if !retryNotFound && util.IsNotFoundError(err) {
				return retry.NonRetryableError(err)
			}
			return retry.RetryableError(err)
		}
		model = flattenEscalationPolicy(escalationPolicy)
		return nil
	})

	return model, err
}

func buildPagerdutyEscalationPolicy(model *resourceEscalationPolicyModel) pagerduty.EscalationPolicy {
	return pagerduty.EscalationPolicy{}
}

func flattenEscalationPolicy(response *pagerduty.EscalationPolicy) resourceEscalationPolicyModel {
	model := resourceEscalationPolicyModel{
		ID: types.StringValue(response.ID),
	}
	return model
}
