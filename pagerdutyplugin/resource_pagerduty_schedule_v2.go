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
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

type resourceScheduleV2 struct{ client *pagerduty.Client }

var (
	_ resource.Resource                = (*resourceScheduleV2)(nil)
	_ resource.ResourceWithConfigure   = (*resourceScheduleV2)(nil)
	_ resource.ResourceWithImportState = (*resourceScheduleV2)(nil)
)

func (r *resourceScheduleV2) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pagerduty_schedule_v2"
}

func (r *resourceScheduleV2) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"description": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Default:       stringdefault.StaticString("Managed by Terraform"),
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"time_zone": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"teams": schema.SetAttribute{
				Optional:      true,
				Computed:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
			},
		},
		Blocks: map[string]schema.Block{
			"rotation": schema.ListNestedBlock{
				Validators: []validator.List{listvalidator.SizeAtLeast(1)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:      true,
							PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						},
						"name": schema.StringAttribute{
							Optional:      true,
							Computed:      true,
							PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						},
						"start_time": schema.StringAttribute{
							Optional:      true,
							Computed:      true,
							PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						},
						"turn_length": schema.Int64Attribute{
							Optional:      true,
							Computed:      true,
							Validators:    []validator.Int64{int64validator.Between(3600, 365*24*3600)},
							PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
						},
						"assignment_strategy": schema.StringAttribute{
							Optional:      true,
							Computed:      true,
							Validators: []validator.String{
								stringvalidator.OneOf(
									"every_member_assignment_strategy",
									"rotating_member_assignment_strategy",
								),
							},
							PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
						},
						"shifts_per_member": schema.Int64Attribute{
							Optional:      true,
							Computed:      true,
							Validators:    []validator.Int64{int64validator.AtLeast(1)},
							PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
						},
					},
					Blocks: map[string]schema.Block{
						"member": schema.ListNestedBlock{
							Validators: []validator.List{listvalidator.SizeAtLeast(1)},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"user_id": schema.StringAttribute{
										Optional:      true,
										Computed:      true,
										PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
									},
									"name": schema.StringAttribute{
										Optional:      true,
										Computed:      true,
										PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
									},
									"color": schema.StringAttribute{
										Optional:      true,
										Computed:      true,
										PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
									},
								},
							},
						},
						"event": schema.ListNestedBlock{
							Validators: []validator.List{listvalidator.SizeAtLeast(1)},
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										Computed:      true,
										PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
									},
									"start_time": schema.StringAttribute{
										Optional:      true,
										Computed:      true,
										PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
									},
									"end_time": schema.StringAttribute{
										Optional:      true,
										Computed:      true,
										PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
									},
									"time_zone": schema.StringAttribute{
										Optional:      true,
										Computed:      true,
										PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
									},
									"effective_since": schema.StringAttribute{
										Optional:      true,
										Computed:      true,
										PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
									},
									"effective_until": schema.StringAttribute{
										Optional:      true,
										Computed:      true,
										PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
									},
									"recurrence": schema.ListAttribute{
										Optional:      true,
										Computed:      true,
										ElementType:   types.StringType,
										PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
									},
									"assignment_strategy": schema.StringAttribute{
										Optional:      true,
										Computed:      true,
										Validators: []validator.String{
											stringvalidator.OneOf(
												"every_member_assignment_strategy",
												"rotating_member_assignment_strategy",
											),
										},
										PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
									},
								},
							},
						},
						"restriction": schema.ListNestedBlock{
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"type": schema.StringAttribute{
										Computed:      true,
										PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
									},
									"start_time_of_day": schema.StringAttribute{
										Optional:      true,
										Computed:      true,
										PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
									},
									"start_day_of_week": schema.Int64Attribute{
										Optional:      true,
										Computed:      true,
										Validators:    []validator.Int64{int64validator.Between(0, 7)},
										PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
									},
									"duration_seconds": schema.Int64Attribute{
										Optional:      true,
										Computed:      true,
										Validators:    []validator.Int64{int64validator.Between(1, 7*24*3600-1)},
										PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
									},
								},
							},
						},
					},
				},
			},
			"final_schedule": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"rendered_coverage_percentage": schema.Float64Attribute{
							Computed: true,
						},
					},
					Blocks: map[string]schema.Block{
						"computed_shift_assignment": schema.ListNestedBlock{
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"start_time": schema.StringAttribute{
										Computed: true,
									},
									"end_time": schema.StringAttribute{
										Computed: true,
									},
								},
								Blocks: map[string]schema.Block{
									"member": schema.ListNestedBlock{
										NestedObject: schema.NestedBlockObject{
											Attributes: map[string]schema.Attribute{
												"type": schema.StringAttribute{
													Computed: true,
												},
												"user_id": schema.StringAttribute{
													Computed: true,
												},
												"name": schema.StringAttribute{
													Computed: true,
												},
												"color": schema.StringAttribute{
													Computed: true,
												},
											},
										},
									},
									"source": schema.ListNestedBlock{
										NestedObject: schema.NestedBlockObject{
											Attributes: map[string]schema.Attribute{
												"type": schema.StringAttribute{
													Computed: true,
												},
												"rotation_id": schema.StringAttribute{
													Computed: true,
												},
												"shift_id": schema.StringAttribute{
													Computed: true,
												},
												"shift_assignment_id": schema.StringAttribute{
													Computed: true,
												},
												"coverage_id": schema.StringAttribute{
													Computed: true,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *resourceScheduleV2) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceScheduleV2Model

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	schedule, diags := buildFlexibleScheduleStruct(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[INFO] Creating PagerDuty flexible schedule: %s", schedule.Name)

	pdSchedule := pagerduty.FlexibleSchedule{
		Name:        schedule.Name,
		Description: schedule.Description,
		TimeZone:    schedule.TimeZone,
		Rotations:   convertRotationsToPD(schedule.Rotations),
		Teams:       schedule.Teams,
	}

	var created *pagerduty.FlexibleSchedule
	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		var err error
		created, err = r.client.CreateFlexibleSchedule(pdSchedule)
		if err != nil {
			if util.IsBadRequestError(err) {
				return retry.NonRetryableError(err)
			}
			return retry.RetryableError(err)
		}
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error creating PagerDuty flexible schedule %s", schedule.Name),
			err.Error(),
		)
		return
	}

	model.ID = types.StringValue(created.ID)

	retryNotFound := true
	state, err := fetchFlexibleScheduleWithContext(ctx, r.client, model.ID.ValueString(), retryNotFound)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading PagerDuty flexible schedule %s", created.ID),
			err.Error(),
		)
		return
	}

	model, diags = flattenScheduleV2(ctx, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *resourceScheduleV2) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceScheduleV2Model

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[INFO] Reading PagerDuty flexible schedule: %s", state.ID.ValueString())

	retryNotFound := false
	schedule, err := fetchFlexibleScheduleWithContext(ctx, r.client, state.ID.ValueString(), retryNotFound)
	if err != nil {
		if util.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading PagerDuty flexible schedule %s", state.ID.ValueString()),
			err.Error(),
		)
		return
	}

	model, diags := flattenScheduleV2(ctx, schedule)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *resourceScheduleV2) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceScheduleV2Model

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	schedule, diags := buildFlexibleScheduleStruct(ctx, &model)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if model.ID.IsNull() || model.ID.IsUnknown() {
		var id types.String
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
		if resp.Diagnostics.HasError() {
			return
		}
		model.ID = id
	}

	log.Printf("[INFO] Updating PagerDuty flexible schedule: %s", model.ID.ValueString())

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		pdSchedule := pagerduty.FlexibleSchedule{
			Name:        schedule.Name,
			Description: schedule.Description,
			TimeZone:    schedule.TimeZone,
			Rotations:   convertRotationsToPD(schedule.Rotations),
			Teams:       schedule.Teams,
		}

		if _, err := r.client.UpdateFlexibleSchedule(model.ID.ValueString(), pdSchedule); err != nil {
			if util.IsBadRequestError(err) {
				return retry.NonRetryableError(err)
			}
			return retry.RetryableError(err)
		}
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error updating PagerDuty flexible schedule %s", model.ID.ValueString()),
			err.Error(),
		)
		return
	}

	retryNotFound := false
	updated, err := fetchFlexibleScheduleWithContext(ctx, r.client, model.ID.ValueString(), retryNotFound)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading PagerDuty flexible schedule after update %s", model.ID.ValueString()),
			err.Error(),
		)
		return
	}

	model, diags = flattenScheduleV2(ctx, updated)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *resourceScheduleV2) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var id types.String

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}

	log.Printf("[INFO] Deleting PagerDuty flexible schedule: %s", id.ValueString())

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		if err := r.client.DeleteFlexibleSchedule(id.ValueString()); err != nil {
			if util.IsNotFoundError(err) {
				return retry.NonRetryableError(err)
			}
			return retry.RetryableError(err)
		}
		return nil
	})
	if err != nil && !util.IsNotFoundError(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error deleting PagerDuty flexible schedule %s", id.ValueString()),
			err.Error(),
		)
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *resourceScheduleV2) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(ConfigurePagerdutyClient(&r.client, req.ProviderData)...)
}

func (r *resourceScheduleV2) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// Model types
type resourceScheduleV2Model struct {
	ID            types.String         `tfsdk:"id"`
	Name          types.String         `tfsdk:"name"`
	Description   types.String         `tfsdk:"description"`
	TimeZone      types.String         `tfsdk:"time_zone"`
	Rotation      []rotationModel      `tfsdk:"rotation"`
	Teams         types.Set            `tfsdk:"teams"`
	FinalSchedule []finalScheduleModel `tfsdk:"final_schedule"`
}

type rotationModel struct {
	ID                 types.String       `tfsdk:"id"`
	Name               types.String       `tfsdk:"name"`
	StartTime          types.String       `tfsdk:"start_time"`
	TurnLength         types.Int64        `tfsdk:"turn_length"`
	AssignmentStrategy types.String       `tfsdk:"assignment_strategy"`
	ShiftsPerMember    types.Int64        `tfsdk:"shifts_per_member"`
	Member             []memberModel      `tfsdk:"member"`
	Event              []eventModel       `tfsdk:"event"`
	Restriction        []restrictionModel `tfsdk:"restriction"`
}

type memberModel struct {
	UserID types.String `tfsdk:"user_id"`
	Name   types.String `tfsdk:"name"`
	Color  types.String `tfsdk:"color"`
}

type eventModel struct {
	ID                 types.String `tfsdk:"id"`
	StartTime          types.String `tfsdk:"start_time"`
	EndTime            types.String `tfsdk:"end_time"`
	TimeZone           types.String `tfsdk:"time_zone"`
	EffectiveSince     types.String `tfsdk:"effective_since"`
	EffectiveUntil     types.String `tfsdk:"effective_until"`
	Recurrence         types.List   `tfsdk:"recurrence"`
	AssignmentStrategy types.String `tfsdk:"assignment_strategy"`
}

type restrictionModel struct {
	Type            types.String `tfsdk:"type"`
	StartTimeOfDay  types.String `tfsdk:"start_time_of_day"`
	StartDayOfWeek  types.Int64  `tfsdk:"start_day_of_week"`
	DurationSeconds types.Int64  `tfsdk:"duration_seconds"`
}

type finalScheduleModel struct {
	RenderedCoveragePercentage types.Float64                  `tfsdk:"rendered_coverage_percentage"`
	ComputedShiftAssignment    []computedShiftAssignmentModel `tfsdk:"computed_shift_assignment"`
}

type computedShiftAssignmentModel struct {
	StartTime types.String                         `tfsdk:"start_time"`
	EndTime   types.String                         `tfsdk:"end_time"`
	Member    []shiftMemberModel                   `tfsdk:"member"`
	Source    []computedShiftAssignmentSourceModel `tfsdk:"source"`
}

type shiftMemberModel struct {
	Type   types.String `tfsdk:"type"`
	UserID types.String `tfsdk:"user_id"`
	Name   types.String `tfsdk:"name"`
	Color  types.String `tfsdk:"color"`
}

type computedShiftAssignmentSourceModel struct {
	Type              types.String `tfsdk:"type"`
	RotationID        types.String `tfsdk:"rotation_id"`
	ShiftID           types.String `tfsdk:"shift_id"`
	ShiftAssignmentID types.String `tfsdk:"shift_assignment_id"`
	CoverageID        types.String `tfsdk:"coverage_id"`
}

// Helper structs for API conversion
type FlexibleSchedule struct {
	ID            string
	Type          string
	Name          string
	Description   string
	TimeZone      string
	Rotations     []*ScheduleRotation
	FinalSchedule *FinalSchedule
	Teams         []pagerduty.APIObject
}

type ScheduleRotation struct {
	ID     string
	Type   string
	Name   string
	Events []*ScheduleEvent
}

type ScheduleEvent struct {
	ID                 string
	Type               string
	StartTime          *ZonedDateTime
	EndTime            *ZonedDateTime
	EffectiveSince     string
	EffectiveUntil     *string
	Recurrence         []string
	AssignmentStrategy *AssignmentStrategy
}

type ZonedDateTime struct {
	DateTime string
	TimeZone string
}

type AssignmentStrategy struct {
	Type            string
	ShiftsPerMember int
	Members         []*ShiftMember
}

type ShiftMember struct {
	Type   string
	UserID string
	Name   string
	Color  string
}

type FinalSchedule struct {
	Type                       string
	RenderedCoveragePercentage float64
	ComputedShiftAssignments   []*ComputedShiftAssignment
}

type ComputedShiftAssignment struct {
	Type      string
	StartTime string
	EndTime   string
	Member    *ShiftMember
	Source    *ComputedShiftAssignmentSource
}

type ComputedShiftAssignmentSource struct {
	Type              string
	RotationID        string
	ShiftID           string
	ShiftAssignmentID string
	CoverageID        string
}

// Build functions
func buildFlexibleScheduleStruct(ctx context.Context, model *resourceScheduleV2Model) (*FlexibleSchedule, diag.Diagnostics) {
	var diags diag.Diagnostics

	rotations, d := expandScheduleRotations(ctx, model.Rotation)
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	schedule := &FlexibleSchedule{
		Name:        model.Name.ValueString(),
		Description: model.Description.ValueString(),
		TimeZone:    model.TimeZone.ValueString(),
		Rotations:   rotations,
		Type:        "schedule",
	}

	if !model.Teams.IsNull() && !model.Teams.IsUnknown() {
		var teams []string
		diags.Append(model.Teams.ElementsAs(ctx, &teams, false)...)
		if !diags.HasError() {
			schedule.Teams = expandScheduleTeams(teams)
		}
	}

	return schedule, diags
}

func expandScheduleRotations(ctx context.Context, rotations []rotationModel) ([]*ScheduleRotation, diag.Diagnostics) {
	var diags diag.Diagnostics
	var result []*ScheduleRotation

	for _, rotation := range rotations {
		scheduleRotation := &ScheduleRotation{
			Name: rotation.Name.ValueString(),
			Type: "schedule_rotation",
		}

		events, d := expandScheduleEvents(ctx, rotation.Event)
		diags.Append(d...)
		if diags.HasError() {
			return nil, diags
		}
		scheduleRotation.Events = events

		result = append(result, scheduleRotation)
	}

	return result, diags
}

func expandScheduleEvents(ctx context.Context, events []eventModel) ([]*ScheduleEvent, diag.Diagnostics) {
	var diags diag.Diagnostics
	var result []*ScheduleEvent

	for _, event := range events {
		scheduleEvent := &ScheduleEvent{
			Type:           "schedule_event",
			EffectiveSince: event.EffectiveSince.ValueString(),
		}

		if !event.StartTime.IsNull() && !event.StartTime.IsUnknown() {
			scheduleEvent.StartTime = &ZonedDateTime{
				DateTime: event.StartTime.ValueString(),
			}
			if !event.TimeZone.IsNull() && !event.TimeZone.IsUnknown() {
				scheduleEvent.StartTime.TimeZone = event.TimeZone.ValueString()
			}
		}

		if !event.EndTime.IsNull() && !event.EndTime.IsUnknown() {
			scheduleEvent.EndTime = &ZonedDateTime{
				DateTime: event.EndTime.ValueString(),
			}
			if !event.TimeZone.IsNull() && !event.TimeZone.IsUnknown() && scheduleEvent.StartTime != nil {
				scheduleEvent.EndTime.TimeZone = event.TimeZone.ValueString()
			}
		}

		if !event.EffectiveUntil.IsNull() && !event.EffectiveUntil.IsUnknown() {
			until := event.EffectiveUntil.ValueString()
			scheduleEvent.EffectiveUntil = &until
		}

		if !event.Recurrence.IsNull() && !event.Recurrence.IsUnknown() {
			var recurrenceRules []string
			diags.Append(event.Recurrence.ElementsAs(ctx, &recurrenceRules, false)...)
			if !diags.HasError() {
				scheduleEvent.Recurrence = recurrenceRules
			}
		}

		if !event.AssignmentStrategy.IsNull() && !event.AssignmentStrategy.IsUnknown() {
			scheduleEvent.AssignmentStrategy = &AssignmentStrategy{
				Type: event.AssignmentStrategy.ValueString(),
			}
		}

		result = append(result, scheduleEvent)
	}

	return result, diags
}

func expandScheduleTeams(teams []string) []pagerduty.APIObject {
	var result []pagerduty.APIObject

	for _, t := range teams {
		team := pagerduty.APIObject{
			ID:   t,
			Type: "team_reference",
		}
		result = append(result, team)
	}

	return result
}

// Convert functions
func convertRotationsToPD(rotations []*ScheduleRotation) []pagerduty.ScheduleRotation {
	var pdRotations []pagerduty.ScheduleRotation

	for _, rotation := range rotations {
		pdRotation := pagerduty.ScheduleRotation{
			APIObject: pagerduty.APIObject{
				ID:   rotation.ID,
				Type: rotation.Type,
			},
			Name:   rotation.Name,
			Events: convertEventsToPD(rotation.Events),
		}
		pdRotations = append(pdRotations, pdRotation)
	}

	return pdRotations
}

func convertEventsToPD(events []*ScheduleEvent) []pagerduty.ScheduleEvent {
	var pdEvents []pagerduty.ScheduleEvent

	for _, event := range events {
		pdEvent := pagerduty.ScheduleEvent{
			APIObject: pagerduty.APIObject{
				ID:   event.ID,
				Type: event.Type,
			},
			EffectiveSince: event.EffectiveSince,
			Recurrence:     event.Recurrence,
		}

		if event.StartTime != nil {
			pdEvent.StartTime = &pagerduty.ZonedDateTime{
				DateTime: event.StartTime.DateTime,
				TimeZone: event.StartTime.TimeZone,
			}
		}

		if event.EndTime != nil {
			pdEvent.EndTime = &pagerduty.ZonedDateTime{
				DateTime: event.EndTime.DateTime,
				TimeZone: event.EndTime.TimeZone,
			}
		}

		if event.EffectiveUntil != nil {
			pdEvent.EffectiveUntil = event.EffectiveUntil
		}

		if event.AssignmentStrategy != nil {
			pdEvent.AssignmentStrategy = &pagerduty.AssignmentStrategy{
				Type:            event.AssignmentStrategy.Type,
				ShiftsPerMember: event.AssignmentStrategy.ShiftsPerMember,
				Members:         convertMembersToPD(event.AssignmentStrategy.Members),
			}
		}

		pdEvents = append(pdEvents, pdEvent)
	}

	return pdEvents
}

func convertMembersToPD(members []*ShiftMember) []pagerduty.ShiftMember {
	var pdMembers []pagerduty.ShiftMember

	for _, member := range members {
		pdMember := pagerduty.ShiftMember{
			Type:   member.Type,
			UserID: member.UserID,
			Name:   member.Name,
			Color:  member.Color,
		}
		pdMembers = append(pdMembers, pdMember)
	}

	return pdMembers
}

// Flatten functions
func flattenScheduleV2(ctx context.Context, schedule *pagerduty.FlexibleSchedule) (resourceScheduleV2Model, diag.Diagnostics) {
	var diags diag.Diagnostics

	model := resourceScheduleV2Model{
		ID:          types.StringValue(schedule.ID),
		Name:        types.StringValue(schedule.Name),
		Description: types.StringValue(schedule.Description),
		TimeZone:    types.StringValue(schedule.TimeZone),
	}

	if len(schedule.Rotations) > 0 {
		rotations, d := flattenScheduleRotations(ctx, schedule.Rotations)
		diags.Append(d...)
		model.Rotation = rotations
	}

	if len(schedule.Teams) > 0 {
		teams := flattenScheduleTeams(schedule.Teams)
		teamsSet, d := types.SetValueFrom(ctx, types.StringType, teams)
		diags.Append(d...)
		model.Teams = teamsSet
	} else {
		model.Teams = types.SetNull(types.StringType)
	}

	if schedule.FinalSchedule != nil {
		finalSchedule := flattenFinalSchedule(ctx, schedule.FinalSchedule)
		model.FinalSchedule = finalSchedule
	}

	return model, diags
}

func flattenScheduleRotations(ctx context.Context, rotations []pagerduty.ScheduleRotation) ([]rotationModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var result []rotationModel

	for _, rotation := range rotations {
		r := rotationModel{
			ID:   types.StringValue(rotation.ID),
			Name: types.StringValue(rotation.Name),
		}

		if len(rotation.Events) > 0 {
			events, d := flattenScheduleEvents(ctx, rotation.Events)
			diags.Append(d...)
			r.Event = events
		}

		result = append(result, r)
	}

	return result, diags
}

func flattenScheduleEvents(ctx context.Context, events []pagerduty.ScheduleEvent) ([]eventModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var result []eventModel

	for _, event := range events {
		e := eventModel{
			ID:             types.StringValue(event.ID),
			EffectiveSince: types.StringValue(event.EffectiveSince),
		}

		if event.StartTime != nil {
			e.StartTime = types.StringValue(event.StartTime.DateTime)
			e.TimeZone = types.StringValue(event.StartTime.TimeZone)
		}

		if event.EndTime != nil {
			e.EndTime = types.StringValue(event.EndTime.DateTime)
			if e.TimeZone.IsNull() || e.TimeZone.ValueString() == "" {
				e.TimeZone = types.StringValue(event.EndTime.TimeZone)
			}
		}

		if event.EffectiveUntil != nil {
			e.EffectiveUntil = types.StringValue(*event.EffectiveUntil)
		}

		if len(event.Recurrence) > 0 {
			recurrenceList, d := types.ListValueFrom(ctx, types.StringType, event.Recurrence)
			diags.Append(d...)
			e.Recurrence = recurrenceList
		} else {
			e.Recurrence = types.ListNull(types.StringType)
		}

		if event.AssignmentStrategy != nil {
			e.AssignmentStrategy = types.StringValue(event.AssignmentStrategy.Type)
		}

		result = append(result, e)
	}

	return result, diags
}

func flattenScheduleTeams(teams []pagerduty.APIObject) []string {
	res := make([]string, len(teams))
	for i, t := range teams {
		res[i] = t.ID
	}
	return res
}

func flattenFinalSchedule(ctx context.Context, finalSchedule *pagerduty.FinalSchedule) []finalScheduleModel {
	result := []finalScheduleModel{
		{
			RenderedCoveragePercentage: types.Float64Value(finalSchedule.RenderedCoveragePercentage),
			ComputedShiftAssignment:    flattenComputedShiftAssignments(finalSchedule.ComputedShiftAssignments),
		},
	}
	return result
}

func flattenComputedShiftAssignments(assignments []pagerduty.ComputedShiftAssignment) []computedShiftAssignmentModel {
	var result []computedShiftAssignmentModel

	for _, assignment := range assignments {
		a := computedShiftAssignmentModel{
			StartTime: types.StringValue(assignment.StartTime),
			EndTime:   types.StringValue(assignment.EndTime),
		}

		if assignment.Member != nil {
			a.Member = []shiftMemberModel{
				{
					Type:   types.StringValue(assignment.Member.Type),
					UserID: types.StringValue(assignment.Member.UserID),
					Name:   types.StringValue(assignment.Member.Name),
					Color:  types.StringValue(assignment.Member.Color),
				},
			}
		}

		if assignment.Source != nil {
			a.Source = []computedShiftAssignmentSourceModel{
				{
					Type:              types.StringValue(assignment.Source.Type),
					RotationID:        types.StringValue(assignment.Source.RotationID),
					ShiftID:           types.StringValue(assignment.Source.ShiftID),
					ShiftAssignmentID: types.StringValue(assignment.Source.ShiftAssignmentID),
					CoverageID:        types.StringValue(assignment.Source.CoverageID),
				},
			}
		}

		result = append(result, a)
	}

	return result
}

// Fetch function
func fetchFlexibleScheduleWithContext(ctx context.Context, client *pagerduty.Client, id string, retryNotFound bool) (*pagerduty.FlexibleSchedule, error) {
	var schedule *pagerduty.FlexibleSchedule

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		var err error
		schedule, err = client.GetFlexibleSchedule(id, pagerduty.GetFlexibleScheduleOptions{})
		if err != nil {
			log.Printf("[WARN] Flexible schedule read error")
			if util.IsBadRequestError(err) {
				return retry.NonRetryableError(err)
			}

			if util.IsNotFoundError(err) {
				if !retryNotFound {
					return retry.NonRetryableError(err)
				}
				time.Sleep(2 * time.Second)
				return retry.RetryableError(err)
			}
			return retry.RetryableError(err)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return schedule, nil
}
