package pagerduty

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/PagerDuty/terraform-provider-pagerduty/util"
	"github.com/PagerDuty/terraform-provider-pagerduty/util/enumtypes"
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
	_ resource.ResourceWithConfigure   = (*resourceServiceIntegration)(nil)
	_ resource.ResourceWithImportState = (*resourceServiceIntegration)(nil)
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
				Optional:   true,
				Computed:   true,
				Validators: []validator.String{
					// TODO
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

var emailParserObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"action":          enumtypes.StringType{OneOf: []string{"resolve", "trigger"} /* TODO required */},
		"id":              types.StringType,
		"match_predicate": types.ListType{ElemType: emailParserMatchPredicateObjectType},
		"value_extractor": types.ListType{ElemType: emailParserValueExtractorObjectType},
	},
}

var emailParserActionType = enumtypes.StringType{OneOf: []string{"resolve", "trigger"} /* TODO required */}

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

func (r *resourceServiceIntegration) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceServiceIntegrationModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan := buildPagerdutyServiceIntegration(&model)
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

	model, err = requestGetServiceIntegration(ctx, r.client, plan.Service.ID, plan.ID, false, &resp.Diagnostics)
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
	state, err := requestGetServiceIntegration(ctx, r.client, serviceID.ValueString(), id.ValueString(), retryNotFound, &resp.Diagnostics)
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

	plan := buildPagerdutyServiceIntegration(&model)
	if plan.ID == "" {
		var id string
		req.State.GetAttribute(ctx, path.Root("id"), &id)
		plan.ID = id
	}
	log.Printf("[INFO] Updating PagerDuty service integration %s", plan.ID)

	serviceIntegration, err := r.client.UpdateIntegrationWithContext(ctx, plan.ID, plan)
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
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type resourceServiceIntegrationModel struct {
	ID types.String `tfsdk:"id"`
}

func requestGetServiceIntegration(ctx context.Context, client *pagerduty.Client, serviceID, id string, retryNotFound bool, diags *diag.Diagnostics) (resourceServiceIntegrationModel, error) {
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

func buildPagerdutyServiceIntegration(model *resourceServiceIntegrationModel) pagerduty.Integration {
	serviceIntegration := pagerduty.Integration{}
	return serviceIntegration
}

func flattenServiceIntegration(response *pagerduty.Integration) resourceServiceIntegrationModel {
	model := resourceServiceIntegrationModel{
		ID: types.StringValue(response.ID),
	}
	return model
}

/*
func resourcePagerDutyServiceIntegration() *schema.Resource {
	return &schema.Resource{
		Create:        resourcePagerDutyServiceIntegrationCreate,
		Read:          resourcePagerDutyServiceIntegrationRead,
		Update:        resourcePagerDutyServiceIntegrationUpdate,
		Delete:        resourcePagerDutyServiceIntegrationDelete,
		CustomizeDiff: customizeServiceIntegrationDiff(),
		Importer: &schema.ResourceImporter{
			State: resourcePagerDutyServiceIntegrationImport,
		},
		Schema: map[string]*schema.Schema{
			"integration_key": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ValidateDiagFunc: func(i interface{}, path cty.Path) diag.Diagnostics {
					v, ok := i.(string)
					if !ok {
						return diag.Diagnostics{
							{
								Severity:      diag.Error,
								Summary:       "Expected String",
								AttributePath: path,
							},
						}
					}

					if v != "" {
						return diag.Diagnostics{
							{
								Severity:      diag.Warning,
								Summary:       "Argument is deprecated. Assignments or updates to this attribute are not supported by Service Integrations API, it is a read-only value. Input support will be dropped in upcomming major release",
								AttributePath: path,
							},
						}
					}
					return diag.Diagnostics{}
				},
			},
		},
	}
}

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

func buildServiceIntegrationStruct(d *schema.ResourceData) (*pagerduty.Integration, error) {
	serviceIntegration := &pagerduty.Integration{
		Name: d.Get("name").(string),
		Type: "service_integration",
		Service: &pagerduty.ServiceReference{
			Type: "service",
			ID:   d.Get("service").(string),
		},
	}

	if attr, ok := d.GetOk("integration_key"); ok {
		serviceIntegration.IntegrationKey = attr.(string)
	}

	if attr, ok := d.GetOk("integration_email"); ok {
		serviceIntegration.IntegrationEmail = attr.(string)
	}

	if attr, ok := d.GetOk("type"); ok {
		serviceIntegration.Type = attr.(string)
	}

	if attr, ok := d.GetOk("vendor"); ok {
		serviceIntegration.Vendor = &pagerduty.VendorReference{
			ID:   attr.(string),
			Type: "vendor",
		}
	}
	if attr, ok := d.GetOk("email_incident_creation"); ok {
		serviceIntegration.EmailIncidentCreation = attr.(string)
	}

	if attr, ok := d.GetOk("email_filter_mode"); ok {
		serviceIntegration.EmailFilterMode = attr.(string)
	}

	if attr, ok := d.GetOk("email_parsing_fallback"); ok {
		serviceIntegration.EmailParsingFallback = attr.(string)
	}

	if attr, ok := d.GetOk("email_parser"); ok {
		parcers, err := expandEmailParsers(attr)
		if err != nil {
			log.Printf("[ERR] Parce PagerDuty service integration email parcers fail %s", err)
		}
		serviceIntegration.EmailParsers = parcers
	}

	if attr, ok := d.GetOk("email_filter"); ok {
		filters, err := expandEmailFilters(attr)
		if err != nil {
			log.Printf("[ERR] Parce PagerDuty service integration email filters fail %s", err)
		}
		serviceIntegration.EmailFilters = filters
	}

	if serviceIntegration.Type == "generic_email_inbound_integration" && serviceIntegration.IntegrationEmail == "" {
		return nil, errors.New(errEmailIntegrationMustHaveEmail)
	}

	return serviceIntegration, nil
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

func expandEmailFilters(v interface{}) ([]*pagerduty.EmailFilter, error) {
	var emailFilters []*pagerduty.EmailFilter

	for _, ef := range v.([]interface{}) {
		ref := ef.(map[string]interface{})

		emailFilter := &pagerduty.EmailFilter{
			ID:             ref["id"].(string),
			SubjectMode:    ref["subject_mode"].(string),
			SubjectRegex:   ref["subject_regex"].(string),
			BodyMode:       ref["body_mode"].(string),
			BodyRegex:      ref["body_regex"].(string),
			FromEmailMode:  ref["from_email_mode"].(string),
			FromEmailRegex: ref["from_email_regex"].(string),
		}

		emailFilters = append(emailFilters, emailFilter)
	}

	return emailFilters, nil
}

func flattenEmailFilters(v []*pagerduty.EmailFilter) []map[string]interface{} {
	var emailFilters []map[string]interface{}

	for _, ef := range v {
		emailFilter := map[string]interface{}{
			"id":               ef.ID,
			"subject_mode":     ef.SubjectMode,
			"subject_regex":    ef.SubjectRegex,
			"body_mode":        ef.BodyMode,
			"body_regex":       ef.BodyRegex,
			"from_email_mode":  ef.FromEmailMode,
			"from_email_regex": ef.FromEmailRegex,
		}

		emailFilters = append(emailFilters, emailFilter)
	}

	return emailFilters
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

func fetchPagerDutyServiceIntegration(d *schema.ResourceData, meta interface{}, errCallback func(error, *schema.ResourceData) error) error {
	client, err := meta.(*Config).Client()
	if err != nil {
		return err
	}

	service := d.Get("service").(string)

	o := &pagerduty.GetIntegrationOptions{}

	return retry.Retry(2*time.Minute, func() *retry.RetryError {
		serviceIntegration, _, err := client.Services.GetIntegration(service, d.Id(), o)
		if err != nil {
			log.Printf("[WARN] Service integration read error")
			if isErrCode(err, http.StatusBadRequest) {
				return retry.NonRetryableError(err)
			}

			errResp := errCallback(err, d)
			if errResp != nil {
				return retry.RetryableError(errResp)
			}

			return nil
		}

		if err := d.Set("name", serviceIntegration.Name); err != nil {
			return retry.RetryableError(err)
		}

		if err := d.Set("type", serviceIntegration.Type); err != nil {
			return retry.RetryableError(err)
		}

		if serviceIntegration.Service != nil {
			if err := d.Set("service", serviceIntegration.Service.ID); err != nil {
				return retry.RetryableError(err)
			}
		}

		if serviceIntegration.Vendor != nil {
			if err := d.Set("vendor", serviceIntegration.Vendor.ID); err != nil {
				return retry.RetryableError(err)
			}
		}

		if serviceIntegration.IntegrationKey != "" {
			if err := d.Set("integration_key", serviceIntegration.IntegrationKey); err != nil {
				return retry.RetryableError(err)
			}
		}

		if serviceIntegration.IntegrationEmail != "" {
			if err := d.Set("integration_email", serviceIntegration.IntegrationEmail); err != nil {
				return retry.RetryableError(err)
			}
		}

		if serviceIntegration.EmailIncidentCreation != "" {
			if err := d.Set("email_incident_creation", serviceIntegration.EmailIncidentCreation); err != nil {
				return retry.RetryableError(err)
			}
		}

		if serviceIntegration.EmailFilterMode != "" {
			if err := d.Set("email_filter_mode", serviceIntegration.EmailFilterMode); err != nil {
				return retry.RetryableError(err)
			}
		}

		if serviceIntegration.EmailParsingFallback != "" {
			if err := d.Set("email_parsing_fallback", serviceIntegration.EmailParsingFallback); err != nil {
				return retry.RetryableError(err)
			}
		}

		if serviceIntegration.HTMLURL != "" {
			if err := d.Set("html_url", serviceIntegration.HTMLURL); err != nil {
				return retry.RetryableError(err)
			}
		}

		if serviceIntegration.EmailFilters != nil {
			if err := d.Set("email_filter", flattenEmailFilters(serviceIntegration.EmailFilters)); err != nil {
				return retry.RetryableError(err)
			}
		}

		if serviceIntegration.EmailParsers != nil {
			if err := d.Set("email_parser", flattenEmailParsers(serviceIntegration.EmailParsers)); err != nil {
				return retry.RetryableError(err)
			}
		}

		return nil
	})
}

func resourcePagerDutyServiceIntegrationCreate(d *schema.ResourceData, meta interface{}) error {
	client, err := meta.(*Config).Client()
	if err != nil {
		return err
	}

	serviceIntegration, err := buildServiceIntegrationStruct(d)
	if err != nil {
		return err
	}

	log.Printf("[INFO] Creating PagerDuty service integration %s", serviceIntegration.Name)

	service := d.Get("service").(string)

	retryErr := retry.Retry(2*time.Minute, func() *retry.RetryError {
		if serviceIntegration, _, err := client.Services.CreateIntegration(service, serviceIntegration); err != nil {
			if isErrCode(err, 400) {
				return retry.RetryableError(err)
			}

			return retry.NonRetryableError(err)
		} else if serviceIntegration != nil {
			d.SetId(serviceIntegration.ID)
		}
		return nil
	})

	if retryErr != nil {
		return retryErr
	}

	return fetchPagerDutyServiceIntegration(d, meta, genError)
}

func resourcePagerDutyServiceIntegrationRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Reading PagerDuty service integration %s", d.Id())
	return fetchPagerDutyServiceIntegration(d, meta, handleNotFoundError)
}

func resourcePagerDutyServiceIntegrationUpdate(d *schema.ResourceData, meta interface{}) error {
	client, err := meta.(*Config).Client()
	if err != nil {
		return err
	}

	serviceIntegration, err := buildServiceIntegrationStruct(d)
	if err != nil {
		return err
	}

	service := d.Get("service").(string)

	log.Printf("[INFO] Updating PagerDuty service integration %s", d.Id())

	if _, _, err := client.Services.UpdateIntegration(service, d.Id(), serviceIntegration); err != nil {
		return err
	}

	return nil
}

func resourcePagerDutyServiceIntegrationDelete(d *schema.ResourceData, meta interface{}) error {
	client, err := meta.(*Config).Client()
	if err != nil {
		return err
	}

	service := d.Get("service").(string)

	log.Printf("[INFO] Removing PagerDuty service integration %s", d.Id())

	if _, err := client.Services.DeleteIntegration(service, d.Id()); err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func resourcePagerDutyServiceIntegrationImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	client, err := meta.(*Config).Client()
	if err != nil {
		return []*schema.ResourceData{}, err
	}

	ids := strings.Split(d.Id(), ".")

	if len(ids) != 2 {
		return []*schema.ResourceData{}, fmt.Errorf("Error importing pagerduty_service_integration. Expecting an importation ID formed as '<service_id>.<integration_id>'")
	}
	sid, id := ids[0], ids[1]

	_, _, err = client.Services.GetIntegration(sid, id, nil)
	if err != nil {
		return []*schema.ResourceData{}, err
	}

	// These are set because an import also calls Read behind the scenes
	d.SetId(id)
	d.Set("service", sid)

	return []*schema.ResourceData{d}, nil
}
*/
