package pagerduty

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/PagerDuty/terraform-provider-pagerduty/util"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

type resourceIncidentCustomFieldOption struct{ client *pagerduty.Client }

var (
	_ resource.ResourceWithConfigure   = (*resourceIncidentCustomFieldOption)(nil)
	_ resource.ResourceWithImportState = (*resourceIncidentCustomFieldOption)(nil)
	// _ resource.ResourceWithValidateConfig = (*resourceIncidentCustomFieldOption)(nil)
)

func (r *resourceIncidentCustomFieldOption) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pagerduty_incident_custom_field_option"
}

func (r *resourceIncidentCustomFieldOption) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"data_type": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf("string"),
				},
			},
			"field": schema.StringAttribute{Required: true},
			"value": schema.StringAttribute{Required: true},
		},
	}
}

// func (r *resourceIncidentCustomFieldOption) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
// 	var model resourceIncidentCustomFieldOptionModel
//
// 	d := req.Config.Get(ctx, &model)
// 	if resp.Diagnostics.Append(d...); d.HasError() {
// 		return
// 	}
//
// 	err := validateIncidentCustomFieldValue(value.ValueString(), datatype.ValueString(), false,  func() error {
// 		return fmt.Errorf("invalid value for data_type %v: %v", datatype, value)
// 	})
// 	if err != nil {
// 		resp.Diagnostics.AddError(err.Error(), "")
// 	}
// }

func (r *resourceIncidentCustomFieldOption) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceIncidentCustomFieldOptionModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan := buildCustomFieldOption(&model)
	log.Printf("[INFO] Creating PagerDuty field option for field %s", plan.FieldID)

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		response, err := r.client.CreateCustomFieldOptionWithContext(ctx, plan.FieldID, plan.CustomFieldOption)
		if err != nil {
			if util.IsBadRequestError(err) {
				return retry.NonRetryableError(err)
			}
			return retry.RetryableError(err)
		}
		model = flattenCustomFieldOption(plan.FieldID, response)
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error creating PagerDuty field option"),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *resourceIncidentCustomFieldOption) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var id types.String
	var fieldID types.String

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("field"), &fieldID)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	log.Printf("[INFO] Reading PagerDuty field option %s", id)

	var found *pagerduty.CustomFieldOption
	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		resp, err := r.client.ListCustomFieldOptionsWithContext(ctx, fieldID.ValueString())
		if err != nil {
			if util.IsBadRequestError(err) {
				return retry.NonRetryableError(err)
			}
			if util.IsNotFoundError(err) {
				return retry.NonRetryableError(err)
			}
			return retry.RetryableError(err)
		}

		for _, o := range resp.FieldOptions {
			if o.ID == id.ValueString() {
				found = &o
				return nil
			}
		}

		return nil
	})
	if err != nil {
		if util.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading PagerDuty field option %s", id),
			err.Error(),
		)
		return
	}

	if found == nil {
		resp.Diagnostics.AddWarning(
			fmt.Sprintf("Unable to locate any field option with id: %s", id),
			"",
		)
		resp.State.RemoveResource(ctx)
		return
	}

	model := flattenCustomFieldOption(fieldID.ValueString(), found)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *resourceIncidentCustomFieldOption) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceIncidentCustomFieldOptionModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan := buildCustomFieldOption(&model)
	log.Printf("[INFO] Updating PagerDuty field option %s", plan.ID)

	fieldOption, err := r.client.UpdateCustomFieldOptionWithContext(ctx, plan.FieldID, plan.CustomFieldOption)
	if err != nil {
		if util.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error updating PagerDuty field option %s", plan.ID),
			err.Error(),
		)
		return
	}
	model = flattenCustomFieldOption(plan.FieldID, fieldOption)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *resourceIncidentCustomFieldOption) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var id types.String
	var fieldID types.String

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("field"), &fieldID)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	log.Printf("[INFO] Deleting PagerDuty field option %s for field %s", id, fieldID)

	err := r.client.DeleteCustomFieldOptionWithContext(ctx, fieldID.ValueString(), id.ValueString())
	if err != nil && !util.IsNotFoundError(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error deleting PagerDuty field option %s", id),
			err.Error(),
		)
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *resourceIncidentCustomFieldOption) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(ConfigurePagerdutyClient(&r.client, req.ProviderData)...)
}

func (r *resourceIncidentCustomFieldOption) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type resourceIncidentCustomFieldOptionModel struct {
	ID       types.String `tfsdk:"id"`
	DataType types.String `tfsdk:"data_type"`
	Value    types.String `tfsdk:"value"`
	Field    types.String `tfsdk:"field"`
}

type customFieldOptionPayload struct {
	pagerduty.CustomFieldOption
	FieldID string
}

func buildCustomFieldOption(model *resourceIncidentCustomFieldOptionModel) *customFieldOptionPayload {
	return &customFieldOptionPayload{
		CustomFieldOption: pagerduty.CustomFieldOption{
			APIReference: pagerduty.APIReference{
				ID: model.ID.ValueString(),
			},
			Data: &pagerduty.CustomFieldOptionData{
				DataType: model.DataType.ValueString(),
				Value:    model.Value.ValueString(),
			},
		},
		FieldID: model.Field.ValueString(),
	}
}

func flattenCustomFieldOption(fieldID string, response *pagerduty.CustomFieldOption) resourceIncidentCustomFieldOptionModel {
	model := resourceIncidentCustomFieldOptionModel{
		ID: types.StringValue(response.ID),
		Field: types.StringValue(fieldID),
	}
	if response != nil && response.Data != nil {
		model.DataType = types.StringValue(response.Data.DataType)
		model.Value = types.StringValue(response.Data.Value)
	}
	return model
}
