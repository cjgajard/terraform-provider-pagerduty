package pagerduty

import (
	"context"
	"log"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/PagerDuty/terraform-provider-pagerduty/util"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

type resourceIncidentWorkflowTrigger struct{ client *pagerduty.Client }

var (
	_ resource.ResourceWithConfigure      = (*resourceIncidentWorkflowTrigger)(nil)
	_ resource.ResourceWithImportState    = (*resourceIncidentWorkflowTrigger)(nil)
	_ resource.ResourceWithValidateConfig = (*resourceIncidentWorkflowTrigger)(nil)
)

func (r *resourceIncidentWorkflowTrigger) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pagerduty_incident_workflow_trigger"
}

var incidentWorkflowTriggerPermissionsAttrTypes = map[string]attr.Type{
	"restricted": types.BoolType,
	"team_id":    types.StringType,
}

func (r *resourceIncidentWorkflowTrigger) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An Incident Workflow Trigger defines when and if an Incident Workflow will be triggered.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "May be `manual`, `conditional` or `incident_type`. Updating causes resource replacement.",
				Validators: []validator.String{
					stringvalidator.OneOf("manual", "conditional", "incident_type"),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"workflow": schema.StringAttribute{
				Required:      true,
				Description:   "The workflow ID for the workflow to trigger. Updating causes resource replacement.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"services": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "A list of service IDs. Incidents in any of the listed services are eligible to fire this trigger.",
			},
			"incident_types": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "A list of incident type IDs. Incidents of any of the listed types are eligible to fire this trigger. Required for, and only allowed on, `incident_type` triggers.",
			},
			"subscribed_to_all_services": schema.BoolAttribute{
				Required:    true,
				Description: "Set to `true` if the trigger should be eligible for firing on all services. Only allowed to be `true` if `services` is not defined or empty.",
			},
			"condition": schema.StringAttribute{
				Optional:    true,
				Description: "A PCL condition string which must be satisfied for the trigger to fire. Required for `conditional`-type triggers, and not allowed for `manual`- or `incident_type`-type triggers.",
			},
		},
		Blocks: map[string]schema.Block{
			"permissions": schema.ListNestedBlock{
				Description: "Indicates who can start this Trigger. Applicable only to `manual`-type triggers.",
				Validators:  []validator.List{listvalidator.SizeAtMost(1)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"restricted": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Default:     booldefault.StaticBool(false),
							Description: "If `true`, indicates that the Trigger can only be started by authorized Users. If `false` (default), any user can start this Trigger.",
						},
						"team_id": schema.StringAttribute{
							Optional:    true,
							Description: "The ID of the Team whose members can manually start this Trigger. Required and allowed only when `restricted` is `true`.",
						},
					},
				},
			},
		},
	}
}

type resourceIncidentWorkflowTriggerModel struct {
	ID                      types.String `tfsdk:"id"`
	Type                    types.String `tfsdk:"type"`
	Workflow                types.String `tfsdk:"workflow"`
	Services                types.List   `tfsdk:"services"`
	IncidentTypes           types.List   `tfsdk:"incident_types"`
	SubscribedToAllServices types.Bool   `tfsdk:"subscribed_to_all_services"`
	Condition               types.String `tfsdk:"condition"`
	Permissions             types.List   `tfsdk:"permissions"`
}

type incidentWorkflowTriggerPermissionsModel struct {
	Restricted types.Bool   `tfsdk:"restricted"`
	TeamID     types.String `tfsdk:"team_id"`
}

// ValidateConfig carries forward every rule the SDKv2 resource enforced via
// CustomizeDiff (validateIncidentWorkflowTrigger) plus the apply-time check
// from expandIncidentWorkflowTriggerPermissions, and adds the equivalent
// rules for the new incident_type trigger type.
func (r *resourceIncidentWorkflowTrigger) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model resourceIncidentWorkflowTriggerModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() || model.Type.IsUnknown() {
		return
	}
	triggerType := model.Type.ValueString()

	// An empty string is treated the same as an omitted condition: the
	// legacy SDKv2 resource used d.GetOk, which cannot tell the two apart
	// either, and existing configs rely on that (e.g. a conditional trigger
	// planned with condition = "" before a real condition is known).
	conditionSet := !model.Condition.IsNull() && !model.Condition.IsUnknown() && model.Condition.ValueString() != ""

	if triggerType == "manual" && conditionSet {
		resp.Diagnostics.AddAttributeError(
			path.Root("condition"),
			"Invalid configuration",
			"when trigger type manual is used, condition must not be specified",
		)
	}
	if triggerType == "incident_type" && conditionSet {
		resp.Diagnostics.AddAttributeError(
			path.Root("condition"),
			"Invalid configuration",
			"condition is not allowed when trigger type is incident_type",
		)
	}

	if !model.SubscribedToAllServices.IsUnknown() && model.SubscribedToAllServices.ValueBool() &&
		!model.Services.IsNull() && !model.Services.IsUnknown() && len(model.Services.Elements()) > 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("services"),
			"Invalid configuration",
			"when subscribed_to_all_services is true, services must either be not defined or empty",
		)
	}

	hasIncidentTypes := !model.IncidentTypes.IsNull() && !model.IncidentTypes.IsUnknown() && len(model.IncidentTypes.Elements()) > 0
	if triggerType == "incident_type" && !hasIncidentTypes && !model.IncidentTypes.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("incident_types"),
			"Invalid configuration",
			"incident_types must be specified when trigger type is incident_type",
		)
	}
	if triggerType != "incident_type" && hasIncidentTypes {
		resp.Diagnostics.AddAttributeError(
			path.Root("incident_types"),
			"Invalid configuration",
			"incident_types can only be specified when trigger type is incident_type",
		)
	}

	if model.Permissions.IsUnknown() || len(model.Permissions.Elements()) == 0 {
		return
	}
	var perms []incidentWorkflowTriggerPermissionsModel
	d := model.Permissions.ElementsAs(ctx, &perms, false)
	if resp.Diagnostics.Append(d...); d.HasError() || len(perms) == 0 {
		return
	}
	p := perms[0]
	restricted := !p.Restricted.IsNull() && !p.Restricted.IsUnknown() && p.Restricted.ValueBool()

	if triggerType != "manual" && restricted {
		resp.Diagnostics.AddAttributeError(
			path.Root("permissions").AtListIndex(0).AtName("restricted"),
			"Invalid configuration",
			"restricted can only be true when trigger type is manual",
		)
	}
	if !restricted && !p.TeamID.IsNull() && !p.TeamID.IsUnknown() && p.TeamID.ValueString() != "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("permissions").AtListIndex(0).AtName("team_id"),
			"Invalid configuration",
			"team_id not allowed when restricted is false",
		)
	}
	if restricted && (p.TeamID.IsNull() || (!p.TeamID.IsUnknown() && p.TeamID.ValueString() == "")) {
		resp.Diagnostics.AddAttributeError(
			path.Root("permissions").AtListIndex(0).AtName("team_id"),
			"Invalid configuration",
			"team_id must be specified when restricted is true",
		)
	}
}

func (r *resourceIncidentWorkflowTrigger) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceIncidentWorkflowTriggerModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan, diags := buildIncidentWorkflowTriggerCreate(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[INFO] Creating PagerDuty incident workflow trigger %s for %s", plan.TriggerType, model.Workflow.ValueString())

	var created *pagerduty.IncidentWorkflowTrigger
	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		var err error
		created, err = r.client.CreateIncidentWorkflowTrigger(ctx, plan)
		if err != nil {
			if util.IsBadRequestError(err) {
				return retry.NonRetryableError(err)
			}
			return retry.RetryableError(err)
		}
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating PagerDuty incident workflow trigger", err.Error())
		return
	}

	// Non-computed attributes (everything but id) must be written back
	// exactly as planned, or Terraform reports "Provider produced
	// inconsistent result after apply". Only id comes from the API.
	model.ID = types.StringValue(created.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *resourceIncidentWorkflowTrigger) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceIncidentWorkflowTriggerModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	log.Printf("[INFO] Reading PagerDuty incident workflow trigger %s", id)

	var trigger *pagerduty.IncidentWorkflowTrigger
	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		var err error
		trigger, err = r.client.GetIncidentWorkflowTrigger(ctx, id, pagerduty.GetIncidentWorkflowTriggerOptions{})
		if err != nil {
			if util.IsBadRequestError(err) {
				return retry.NonRetryableError(err)
			}
			if util.IsNotFoundError(err) {
				return retry.NonRetryableError(err)
			}
			return retry.RetryableError(err)
		}
		return nil
	})
	if err != nil {
		if util.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading PagerDuty incident workflow trigger", err.Error())
		return
	}

	model, diags := flattenIncidentWorkflowTrigger(trigger, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *resourceIncidentWorkflowTrigger) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceIncidentWorkflowTriggerModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan, diags := buildIncidentWorkflowTriggerUpdate(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := model.ID.ValueString()
	log.Printf("[INFO] Updating PagerDuty incident workflow trigger %s", id)

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		_, err := r.client.UpdateIncidentWorkflowTrigger(ctx, id, plan)
		if err != nil {
			if util.IsBadRequestError(err) {
				return retry.NonRetryableError(err)
			}
			return retry.RetryableError(err)
		}
		return nil
	})
	if err != nil {
		if util.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error updating PagerDuty incident workflow trigger", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *resourceIncidentWorkflowTrigger) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state resourceIncidentWorkflowTriggerModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	log.Printf("[INFO] Deleting PagerDuty incident workflow trigger %s", id)

	err := r.client.DeleteIncidentWorkflowTrigger(ctx, id)
	if err != nil && !util.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting PagerDuty incident workflow trigger", err.Error())
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *resourceIncidentWorkflowTrigger) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(ConfigurePagerdutyClient(&r.client, req.ProviderData)...)
}

func (r *resourceIncidentWorkflowTrigger) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// buildIncidentWorkflowTriggerCreate translates a plan into a
// CreateIncidentWorkflowTriggerOptions. TriggerType and Workflow are only
// ever sent on create: neither may change after creation (both force
// resource replacement).
func buildIncidentWorkflowTriggerCreate(ctx context.Context, model *resourceIncidentWorkflowTriggerModel) (pagerduty.CreateIncidentWorkflowTriggerOptions, diag.Diagnostics) {
	var diags diag.Diagnostics

	services, d := stringListToAPIReferences(ctx, model.Services, "service_reference")
	diags.Append(d...)
	incidentTypes, d := stringListToAPIReferences(ctx, model.IncidentTypes, "incident_type_reference")
	diags.Append(d...)
	permissions, d := buildIncidentWorkflowTriggerPermissions(ctx, model.Permissions)
	diags.Append(d...)

	opts := pagerduty.CreateIncidentWorkflowTriggerOptions{
		TriggerType:             pagerduty.IncidentWorkflowTriggerType(model.Type.ValueString()),
		Workflow:                &pagerduty.APIReference{ID: model.Workflow.ValueString(), Type: "workflow_reference"},
		Services:                services,
		IncidentTypes:           incidentTypes,
		SubscribedToAllServices: model.SubscribedToAllServices.ValueBool(),
		Permissions:             permissions,
	}
	opts.Condition = buildIncidentWorkflowTriggerCondition(model)

	return opts, diags
}

// buildIncidentWorkflowTriggerUpdate translates a plan into an
// UpdateIncidentWorkflowTriggerOptions.
func buildIncidentWorkflowTriggerUpdate(ctx context.Context, model *resourceIncidentWorkflowTriggerModel) (pagerduty.UpdateIncidentWorkflowTriggerOptions, diag.Diagnostics) {
	var diags diag.Diagnostics

	services, d := stringListToAPIReferences(ctx, model.Services, "service_reference")
	diags.Append(d...)
	incidentTypes, d := stringListToAPIReferences(ctx, model.IncidentTypes, "incident_type_reference")
	diags.Append(d...)
	permissions, d := buildIncidentWorkflowTriggerPermissions(ctx, model.Permissions)
	diags.Append(d...)

	opts := pagerduty.UpdateIncidentWorkflowTriggerOptions{
		Services:                services,
		IncidentTypes:           incidentTypes,
		SubscribedToAllServices: model.SubscribedToAllServices.ValueBool(),
		Permissions:             permissions,
	}
	opts.Condition = buildIncidentWorkflowTriggerCondition(model)

	return opts, diags
}

// buildIncidentWorkflowTriggerCondition reproduces the SDKv2 resource's
// special handling for condition: a conditional trigger requires an explicit
// (possibly empty) condition string to be sent, even though Terraform
// considers an omitted attribute and an empty string different things.
func buildIncidentWorkflowTriggerCondition(model *resourceIncidentWorkflowTriggerModel) *string {
	if model.Type.ValueString() != "conditional" && model.Condition.IsNull() {
		return nil
	}
	condition := model.Condition.ValueString()
	return &condition
}

func stringListToAPIReferences(ctx context.Context, list types.List, referenceType string) ([]pagerduty.APIReference, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}

	var ids []string
	diags := list.ElementsAs(ctx, &ids, false)
	if diags.HasError() || len(ids) == 0 {
		return nil, diags
	}

	refs := make([]pagerduty.APIReference, 0, len(ids))
	for _, id := range ids {
		refs = append(refs, pagerduty.APIReference{ID: id, Type: referenceType})
	}
	return refs, diags
}

func buildIncidentWorkflowTriggerPermissions(ctx context.Context, list types.List) (*pagerduty.IncidentWorkflowTriggerPermissions, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() || len(list.Elements()) == 0 {
		return nil, nil
	}

	var perms []incidentWorkflowTriggerPermissionsModel
	diags := list.ElementsAs(ctx, &perms, false)
	if diags.HasError() || len(perms) == 0 {
		return nil, diags
	}

	p := perms[0]
	return &pagerduty.IncidentWorkflowTriggerPermissions{
		Restricted: !p.Restricted.IsNull() && p.Restricted.ValueBool(),
		TeamID:     p.TeamID.ValueString(),
	}, diags
}

// flattenIncidentWorkflowTrigger converts an API response into resource
// state. prior is the state before this Read, used only to decide whether an
// empty API-returned list should collapse to null (preserving a
// never-configured attribute) or an explicit empty list — the API cannot
// distinguish the two, so a Read has to fall back to what was already there.
func flattenIncidentWorkflowTrigger(t *pagerduty.IncidentWorkflowTrigger, prior *resourceIncidentWorkflowTriggerModel) (*resourceIncidentWorkflowTriggerModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	model := &resourceIncidentWorkflowTriggerModel{
		ID:                      types.StringValue(t.ID),
		Type:                    types.StringValue(string(t.TriggerType)),
		SubscribedToAllServices: types.BoolValue(t.SubscribedToAllServices),
	}

	if t.Workflow != nil {
		model.Workflow = types.StringValue(t.Workflow.ID)
	} else {
		model.Workflow = prior.Workflow
	}

	model.Services = apiObjectsToStringListOrNull(prior.Services, t.Services)
	model.IncidentTypes = apiObjectsToStringListOrNull(prior.IncidentTypes, t.IncidentTypes)

	if t.Condition != nil {
		model.Condition = types.StringValue(*t.Condition)
	} else {
		model.Condition = types.StringNull()
	}

	permissions, d := flattenIncidentWorkflowTriggerPermissions(t.Permissions)
	diags.Append(d...)
	model.Permissions = permissions

	return model, diags
}

func apiObjectsToStringListOrNull(prior types.List, objects []pagerduty.APIObject) types.List {
	if len(objects) == 0 {
		if prior.IsNull() {
			return types.ListNull(types.StringType)
		}
		return types.ListValueMust(types.StringType, []attr.Value{})
	}

	elements := make([]attr.Value, 0, len(objects))
	for _, o := range objects {
		elements = append(elements, types.StringValue(o.ID))
	}
	return types.ListValueMust(types.StringType, elements)
}

func flattenIncidentWorkflowTriggerPermissions(p *pagerduty.IncidentWorkflowTriggerPermissions) (types.List, diag.Diagnostics) {
	objType := types.ObjectType{AttrTypes: incidentWorkflowTriggerPermissionsAttrTypes}

	if p == nil {
		return types.ListValueMust(objType, []attr.Value{}), nil
	}

	obj, diags := types.ObjectValue(incidentWorkflowTriggerPermissionsAttrTypes, map[string]attr.Value{
		"restricted": types.BoolValue(p.Restricted),
		"team_id":    types.StringValue(p.TeamID),
	})
	if diags.HasError() {
		return types.ListNull(objType), diags
	}

	list, d := types.ListValue(objType, []attr.Value{obj})
	diags.Append(d...)
	return list, diags
}
