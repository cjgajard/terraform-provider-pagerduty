package pagerduty

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/PagerDuty/terraform-provider-pagerduty/util"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

type resourceMaintenanceWindow struct{ client *pagerduty.Client }

var (
	_ resource.ResourceWithConfigure   = (*resourceMaintenanceWindow)(nil)
	_ resource.ResourceWithImportState = (*resourceMaintenanceWindow)(nil)
)

func (r *resourceMaintenanceWindow) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pagerduty_maintenance_window"
}

func (r *resourceMaintenanceWindow) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
			},
			"start_time": schema.StringAttribute{
				Required:   true,
				CustomType: timetypes.RFC3339Type{},
			},
			"end_time": schema.StringAttribute{
				Required:   true,
				CustomType: timetypes.RFC3339Type{},
			},
			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("Managed by Terraform"),
			},
			"services": schema.SetAttribute{
				Required:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *resourceMaintenanceWindow) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceMaintenanceWindowModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan := buildPagerdutyMaintenanceWindow(ctx, &model, &resp.Diagnostics)
	log.Printf("[INFO] Creating PagerDuty maintenance window")

	from := "user@email.com" // TODO
	mw, err := r.client.CreateMaintenanceWindowWithContext(ctx, from, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating PagerDuty maintenance window",
			err.Error(),
		)
		return
	}

	model = flattenMaintenanceWindow(mw, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *resourceMaintenanceWindow) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceMaintenanceWindowModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	log.Printf("[INFO] Reading PagerDuty maintenance window %s", state.ID)

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		opts := pagerduty.GetMaintenanceWindowOptions{}
		maintenanceWindow, err := r.client.GetMaintenanceWindowWithContext(ctx, state.ID.ValueString(), opts)
		if err != nil {
			if util.IsBadRequestError(err) || util.IsNotFoundError(err) {
				return retry.NonRetryableError(err)
			}
			return retry.RetryableError(err)
		}
		state = flattenMaintenanceWindow(maintenanceWindow, &resp.Diagnostics)
		return nil
	})
	if err != nil {
		if util.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading PagerDuty maintenance window %s", state.ID),
			err.Error(),
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *resourceMaintenanceWindow) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceMaintenanceWindowModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan := buildPagerdutyMaintenanceWindow(ctx, &model, &resp.Diagnostics)
	if plan.ID == "" {
		var id string
		req.State.GetAttribute(ctx, path.Root("id"), &id)
		plan.ID = id
	}
	log.Printf("[INFO] Updating PagerDuty maintenance window %s", plan.ID)

	maintenanceWindow, err := r.client.UpdateMaintenanceWindowWithContext(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error updating PagerDuty maintenance window %s", plan.ID),
			err.Error(),
		)
		return
	}
	model = flattenMaintenanceWindow(maintenanceWindow, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *resourceMaintenanceWindow) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var id types.String

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	log.Printf("[INFO] Deleting PagerDuty maintenance window %s", id)

	err := r.client.DeleteMaintenanceWindowWithContext(ctx, id.ValueString())
	if err != nil && !util.IsStatusCodeError(err, http.StatusMethodNotAllowed) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error deleting PagerDuty maintenance window %s", id),
			err.Error(),
		)
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *resourceMaintenanceWindow) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(ConfigurePagerdutyClient(&r.client, req.ProviderData)...)
}

func (r *resourceMaintenanceWindow) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type resourceMaintenanceWindowModel struct {
	ID          types.String      `tfsdk:"id"`
	StartTime   timetypes.RFC3339 `tfsdk:"start_time"`
	EndTime     timetypes.RFC3339 `tfsdk:"end_time"`
	Services    types.Set         `tfsdk:"services"`
	Description types.String      `tfsdk:"description"`
}

func buildPagerdutyMaintenanceWindow(ctx context.Context, model *resourceMaintenanceWindowModel, diags *diag.Diagnostics) pagerduty.MaintenanceWindow {
	maintenanceWindow := pagerduty.MaintenanceWindow{
		StartTime:   model.StartTime.ValueString(),
		EndTime:     model.EndTime.ValueString(),
		Services:    buildMaintenanceWindowServices(ctx, model.Services, diags),
		Description: model.Description.ValueString(),
	}
	return maintenanceWindow
}

func buildMaintenanceWindowServices(ctx context.Context, set types.Set, diags *diag.Diagnostics) []pagerduty.APIObject {
	if set.IsNull() || set.IsUnknown() {
		return []pagerduty.APIObject{}
	}

	var ids []string
	d := set.ElementsAs(ctx, &ids, false)
	diags.Append(d...)
	if d.HasError() {
		return []pagerduty.APIObject{}
	}

	list := make([]pagerduty.APIObject, 0, len(ids))
	for _, id := range ids {
		list = append(list, pagerduty.APIObject{
			Type: "service_reference",
			ID:   id,
		})
	}

	return list
}

func flattenMaintenanceWindow(window *pagerduty.MaintenanceWindow, diags *diag.Diagnostics) resourceMaintenanceWindowModel {
	startTime, d := timetypes.NewRFC3339Value(window.StartTime)
	diags.Append(d...)

	endTime, d := timetypes.NewRFC3339Value(window.EndTime)
	diags.Append(d...)

	model := resourceMaintenanceWindowModel{
		ID:          types.StringValue(window.ID),
		StartTime:   startTime,
		EndTime:     endTime,
		Description: types.StringValue(window.Description),
		Services:    flattenMaintenanceWindowServices(window.Services),
	}
	return model
}

func flattenMaintenanceWindowServices(services []pagerduty.APIObject) types.Set {
	elements := make([]attr.Value, 0, len(services))
	for _, s := range services {
		elements = append(elements, types.StringValue(s.ID))
	}
	return types.SetValueMust(types.StringType, elements)
}
