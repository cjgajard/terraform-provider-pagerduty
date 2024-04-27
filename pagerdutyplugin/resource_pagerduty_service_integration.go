package pagerduty

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/PagerDuty/terraform-provider-pagerduty/util"
	"github.com/PagerDuty/terraform-provider-pagerduty/util/enumtypes"
	"github.com/PagerDuty/terraform-provider-pagerduty/util/validate"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

type resourceServiceIntegration struct{ client *pagerduty.Client }

var (
	_ resource.ResourceWithConfigure        = (*resourceServiceIntegration)(nil)
	_ resource.ResourceWithImportState      = (*resourceServiceIntegration)(nil)
	_ resource.ResourceWithConfigValidators = (*resourceServiceIntegration)(nil)
)

func (r *resourceServiceIntegration) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pagerduty_service_integration"
}

func (r *resourceServiceIntegration) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":   schema.StringAttribute{Computed: true},
			"name": schema.StringAttribute{Optional: true},
			"service": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIfConfigured()},
			},

			"type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(
						"aws_cloudwatch_inbound_integration",
						"cloudkick_inbound_integration",
						"event_transformer_api_inbound_integration",
						"events_api_v2_inbound_integration",
						"generic_email_inbound_integration",
						"generic_events_api_inbound_integration",
						"keynote_inbound_integration",
						"nagios_inbound_integration",
						"pingdom_inbound_integration",
						"sql_monitor_inbound_integration",
					),
				},
			},

			"vendor": schema.StringAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("type")),
				},
			},

			"integration_key": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Validators: []validator.String{
					validate.DeprecatedIfPresent("Argument is deprecated. " +
						"Assignments or updates to this attribute are not " +
						"supported by Service Integrations API, it is a " +
						"read-only value. Input support will be dropped in " +
						"upcomming major release"),
				},
			},

			"integration_email":       schema.StringAttribute{Optional: true, Computed: true},
			"email_incident_creation": schema.StringAttribute{Optional: true, Computed: true},
			"email_filter_mode":       schema.StringAttribute{Optional: true, Computed: true},
			"email_parsing_fallback":  schema.StringAttribute{Optional: true, Computed: true},

			"email_parser": schema.ListAttribute{
				Optional:    true,
				ElementType: emailParserObjectType,
			},

			"email_filter": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: emailFilterObjectType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplaceIfConfigured(),
					listplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *resourceServiceIntegration) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		validate.RequireAIfBEqual(
			path.Root("integration_email"),
			path.Root("type"),
			types.StringValue("generic_email_inbound_integration"),
		),
	}
}

func (r *resourceServiceIntegration) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceServiceIntegrationModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan := buildPagerdutyIntegration(ctx, &model, &resp.Diagnostics)
	log.Printf("[INFO] Creating PagerDuty service integration %s", plan.Name)

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		response, err := r.client.CreateIntegrationWithContext(ctx, plan.Service.ID, plan)
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
			fmt.Sprintf("Error creating PagerDuty service integration %s", plan.Name),
			err.Error(),
		)
		return
	}

	model, err = requestGetServiceIntegration(ctx, r.client, plan.Service.ID, plan.ID, false)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading PagerDuty service integration %s", plan.ID),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *resourceServiceIntegration) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var id types.String
	var serviceID types.String

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("service"), &serviceID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	log.Printf("[INFO] Reading PagerDuty service integration %s", id)

	retryNotFound := true
	state, err := requestGetServiceIntegration(ctx, r.client, serviceID.ValueString(), id.ValueString(), retryNotFound)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading PagerDuty service integration %s", id),
			err.Error(),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *resourceServiceIntegration) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceServiceIntegrationModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan := buildPagerdutyIntegration(ctx, &model, &resp.Diagnostics)
	if plan.ID == "" {
		var id string
		req.State.GetAttribute(ctx, path.Root("id"), &id)
		plan.ID = id
	}
	log.Printf("[INFO] Updating PagerDuty service integration %s", plan.ID)

	serviceIntegration, err := r.client.UpdateIntegrationWithContext(ctx, plan.Service.ID, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error updating PagerDuty service integration %s", plan.ID),
			err.Error(),
		)
		return
	}
	model = flattenServiceIntegration(serviceIntegration)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *resourceServiceIntegration) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var id types.String
	var serviceID types.String

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("service"), &serviceID)...)
	if resp.Diagnostics.HasError() {
		return
	}
	log.Printf("[INFO] Deleting PagerDuty service integration %s for %s", id, serviceID)

	err := r.client.DeleteIntegrationWithContext(ctx, serviceID.ValueString(), id.ValueString())
	if err != nil && !util.IsNotFoundError(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error deleting PagerDuty service integration %s", id),
			err.Error(),
		)
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *resourceServiceIntegration) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(ConfigurePagerdutyClient(&r.client, req.ProviderData)...)
}

func (r *resourceServiceIntegration) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ids := strings.Split(req.ID, ".")
	if len(ids) != 2 {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error importing pagerduty_service_integration %v", req.ID),
			"Expecting an importation ID formed as '<service_id>.<integration_id>'",
		)
	}

	_, err := requestGetServiceIntegration(ctx, r.client, ids[0], ids[1], false)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error importing pagerduty_service_integration %v", req.ID),
			err.Error(),
		)
	}

	resp.State.SetAttribute(ctx, path.Root("id"), ids[1])
	resp.State.SetAttribute(ctx, path.Root("service"), ids[0])
}

type resourceServiceIntegrationModel struct {
	ID                    types.String `tfsdk:"id"`
	Name                  types.String `tfsdk:"name"`
	Service               types.String `tfsdk:"service"`
	Type                  types.String `tfsdk:"type"`
	Vendor                types.String `tfsdk:"vendor"`
	IntegrationKey        types.String `tfsdk:"integration_key"`
	IntegrationEmail      types.String `tfsdk:"integration_email"`
	EmailIncidentCreation types.String `tfsdk:"email_incident_creation"`
	EmailFilterMode       types.String `tfsdk:"email_filter_mode"`
	EmailParsingFallback  types.String `tfsdk:"email_parsing_fallback"`
	EmailParser           types.List   `tfsdk:"email_parser"`
	EmailFilter           types.List   `tfsdk:"email_filter"`
	HTMLURL               types.String `tfsdk:"html_url"`
}

func requestGetServiceIntegration(ctx context.Context, client *pagerduty.Client, serviceID, id string, retryNotFound bool) (resourceServiceIntegrationModel, error) {
	var model resourceServiceIntegrationModel
	opts := pagerduty.GetIntegrationOptions{}

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		serviceIntegration, err := client.GetIntegrationWithContext(ctx, serviceID, id, opts)
		if err != nil {
			if util.IsBadRequestError(err) {
				return retry.NonRetryableError(err)
			}
			if !retryNotFound && util.IsNotFoundError(err) {
				return retry.NonRetryableError(err)
			}
			return retry.RetryableError(err)
		}
		model = flattenServiceIntegration(serviceIntegration)
		return nil
	})

	return model, err
}

func buildPagerdutyIntegration(ctx context.Context, model *resourceServiceIntegrationModel, diags *diag.Diagnostics) pagerduty.Integration {
	return pagerduty.Integration{
		EmailFilters: buildEmailFilters(ctx, model.EmailFilter, diags),
		// EmailParsers: buildEmailParcers(model.EmailParser),
	}
}

// func buildEmailParcers(_ types.List, _ *diag.Diagnostics) []interface{} {
// 	if list.IsNull() || list.IsUnknown() {
// 		return nil
// 	}
// 	if err != nil {
// 		log.Printf("[ERR] Parce PagerDuty service integration email parcers fail %s", err) }
// 	}
// 	return nil
// }

func buildEmailFilters(ctx context.Context, list types.List, diags *diag.Diagnostics) []pagerduty.IntegrationEmailFilterRule {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}

	var target []struct {
		ID             types.String `tfsdk:"id"`
		SubjectMode    types.String `tfsdk:"subject_mode"`
		SubjectRegex   types.String `tfsdk:"subject_regex"`
		BodyMode       types.String `tfsdk:"body_mode"`
		BodyRegex      types.String `tfsdk:"body_regex"`
		FromEmailMode  types.String `tfsdk:"from_email_mode"`
		FromEmailRegex types.String `tfsdk:"from_email_regex"`
	}

	d := list.ElementsAs(ctx, &target, false)
	diags.Append(d...)
	if d.HasError() {
		return nil
	}

	emailFilters := make([]pagerduty.IntegrationEmailFilterRule, 0, len(target))
	for _, ef := range target {
		emailFilters = append(emailFilters, pagerduty.IntegrationEmailFilterRule{
			// ID:             ef.ID.ValueString(),
			SubjectMode:    buildPagerDutyEmailFilterRuleMode(ef.SubjectMode.ValueString()),
			SubjectRegex:   ef.SubjectRegex.ValueStringPointer(),
			BodyMode:       buildPagerDutyEmailFilterRuleMode(ef.BodyMode.ValueString()),
			BodyRegex:      ef.BodyRegex.ValueStringPointer(),
			FromEmailMode:  buildPagerDutyEmailFilterRuleMode(ef.FromEmailMode.ValueString()),
			FromEmailRegex: ef.FromEmailRegex.ValueStringPointer(),
		})
	}

	return emailFilters
}

func buildPagerDutyEmailFilterRuleMode(s string) pagerduty.IntegrationEmailFilterRuleMode {
	switch s {
	case "always":
		return pagerduty.EmailFilterRuleModeAlways
	case "match":
		return pagerduty.EmailFilterRuleModeMatch
	case "no-match":
		return pagerduty.EmailFilterRuleModeNoMatch
	default:
		return pagerduty.EmailFilterRuleModeInvalid
	}
}

func buildPagerdutyServiceIntegration(model *resourceServiceIntegrationModel) pagerduty.Integration {
	integration := pagerduty.Integration{
		Name: model.Name.ValueString(),
		Service: &pagerduty.APIObject{
			ID:   model.Service.ValueString(),
			Type: "service",
		},
	}

	integration.Type = model.Type.ValueString()

	if !model.IntegrationKey.IsNull() && !model.IntegrationKey.IsUnknown() {
		integration.IntegrationKey = model.IntegrationKey.ValueString()
	}

	if !model.IntegrationEmail.IsNull() && !model.IntegrationEmail.IsUnknown() {
		integration.IntegrationEmail = model.IntegrationEmail.ValueString()
	}

	if !model.Vendor.IsNull() && !model.Vendor.IsUnknown() {
		integration.Vendor = &pagerduty.APIObject{
			ID:   model.Vendor.ValueString(),
			Type: "vendor",
		}
	}

	// if !model.EmailIncidentCreation.IsNull() && !model.EmailIncidentCreation.IsUnknown() {
	//	integration.EmailIncidentCreation = model.EmailIncidentCreation.ValueString()
	//}

	// TODO: add EmailParsingFallback to client
	// if !model.EmailParsingFallback.IsNull() && !model.EmailParsingFallback.IsUnknown() {
	//	integration.EmailParsingFallback = model.EmailParsingFallback.ValueString()
	//}

	if !model.EmailFilterMode.IsNull() && !model.EmailFilterMode.IsUnknown() {
		switch model.EmailFilterMode.ValueString() {
		case "all-email":
			integration.EmailFilterMode = pagerduty.EmailFilterModeAll
		case "or-rules-email":
			integration.EmailFilterMode = pagerduty.EmailFilterModeOr
		case "and-rules-email":
			integration.EmailFilterMode = pagerduty.EmailFilterModeAnd
		default:
			integration.EmailFilterMode = pagerduty.EmailFilterModeInvalid
		}
	}

	return integration
}

func flattenServiceIntegration(response *pagerduty.Integration) resourceServiceIntegrationModel {
	model := resourceServiceIntegrationModel{
		ID:      types.StringValue(response.ID),
		Name:    types.StringValue(response.Name),
		Type:    types.StringValue(response.Type),
		Service: types.StringValue(response.Service.ID),
		Vendor:  types.StringValue(response.Vendor.ID),
	}

	if response.HTMLURL != "" {
		model.HTMLURL = types.StringValue(response.HTMLURL)
	}

	if response.Service != nil {
		model.Service = types.StringValue(response.Service.ID)
	}
	if response.Vendor != nil {
		model.Vendor = types.StringValue(response.Vendor.ID)
	}

	if response.IntegrationKey != "" {
		model.IntegrationKey = types.StringValue(response.IntegrationKey)
	}
	if response.IntegrationEmail != "" {
		model.IntegrationEmail = types.StringValue(response.IntegrationEmail)
	}

	// if response.EmailIncidentCreation != "" {
	// 	model.EmailIncidentCreation = types.StringValue(response.EmailIncidentCreation)
	// }

	// if response.EmailParsingFallback != "" {
	// 	model.EmailParsingFallback = types.StringValue(response.IntegrationEmail)
	// }

	if !util.IsNilFunc(response.EmailFilters) {
		model.EmailFilter = flattenEmailFilters(response.EmailFilters)
	}
	return model
}

func flattenEmailFilters(list []pagerduty.IntegrationEmailFilterRule) types.List {
	elements := []attr.Value{}
	for _, ef := range list {
		values := map[string]attr.Value{
			// "id":               types.StringValue(ef.ID),
			"id":               types.StringNull(),
			"subject_regex":    types.StringNull(),
			"body_regex":       types.StringNull(),
			"from_email_regex": types.StringNull(),
			"subject_mode":     enumtypes.NewStringValue(ef.SubjectMode.String(), emailFilterModeType),
			"body_mode":        enumtypes.NewStringValue(ef.BodyMode.String(), emailFilterModeType),
			"from_email_mode":  enumtypes.NewStringValue(ef.FromEmailMode.String(), emailFilterModeType),
		}

		if ef.SubjectRegex != nil {
			values["subject_regex"] = types.StringValue(*ef.SubjectRegex)
		}

		if ef.BodyRegex != nil {
			values["body_regex"] = types.StringValue(*ef.BodyRegex)
		}

		if ef.FromEmailRegex != nil {
			values["from_email_regex"] = types.StringValue(*ef.FromEmailRegex)
		}

		obj := types.ObjectValueMust(emailFilterObjectType.AttrTypes, values)
		elements = append(elements, obj)
	}
	return types.ListValueMust(emailFilterObjectType, elements)
}

var emailParserObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"action":          emailParserActionType, /* TODO required */
		"id":              types.StringType,
		"match_predicate": types.ListType{ElemType: emailParserMatchPredicateObjectType},
		"value_extractor": types.ListType{ElemType: emailParserValueExtractorObjectType},
	},
}

var emailParserActionType = enumtypes.StringType{OneOf: []string{"resolve", "trigger"}}

var emailParserMatchPredicateObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"type":      emailParserMatchPredicateTypeType,
		"predicate": types.ListType{ElemType: emailParserMatchPredicatePredicateObjectType},
	},
}

var emailParserMatchPredicateTypeType = enumtypes.StringType{OneOf: []string{"all", "any"} /* TODO required */}

var emailParserMatchPredicatePredicateObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"matcher":   types.StringType,
		"part":      emailParserMatchPredicatePredicatePartType,
		"predicate": types.ListType{ElemType: emailParserMatchPredicatePredicatePredicateObjectType},
		"type":      emailParserMatchPredicatePredicateTypeType, // required
	},
}

var emailParserMatchPredicatePredicatePartType = enumtypes.StringType{OneOf: []string{"body", "from_address", "subject"}}
var emailParserMatchPredicatePredicateTypeType = enumtypes.StringType{OneOf: []string{"contains", "exactly", "not", "regex"}}

var emailParserMatchPredicatePredicatePredicateObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"matcher": types.StringType,                                    // required
		"part":    emailParserMatchPredicatePredicatePartType,          // required
		"type":    emailParserMatchPredicatePredicatePredicateTypeType, // required
	},
}
var emailParserMatchPredicatePredicatePredicateTypeType = enumtypes.StringType{OneOf: []string{"contains", "exactly", "regex"}}

var emailParserValueExtractorObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"ends_before":  types.StringType,
		"part":         emailParserValueExtractorPartType, // required
		"type":         emailParserValueExtractorTypeType, // required
		"regex":        types.StringType,
		"starts_after": types.StringType,
		"value_name":   types.StringType, // required
	},
}

var emailParserValueExtractorPartType = enumtypes.StringType{OneOf: []string{"body", "from_address", "subject"}}
var emailParserValueExtractorTypeType = enumtypes.StringType{OneOf: []string{"between", "entire", "regex"}}

var emailFilterObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"id":               types.StringType,
		"subject_mode":     emailFilterModeType,
		"subject_regex":    types.StringType,
		"body_mode":        emailFilterModeType,
		"body_regex":       types.StringType,
		"from_email_mode":  emailFilterModeType,
		"from_email_regex": types.StringType,
	},
}

var emailFilterModeType = enumtypes.StringType{OneOf: []string{"always", "match", "no-match"}}

/*
func customizeServiceIntegrationDiff() schema.CustomizeDiffFunc {
	flattenEFConfigBlock := func(v interface{}) []map[string]interface{} {
		var efConfigBlock []map[string]interface{}
		if isNilFunc(v) {
			return efConfigBlock
		}
		for _, ef := range v.([]interface{}) {
			var efConfig map[string]interface{}
			if !isNilFunc(ef) {
				efConfig = ef.(map[string]interface{})
			}
			efConfigBlock = append(efConfigBlock, efConfig)
		}
		return efConfigBlock
	}

	isEFEmptyConfigBlock := func(ef map[string]interface{}) bool {
		var isEmpty bool
		if ef["body_mode"].(string) == "" &&
			ef["body_regex"].(string) == "" &&
			ef["from_email_mode"].(string) == "" &&
			ef["from_email_regex"].(string) == "" &&
			ef["subject_mode"].(string) == "" &&
			ef["subject_regex"].(string) == "" {
			isEmpty = true
		}
		return isEmpty
	}

	isEFDefaultConfigBlock := func(ef map[string]interface{}) bool {
		var isDefault bool
		if ef["body_mode"].(string) == "always" &&
			ef["body_regex"].(string) == "" &&
			ef["from_email_mode"].(string) == "always" &&
			ef["from_email_regex"].(string) == "" &&
			ef["subject_mode"].(string) == "always" &&
			ef["subject_regex"].(string) == "" {
			isDefault = true
		}
		return isDefault
	}

	return func(context context.Context, diff *schema.ResourceDiff, i interface{}) error {
		t := diff.Get("type").(string)
		if t == "generic_email_inbound_integration" && diff.Get("integration_email").(string) == "" && diff.NewValueKnown("integration_email") {
			return errors.New(errEmailIntegrationMustHaveEmail)
		}

		// All this custom diff logic is needed because the email_filters API
		// response returns a default value for its structure even when this
		// configuration is sent empty, so it produces a permanent diff on each Read
		// that has an empty configuration for email_filter attribute on HCL code.
		vOldEF, vNewEF := diff.GetChange("email_filter")
		oldEF := flattenEFConfigBlock(vOldEF)
		newEF := flattenEFConfigBlock(vNewEF)
		if len(oldEF) > 0 && len(newEF) > 0 && len(oldEF) == len(newEF) {
			var updatedEF []map[string]interface{}
			for idx, new := range newEF {
				old := oldEF[idx]
				isSameEFConfig := old["id"] == new["id"]

				efConfig := new
				if isSameEFConfig && isEFDefaultConfigBlock(old) && isEFEmptyConfigBlock(new) {
					efConfig = old
				}
				updatedEF = append(updatedEF, efConfig)
			}

			diff.SetNew("email_filter", updatedEF)
		}

		return nil
	}
}

func expandEmailParsers(v interface{}) ([]*pagerduty.EmailParser, error) {
	var emailParsers []*pagerduty.EmailParser

	for _, ep := range v.([]interface{}) {
		rep := ep.(map[string]interface{})

		repid := rep["id"].(int)
		emailParser := &pagerduty.EmailParser{
			ID:     &repid,
			Action: rep["action"].(string),
		}

		mp := rep["match_predicate"].([]interface{})[0].(map[string]interface{})

		matchPredicate := &pagerduty.MatchPredicate{
			Type: mp["type"].(string),
		}

		for _, p := range mp["predicate"].([]interface{}) {
			rp := p.(map[string]interface{})

			predicate := &pagerduty.Predicate{
				Type: rp["type"].(string),
			}
			if predicate.Type == "not" {
				mp := rp["predicate"].([]interface{})[0].(map[string]interface{})
				predicate2 := &pagerduty.Predicate{
					Type:    mp["type"].(string),
					Part:    mp["part"].(string),
					Matcher: mp["matcher"].(string),
				}
				predicate.Predicates = append(predicate.Predicates, predicate2)
			} else {
				predicate.Part = rp["part"].(string)
				predicate.Matcher = rp["matcher"].(string)
			}

			matchPredicate.Predicates = append(matchPredicate.Predicates, predicate)
		}

		emailParser.MatchPredicate = matchPredicate

		if rep["value_extractor"] != nil {
			for _, ve := range rep["value_extractor"].([]interface{}) {
				rve := ve.(map[string]interface{})

				extractor := &pagerduty.ValueExtractor{
					Type:      rve["type"].(string),
					ValueName: rve["value_name"].(string),
					Part:      rve["part"].(string),
				}

				if extractor.Type == "regex" {
					extractor.Regex = rve["regex"].(string)
				} else {
					extractor.StartsAfter = rve["starts_after"].(string)
					extractor.EndsBefore = rve["ends_before"].(string)
				}

				emailParser.ValueExtractors = append(emailParser.ValueExtractors, extractor)
			}
		}

		emailParsers = append(emailParsers, emailParser)
	}

	return emailParsers, nil
}

func flattenEmailParsers(v []*pagerduty.EmailParser) []map[string]interface{} {
	var emailParsers []map[string]interface{}

	for _, ef := range v {
		emailParser := map[string]interface{}{
			"id":     ef.ID,
			"action": ef.Action,
		}

		matchPredicate := map[string]interface{}{
			"type": ef.MatchPredicate.Type,
		}

		var predicates []map[string]interface{}

		for _, p := range ef.MatchPredicate.Predicates {
			predicate := map[string]interface{}{
				"type": p.Type,
			}

			if p.Type == "not" && len(p.Predicates) > 0 {
				var predicates2 []map[string]interface{}
				predicate2 := map[string]interface{}{
					"type":    p.Predicates[0].Type,
					"part":    p.Predicates[0].Part,
					"matcher": p.Predicates[0].Matcher,
				}

				predicates2 = append(predicates2, predicate2)

				predicate["predicate"] = predicates2

			} else {
				predicate["part"] = p.Part
				predicate["matcher"] = p.Matcher
			}

			predicates = append(predicates, predicate)
		}

		matchPredicate["predicate"] = predicates

		emailParser["match_predicate"] = []interface{}{matchPredicate}

		var valueExtractors []map[string]interface{}

		for _, ve := range ef.ValueExtractors {
			extractor := map[string]interface{}{
				"type":       ve.Type,
				"value_name": ve.ValueName,
				"part":       ve.Part,
			}

			if ve.Type == "regex" {
				extractor["regex"] = ve.Regex
			} else {
				extractor["starts_after"] = ve.StartsAfter
				extractor["ends_before"] = ve.EndsBefore
			}

			valueExtractors = append(valueExtractors, extractor)
		}

		emailParser["value_extractor"] = valueExtractors

		emailParsers = append(emailParsers, emailParser)
	}

	return emailParsers
}
*/
