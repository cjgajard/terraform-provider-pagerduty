package pagerduty

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/PagerDuty/terraform-provider-pagerduty/util"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

type resourceIncidentCustomField struct{ client *pagerduty.Client }

var (
	_ resource.ResourceWithConfigure      = (*resourceIncidentCustomField)(nil)
	_ resource.ResourceWithImportState    = (*resourceIncidentCustomField)(nil)
	_ resource.ResourceWithValidateConfig = (*resourceIncidentCustomField)(nil)
)

func (r *resourceIncidentCustomField) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pagerduty_incident_custom_field"
}

func (r *resourceIncidentCustomField) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name":          schema.StringAttribute{Required: true},
			"display_name":  schema.StringAttribute{Required: true},
			"description":   schema.StringAttribute{Optional: true},
			"default_value": schema.StringAttribute{Optional: true},
			"data_type":     schema.StringAttribute{Required: true},
			"field_type":    schema.StringAttribute{Required: true},
		},
	}
}

func (r *resourceIncidentCustomField) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	validateCustomFieldDataType(ctx, req, resp)
	validateCustomFieldFieldType(ctx, req, resp)
}

func (r *resourceIncidentCustomField) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceIncidentCustomFieldModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan := buildPagerdutyIncidentCustomField(&model, &resp.Diagnostics)
	log.Printf("[INFO] Creating PagerDuty incident custom field %s", plan.Name)

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		response, err := r.client.CreateCustomFieldWithContext(ctx, plan)
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
			fmt.Sprintf("Error creating PagerDuty incident custom field %s", plan.Name),
			err.Error(),
		)
		return
	}

	model, err = requestGetIncidentCustomField(ctx, r.client, plan.ID, true, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading PagerDuty incident custom field %s", plan.ID),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *resourceIncidentCustomField) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var id types.String

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	log.Printf("[INFO] Reading PagerDuty incident custom field %s", id)

	state, err := requestGetIncidentCustomField(ctx, r.client, id.ValueString(), false, &resp.Diagnostics)
	if err != nil {
		if util.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading PagerDuty incident custom field %s", id),
			err.Error(),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *resourceIncidentCustomField) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceIncidentCustomFieldModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan := buildPagerdutyIncidentCustomField(&model, &resp.Diagnostics)
	log.Printf("[INFO] Updating PagerDuty incident custom field %s", plan.ID)

	incidentCustomField, err := r.client.UpdateCustomFieldWithContext(ctx, plan)
	if err != nil {
		if util.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error updating PagerDuty incident custom field %s", plan.ID),
			err.Error(),
		)
		return
	}
	model = flattenIncidentCustomField(incidentCustomField, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *resourceIncidentCustomField) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var id types.String

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	log.Printf("[INFO] Deleting PagerDuty incident custom field %s", id)

	err := r.client.DeleteCustomFieldWithContext(ctx, id.ValueString())
	if err != nil && !util.IsNotFoundError(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error deleting PagerDuty incident custom field %s", id),
			err.Error(),
		)
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *resourceIncidentCustomField) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(ConfigurePagerdutyClient(&r.client, req.ProviderData)...)
}

func (r *resourceIncidentCustomField) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type resourceIncidentCustomFieldModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	DisplayName  types.String `tfsdk:"display_name"`
	Description  types.String `tfsdk:"description"`
	DefaultValue types.String `tfsdk:"default_value"`
	DataType     types.String `tfsdk:"data_type"`
	FieldType    types.String `tfsdk:"field_type"`
}

func requestGetIncidentCustomField(ctx context.Context, client *pagerduty.Client, id string, retryNotFound bool, diags *diag.Diagnostics) (resourceIncidentCustomFieldModel, error) {
	var model resourceIncidentCustomFieldModel

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		incidentCustomField, err := client.GetCustomFieldWithContext(ctx, id, pagerduty.GetCustomFieldOptions{})
		if err != nil {
			if util.IsBadRequestError(err) {
				return retry.NonRetryableError(err)
			}
			if !retryNotFound && util.IsNotFoundError(err) {
				return retry.NonRetryableError(err)
			}
			return retry.RetryableError(err)
		}
		model = flattenIncidentCustomField(incidentCustomField, diags)
		return nil
	})

	return model, err
}

func buildPagerdutyIncidentCustomField(model *resourceIncidentCustomFieldModel, diags *diag.Diagnostics) pagerduty.CustomField {
	// Description  len<=1000
	// DataType     one of: boolean integer float string datetime url(len<=200)
	// FieldType    one of: single_value single_value_fixed multi_value multi_value_fixed
	return pagerduty.CustomField{
		APIObject:    pagerduty.APIObject{ID: model.ID.ValueString()},
		Name:         model.Name.ValueString(),
		DisplayName:  model.DisplayName.ValueString(),
		DataType:     model.DataType.ValueString(),
		FieldType:    model.FieldType.ValueString(),
		Description:  model.Description.ValueString(),
		DefaultValue: buildPagerdutyIncidentCustomFieldDefaultValue(model, diags),
	}
}

func buildPagerdutyIncidentCustomFieldDefaultValue(model *resourceIncidentCustomFieldModel, diags *diag.Diagnostics) interface{} {
	if model.DefaultValue.IsNull() || model.DefaultValue.IsUnknown() {
		return nil
	}
	switch model.FieldType.ValueString() {
	case "string":
		return model.DefaultValue.ValueString()
	default:
		diags.AddError("A field_type other than string is not supported yet", "")
		return nil
	}
}

func flattenIncidentCustomField(response *pagerduty.CustomField, diags *diag.Diagnostics) resourceIncidentCustomFieldModel {
	model := resourceIncidentCustomFieldModel{
		ID:          types.StringValue(response.ID),
		Name:        types.StringValue(response.Name),
		DisplayName: types.StringValue(response.DisplayName),
		DataType:    types.StringValue(response.DataType),
		FieldType:   types.StringValue(response.FieldType),
	}
	if response.Description != "" {
		model.Description = types.StringValue(response.Description)
	}
	if !util.IsNilFunc(response.DefaultValue) {
		model.DefaultValue = flattenIncidentCustomFieldDefaultValue(response.DefaultValue, diags)
	}
	return model
}

func flattenIncidentCustomFieldDefaultValue(defaultValue interface{}, diags *diag.Diagnostics) types.String {
	if isCustomFieldMultiValue(defaultValue) {
		b, err := json.Marshal(defaultValue)
		if err != nil {
			diags.AddError("Cannot parse field's default value", err.Error())
			return types.StringNull()
		}
		return types.StringValue(string(b))
	}
	return types.StringValue(fmt.Sprintf("%v", defaultValue))
}

func isCustomFieldMultiValue(fieldValue interface{}) bool {
	v, ok := fieldValue.(string)
	if !ok {
		return false
	}
	return v == "multi_value" || v == "multi_value_fixed"
}

var validateCustomFieldDataTypeAllowed = map[string]struct{}{
	"string":   {},
	"integer":  {},
	"float":    {},
	"boolean":  {},
	"url":      {},
	"datetime": {},
}

func validateCustomFieldDataType(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var dataType types.String
	dataTypePath := path.Root("data_type")

	d := req.Config.GetAttribute(ctx, dataTypePath, &dataType)
	if resp.Diagnostics.Append(d...); d.HasError() {
		return
	}

	if _, ok := validateCustomFieldDataTypeAllowed[dataType.ValueString()]; !ok {
		resp.Diagnostics.AddAttributeError(dataTypePath, fmt.Sprintf("Unknown data_type %v", dataType.ValueString()), "")
	}
}

var validateCustomFieldFieldTypeAllowed = map[string]struct{}{
	"single_value":       {},
	"single_value_fixed": {},
	"multi_value":        {},
	"multi_value_fixed":  {},
}

func validateCustomFieldFieldType(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var fieldType types.String
	fieldTypePath := path.Root("field_type")

	d := req.Config.GetAttribute(ctx, fieldTypePath, &fieldType)
	if resp.Diagnostics.Append(d...); d.HasError() {
		return
	}

	if _, ok := validateCustomFieldFieldTypeAllowed[fieldType.ValueString()]; !ok {
		resp.Diagnostics.AddAttributeError(fieldTypePath, fmt.Sprintf("Unknown field_type %v", fieldType.ValueString()), "")
	}
}
