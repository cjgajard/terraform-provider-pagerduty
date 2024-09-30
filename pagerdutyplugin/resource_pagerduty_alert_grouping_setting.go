package pagerduty

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/PagerDuty/terraform-provider-pagerduty/util"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

type resourceAlertGroupingSetting struct{ client *pagerduty.Client }

var (
	_ resource.ResourceWithConfigure   = (*resourceAlertGroupingSetting)(nil)
	_ resource.ResourceWithImportState = (*resourceAlertGroupingSetting)(nil)
)

func (r *resourceAlertGroupingSetting) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pagerduty_alert_grouping_setting"
}

func (r *resourceAlertGroupingSetting) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			},
			"description": schema.StringAttribute{
				Computed: true,
				Optional: true,
				Default:  stringdefault.StaticString("Managed by Terraform"),
			},
			"type": schema.StringAttribute{
				Computed: true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"content_based",
						"content_based_intelligent",
						"intelligent",
						"time",
					),
				},
			},
			"services": schema.ListAttribute{
				ElementType: types.StringType,
				Optional:    true,
			},

			"config_content_based": schema.ObjectAttribute{
				AttributeTypes: map[string]attr.Type{
					"time_window": types.Int64Type,
					"aggregate":   types.StringType,
					"fields":      types.ListType{ElemType: types.StringType},
				},
				Optional: true,
				Validators: []validator.Object{
					objectvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("config_content_based_intelligent"),
						path.MatchRelative().AtParent().AtName("config_intelligent"),
						path.MatchRelative().AtParent().AtName("config_time"),
					),
				},
			},

			"config_content_based_intelligent": schema.ObjectAttribute{
				AttributeTypes: map[string]attr.Type{
					"time_window": types.Int64Type,
					"aggregate":   types.StringType,
					"fields":      types.ListType{ElemType: types.StringType},
				},
				Optional: true,
				Validators: []validator.Object{
					objectvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("config_content_based"),
						path.MatchRelative().AtParent().AtName("config_intelligent"),
						path.MatchRelative().AtParent().AtName("config_time"),
					),
				},
			},

			"config_intelligent": schema.ObjectAttribute{
				AttributeTypes: map[string]attr.Type{
					"time_window": types.Int64Type,
				},
				Optional: true,
				Validators: []validator.Object{
					objectvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("config_content_based"),
						path.MatchRelative().AtParent().AtName("config_content_based_intelligent"),
						path.MatchRelative().AtParent().AtName("config_time"),
					),
				},
			},

			"config_time": schema.ObjectAttribute{
				AttributeTypes: map[string]attr.Type{
					"timeout": types.Int64Type,
				},
				Optional: true,
				Validators: []validator.Object{
					objectvalidator.ConflictsWith(
						path.MatchRelative().AtParent().AtName("config_content_based"),
						path.MatchRelative().AtParent().AtName("config_content_based_intelligent"),
						path.MatchRelative().AtParent().AtName("config_intelligent"),
					),
				},
			},
		},
	}
}

func (r *resourceAlertGroupingSetting) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceAlertGroupingSettingModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan := buildPagerdutyAlertGroupingSetting(ctx, &model, &resp.Diagnostics)
	log.Printf("[INFO] Creating PagerDuty alert grouping setting %s", plan.Name)

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		response, err := r.client.CreateAlertGroupingSetting(ctx, plan)
		if err != nil {
			return retry.RetryableError(err)
		}
		plan.ID = response.ID
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error creating PagerDuty alert grouping setting %s", plan.Name),
			err.Error(),
		)
		return
	}

	model, err = requestGetAlertGroupingSetting(ctx, r.client, plan.ID, true, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading PagerDuty alert grouping setting %s", plan.ID),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *resourceAlertGroupingSetting) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var id types.String

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	log.Printf("[INFO] Reading PagerDuty alert grouping setting %s", id)

	state, err := requestGetAlertGroupingSetting(ctx, r.client, id.ValueString(), false, &resp.Diagnostics)
	if err != nil {
		if util.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading PagerDuty alert grouping setting %s", id),
			err.Error(),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *resourceAlertGroupingSetting) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceAlertGroupingSettingModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan := buildPagerdutyAlertGroupingSetting(ctx, &model, &resp.Diagnostics)
	log.Printf("[INFO] Updating PagerDuty alert grouping setting %s", plan.ID)

	alertGroupingSetting, err := r.client.UpdateAlertGroupingSetting(ctx, plan)
	if err != nil {
		if util.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error updating PagerDuty alert grouping setting %s", plan.ID),
			err.Error(),
		)
		return
	}
	model = flattenAlertGroupingSetting(alertGroupingSetting)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *resourceAlertGroupingSetting) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var id types.String

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	log.Printf("[INFO] Deleting PagerDuty alert grouping setting %s", id)

	err := r.client.DeleteAlertGroupingSetting(ctx, id.ValueString())
	if err != nil && !util.IsNotFoundError(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error deleting PagerDuty alert grouping setting %s", id),
			err.Error(),
		)
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *resourceAlertGroupingSetting) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(ConfigurePagerdutyClient(&r.client, req.ProviderData)...)
}

func (r *resourceAlertGroupingSetting) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type resourceAlertGroupingSettingModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	Type               types.String `tfsdk:"type"`
	ConfigContentBased types.Object `tfsdk:"config_content_based"`
	ConfigCBI          types.Object `tfsdk:"config_content_based_intelligent"`
	ConfigIntelligent  types.Object `tfsdk:"config_intelligent"`
	ConfigTime         types.Object `tfsdk:"config_time"`
	Services           types.List   `tfsdk:"services"`
}

func requestGetAlertGroupingSetting(ctx context.Context, client *pagerduty.Client, id string, retryNotFound bool, diags *diag.Diagnostics) (resourceAlertGroupingSettingModel, error) {
	var model resourceAlertGroupingSettingModel

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		alertGroupingSetting, err := client.GetAlertGroupingSetting(ctx, id)
		if err != nil {
			if util.IsBadRequestError(err) {
				return retry.NonRetryableError(err)
			}
			if !retryNotFound && util.IsNotFoundError(err) {
				return retry.NonRetryableError(err)
			}
			return retry.RetryableError(err)
		}
		model = flattenAlertGroupingSetting(alertGroupingSetting)
		return nil
	})

	return model, err
}

func buildPagerdutyAlertGroupingSetting(ctx context.Context, model *resourceAlertGroupingSettingModel, diags *diag.Diagnostics) pagerduty.AlertGroupingSetting {
	configType := buildPagerdutyAlertGroupingSettingType(model)
	alertGroupingSetting := pagerduty.AlertGroupingSetting{
		ID:          model.ID.ValueString(),
		Name:        model.Name.ValueString(),
		Description: model.Description.ValueString(),
		Type:        configType,
		Config:      buildPagerdutyAlertGroupingSettingConfig(ctx, model, configType, diags),
		Services:    buildPagerdutyAlertGroupingSettingServices(model),
	}
	return alertGroupingSetting
}

func buildPagerdutyAlertGroupingSettingType(model *resourceAlertGroupingSettingModel) pagerduty.AlertGroupingSettingType {
	if !model.ConfigTime.IsNull() && !model.ConfigTime.IsUnknown() {
		return pagerduty.AlertGroupingSettingTimeType
	}
	if !model.ConfigIntelligent.IsNull() && !model.ConfigIntelligent.IsUnknown() {
		return pagerduty.AlertGroupingSettingIntelligentType
	}
	if !model.ConfigCBI.IsNull() && !model.ConfigCBI.IsUnknown() {
		return pagerduty.AlertGroupingSettingContentBasedIntelligentType
	}
	if !model.ConfigContentBased.IsNull() && !model.ConfigContentBased.IsUnknown() {
		return pagerduty.AlertGroupingSettingContentBasedType
	}
	return pagerduty.AlertGroupingSettingContentBasedType // unreachable
}

func buildPagerdutyAlertGroupingSettingConfig(
	ctx context.Context,
	model *resourceAlertGroupingSettingModel,
	configType pagerduty.AlertGroupingSettingType,
	diags *diag.Diagnostics,
) interface{} {

	switch configType {
	case pagerduty.AlertGroupingSettingContentBasedType:
		var target struct {
			TimeWindow types.Int64  `tfsdk:"time_window"`
			Aggregate  types.String `tfsdk:"aggregate"`
			Fields     types.List   `tfsdk:"fields"`
		}
		diags.Append(model.ConfigContentBased.As(ctx, &target, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true})...)
		if diags.HasError() {
			log.Printf("[CG] %+v", diags)
			panic("ups")
		}
		fields := []string{}
		diags.Append(target.Fields.ElementsAs(ctx, &fields, false)...)
		return pagerduty.AlertGroupingSettingConfigContentBased{
			TimeWindow: uint(target.TimeWindow.ValueInt64()),
			Aggregate:  target.Aggregate.ValueString(),
			Fields:     fields,
		}

	case pagerduty.AlertGroupingSettingContentBasedIntelligentType:
		var target struct {
			TimeWindow types.Int64  `tfsdk:"time_window"`
			Aggregate  types.String `tfsdk:"aggregate"`
			Fields     types.List   `tfsdk:"fields"`
		}
		diags.Append(model.ConfigCBI.As(ctx, &target, basetypes.ObjectAsOptions{UnhandledNullAsEmpty: true})...)
		fields := []string{}
		diags.Append(target.Fields.ElementsAs(ctx, &fields, false)...)
		return pagerduty.AlertGroupingSettingConfigContentBased{
			TimeWindow: uint(target.TimeWindow.ValueInt64()),
			Aggregate:  target.Aggregate.ValueString(),
			Fields:     fields,
		}

	case pagerduty.AlertGroupingSettingIntelligentType:
		var target struct {
			TimeWindow types.Int64 `tfsdk:"time_window"`
		}
		diags.Append(model.ConfigIntelligent.As(ctx, &target, basetypes.ObjectAsOptions{})...)
		return pagerduty.AlertGroupingSettingConfigIntelligent{
			TimeWindow: uint(target.TimeWindow.ValueInt64()),
		}

	case pagerduty.AlertGroupingSettingTimeType:
		var target struct {
			Timeout types.Int64 `tfsdk:"timeout"`
		}
		diags.Append(model.ConfigTime.As(ctx, &target, basetypes.ObjectAsOptions{})...)
		return pagerduty.AlertGroupingSettingConfigTime{
			Timeout: uint(target.Timeout.ValueInt64()),
		}
	}

	return nil
}

func buildPagerdutyAlertGroupingSettingServices(model *resourceAlertGroupingSettingModel) []pagerduty.AlertGroupingSettingService {
	elements := model.Services.Elements()
	list := make([]pagerduty.AlertGroupingSettingService, 0, len(elements))
	for _, e := range elements {
		v, _ := e.(types.String)
		list = append(list, pagerduty.AlertGroupingSettingService{
			ID: v.ValueString(),
		})
	}
	return list
}

func flattenAlertGroupingSetting(response *pagerduty.AlertGroupingSetting) resourceAlertGroupingSettingModel {
	model := resourceAlertGroupingSettingModel{
		ID:                 types.StringValue(response.ID),
		Name:               types.StringValue(response.Name),
		Description:        types.StringValue(response.Description),
		Type:               types.StringValue(string(response.Type)),
		ConfigContentBased: flattenAlertGroupingSettingConfigContentBased(response),
		ConfigCBI:          flattenAlertGroupingSettingConfigContentBasedIntelligent(response),
		ConfigIntelligent:  flattenAlertGroupingSettingConfigIntelligent(response),
		ConfigTime:         flattenAlertGroupingSettingConfigTime(response),
		Services:           flattenAlertGroupingSettingServices(response),
	}
	return model
}

func flattenAlertGroupingSettingConfigContentBased(response *pagerduty.AlertGroupingSetting) types.Object {
	var alertGroupingSettingConfigAttrTypes = map[string]attr.Type{
		"time_window": types.Int64Type,
		"aggregate":   types.StringType,
		"fields":      types.ListType{ElemType: types.StringType},
	}

	c, ok := response.Config.(pagerduty.AlertGroupingSettingConfigContentBased)
	if !ok || response.Type != "content_based" {
		return types.ObjectNull(alertGroupingSettingConfigAttrTypes)
	}

	fields := make([]attr.Value, 0, len(c.Fields))
	for _, f := range c.Fields {
		fields = append(fields, types.StringValue(f))
	}

	timeWindow := types.Int64Value(int64(c.TimeWindow))
	// Use configuration's value if the API response is different
	// from it to prevent an inconsistency check error.

	return types.ObjectValueMust(alertGroupingSettingConfigAttrTypes, map[string]attr.Value{
		"time_window": timeWindow,
		"aggregate":   types.StringValue(c.Aggregate),
		"fields":      types.ListValueMust(types.StringType, fields),
	})
}

func flattenAlertGroupingSettingConfigContentBasedIntelligent(response *pagerduty.AlertGroupingSetting) types.Object {
	var alertGroupingSettingConfigAttrTypes = map[string]attr.Type{
		"time_window": types.Int64Type,
		"aggregate":   types.StringType,
		"fields":      types.ListType{ElemType: types.StringType},
	}
	// TODO
	return types.ObjectNull(alertGroupingSettingConfigAttrTypes)
}

func flattenAlertGroupingSettingConfigIntelligent(response *pagerduty.AlertGroupingSetting) types.Object {
	var alertGroupingSettingConfigAttrTypes = map[string]attr.Type{"time_window": types.Int64Type}
	c, ok := response.Config.(pagerduty.AlertGroupingSettingConfigIntelligent)
	if !ok || response.Type != "intelligent" {
		return types.ObjectNull(alertGroupingSettingConfigAttrTypes)
	}
	return types.ObjectValueMust(alertGroupingSettingConfigAttrTypes, map[string]attr.Value{"time_window": types.Int64Value(int64(c.TimeWindow))})
}

func flattenAlertGroupingSettingConfigTime(response *pagerduty.AlertGroupingSetting) types.Object {
	var alertGroupingSettingConfigAttrTypes = map[string]attr.Type{"timeout": types.Int64Type}
	c, ok := response.Config.(pagerduty.AlertGroupingSettingConfigTime)
	if !ok || response.Type != "intelligent" {
		return types.ObjectNull(alertGroupingSettingConfigAttrTypes)
	}
	return types.ObjectValueMust(alertGroupingSettingConfigAttrTypes, map[string]attr.Value{"timeout": types.Int64Value(int64(c.Timeout))})
}

func flattenAlertGroupingSettingServices(response *pagerduty.AlertGroupingSetting) types.List {
	serviceIDs := make([]attr.Value, 0, len(response.Services))
	for _, s := range response.Services {
		serviceIDs = append(serviceIDs, types.StringValue(s.ID))
	}
	return types.ListValueMust(types.StringType, serviceIDs)
}
