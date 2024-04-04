package pagerduty

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/PagerDuty/terraform-provider-pagerduty/util"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

type resourceEventOrchestrationIntegration struct{ client *pagerduty.Client }

var (
	_ resource.ResourceWithConfigure   = (*resourceEventOrchestrationIntegration)(nil)
	_ resource.ResourceWithImportState = (*resourceEventOrchestrationIntegration)(nil)
)

func (r *resourceEventOrchestrationIntegration) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pagerduty_event_orchestration_integration"
}

func (r *resourceEventOrchestrationIntegration) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *resourceEventOrchestrationIntegration) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceEventOrchestrationIntegrationModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan := buildPagerdutyEventOrchestrationIntegration(&model)
	log.Printf("[INFO] Creating PagerDuty event orchestration integration %s", plan.Name)

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		response, err := r.client.CreateEventOrchestrationIntegrationWithContext(ctx, plan)
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
			fmt.Sprintf("Error creating PagerDuty event orchestration integration %s", plan.Name),
			err.Error(),
		)
		return
	}

	model, err = requestGetEventOrchestrationIntegration(ctx, r.client, plan.ID, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading PagerDuty event orchestration integration %s", plan.ID),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *resourceEventOrchestrationIntegration) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var id types.String

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	log.Printf("[INFO] Reading PagerDuty event orchestration integration %s", id)

	state, err := requestGetEventOrchestrationIntegration(ctx, r.client, id.ValueString(), &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading PagerDuty event orchestration integration %s", id),
			err.Error(),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *resourceEventOrchestrationIntegration) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceEventOrchestrationIntegrationModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan := buildPagerdutyEventOrchestrationIntegration(&model)
	if plan.ID == "" {
		var id string
		req.State.GetAttribute(ctx, path.Root("id"), &id)
		plan.ID = id
	}
	log.Printf("[INFO] Updating PagerDuty event orchestration integration %s", plan.ID)

	eventOrchestrationIntegration, err := r.client.UpdateEventOrchestrationIntegrationWithContext(ctx, plan.ID, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error updating PagerDuty event orchestration integration %s", plan.ID),
			err.Error(),
		)
		return
	}
	model = flattenEventOrchestrationIntegration(eventOrchestrationIntegration)

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *resourceEventOrchestrationIntegration) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var id types.String

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	log.Printf("[INFO] Deleting PagerDuty event orchestration integration %s", id)

	err := r.client.DeleteEventOrchestrationIntegrationWithContext(ctx, id.ValueString())
	if err != nil && !util.IsNotFoundError(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error deleting PagerDuty event orchestration integration %s", id),
			err.Error(),
		)
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *resourceEventOrchestrationIntegration) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(ConfigurePagerdutyClient(&r.client, req.ProviderData)...)
}

func (r *resourceEventOrchestrationIntegration) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type resourceEventOrchestrationIntegrationModel struct {
	ID types.String `tfsdk:"id"`
}

func requestGetEventOrchestrationIntegration(ctx context.Context, client *pagerduty.Client, id string, retryNotFound bool, diags *diag.Diagnostics) (resourceEventOrchestrationIntegrationModel, error) {
	var model resourceEventOrchestrationIntegrationModel

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		eventOrchestrationIntegration, err := client.GetEventOrchestrationIntegrationWithContext(ctx, id)
		if err != nil {
			if util.IsBadRequestError(err) {
				return retry.NonRetryableError(err)
			}
			if !retryNotFound && util.IsNotFoundError(err) {
				return retry.NonRetryableError(err)
			}
			return retry.RetryableError(err)
		}
		model = flattenEventOrchestrationIntegration(eventOrchestrationIntegration)
		return nil
	})

	return model, err
}

func buildPagerdutyEventOrchestrationIntegration(model *resourceEventOrchestrationIntegrationModel) *pagerduty.EventOrchestrationIntegration {
	eventOrchestrationIntegration := pagerduty.EventOrchestrationIntegration{}
	return &eventOrchestrationIntegration
}

func flattenEventOrchestrationIntegration(response *pagerduty.EventOrchestrationIntegration) resourceEventOrchestrationIntegrationModel {
	model := resourceEventOrchestrationIntegrationModel{
		ID: types.StringValue(response.ID),
	}
	return model
}

/*
func resourcePagerDutyEventOrchestrationIntegration() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePagerDutyEventOrchestrationIntegrationCreate,
		ReadContext:   resourcePagerDutyEventOrchestrationIntegrationRead,
		UpdateContext: resourcePagerDutyEventOrchestrationIntegrationUpdate,
		DeleteContext: resourcePagerDutyEventOrchestrationIntegrationDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourcePagerDutyEventOrchestrationIntegrationImport,
		},
		Schema: map[string]*schema.Schema{
			"event_orchestration": {
				Type:             schema.TypeString,
				Required:         true,
				ValidateDiagFunc: addIntegrationMigrationWarning(),
			},
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"label": {
				Type:     schema.TypeString,
				Required: true,
			},
			"parameters": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"routing_key": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func addIntegrationMigrationWarning() schema.SchemaValidateDiagFunc {
	return func(v interface{}, p cty.Path) diag.Diagnostics {
		var diags diag.Diagnostics

		diags = append(diags, diag.Diagnostic{
			Severity:      diag.Warning,
			Summary:       "Modifying the event_orchestration property of the 'pagerduty_event_orchestration_integration' resource will cause all future events sent with this integration's routing key to be evaluated against the new Event Orchestration.",
			AttributePath: p,
		})

		return diags
	}
}

func getEventOrchestrationIntegrationPayloadData(d *schema.ResourceData) (string, *pagerduty.EventOrchestrationIntegration) {
	orchestrationId := d.Get("event_orchestration").(string)

	integration := &pagerduty.EventOrchestrationIntegration{
		Label: d.Get("label").(string),
	}

	return orchestrationId, integration
}

func fetchPagerDutyEventOrchestrationIntegration(ctx context.Context, d *schema.ResourceData, meta interface{}, oid string, id string, compareLabels bool) (*pagerduty.EventOrchestrationIntegration, error) {
	client, err := meta.(*Config).Client()
	if err != nil {
		return nil, err
	}

	if integration, _, err := client.EventOrchestrationIntegrations.GetContext(ctx, oid, id); err != nil {
		return nil, err
	} else if integration != nil {
		lbl := d.Get("label").(string)
		if compareLabels && strings.Compare(integration.Label, lbl) != 0 {
			return integration, fmt.Errorf("Integration '%s' for PagerDuty Event Orchestration '%s' error - stored label '%s' doesn't match resource label '%s'.", id, oid, integration.Label, lbl)
		} else {
			d.SetId(id)
			d.Set("event_orchestration", oid)
			setEventOrchestrationIntegrationProps(d, integration)
			return integration, nil
		}
	}

	return nil, fmt.Errorf("Reading Integration '%s' for PagerDuty Event Orchestration '%s' returned `nil`.", id, oid)
}

func resourcePagerDutyEventOrchestrationIntegrationCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := meta.(*Config).Client()
	if err != nil {
		return diag.FromErr(err)
	}

	oid, payload := getEventOrchestrationIntegrationPayloadData(d)

	retryErr := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		log.Printf("[INFO] Creating Integration '%s' for PagerDuty Event Orchestration '%s'", payload.Label, oid)

		if integration, _, err := client.EventOrchestrationIntegrations.CreateContext(ctx, oid, payload); err != nil {
			if isErrCode(err, 400) {
				return retry.NonRetryableError(err)
			}
			return retry.RetryableError(err)
		} else if integration != nil {
			// Try reading an integration after creation, retry if not found:
			if _, readErr := fetchPagerDutyEventOrchestrationIntegration(ctx, d, meta, oid, integration.ID, true); readErr != nil {
				log.Printf("[WARN] Cannot locate Integration '%s' on PagerDuty Event Orchestration '%s'. Retrying creation...", integration.ID, oid)
				return retry.RetryableError(readErr)
			}
		}
		return nil
	})

	if retryErr != nil {
		time.Sleep(2 * time.Second)
		return diag.FromErr(retryErr)
	}

	return nil
}

func resourcePagerDutyEventOrchestrationIntegrationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	id := d.Id()
	oid := d.Get("event_orchestration").(string)

	retryErr := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		log.Printf("[INFO] Reading Integration '%s' for PagerDuty Event Orchestration: %s", id, oid)

		if _, err := fetchPagerDutyEventOrchestrationIntegration(ctx, d, meta, oid, id, false); err != nil {
			if isErrCode(err, http.StatusBadRequest) {
				return retry.NonRetryableError(err)
			}

			return retry.RetryableError(err)
		}

		return nil
	})

	if retryErr != nil {
		return diag.FromErr(retryErr)
	}

	return nil
}

func resourcePagerDutyEventOrchestrationIntegrationUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := meta.(*Config).Client()
	if err != nil {
		return diag.FromErr(err)
	}

	id := d.Id()

	// Migrate integration if the event_orchestration property was modified
	if d.HasChange("event_orchestration") {
		o, n := d.GetChange("event_orchestration")
		sourceOrchId := o.(string)
		destinationOrchId := n.(string)

		retryErr := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
			log.Printf("[INFO] Migrating Event Orchestration Integration '%s': source - '%s', destination - '%s'", id, sourceOrchId, destinationOrchId)

			if _, _, err := client.EventOrchestrationIntegrations.MigrateFromOrchestrationContext(ctx, destinationOrchId, sourceOrchId, id); err != nil {
				if isErrCode(err, 400) {
					return retry.NonRetryableError(err)
				}
				return retry.RetryableError(err)
			} else {
				// try reading the migrated integration from destination and source:
				_, _, readDestErr := client.EventOrchestrationIntegrations.GetContext(ctx, destinationOrchId, id)
				srcInt, _, readSrcErr := client.EventOrchestrationIntegrations.GetContext(ctx, sourceOrchId, id)

				// retry migration if the read request returned an error:
				if readDestErr != nil {
					log.Printf("[WARN] Integration '%s' cannot be found on the destination PagerDuty Event Orchestration '%s'. Retrying migration....", id, destinationOrchId)
					return retry.RetryableError(readDestErr)
				}

				// retry migration if the integration still exists on the source:
				if readSrcErr == nil && srcInt != nil {
					log.Printf("[WARN] Integration '%s' still exists on the source PagerDuty Event Orchestration '%s'. Retrying migration....", id, sourceOrchId)
					return retry.RetryableError(fmt.Errorf("Integration '%s' still exists on the source PagerDuty Event Orchestration '%s'.", id, sourceOrchId))
				}
			}
			return nil
		})

		if retryErr != nil {
			time.Sleep(2 * time.Second)
			return diag.FromErr(retryErr)
		}
	}

	// Update migrated integration if the label property was modified
	if d.HasChange("label") {
		oid, payload := getEventOrchestrationIntegrationPayloadData(d)

		retryErr := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
			log.Printf("[INFO] Updating Integration '%s' for PagerDuty Event Orchestration: %s", id, oid)

			if integration, _, err := client.EventOrchestrationIntegrations.UpdateContext(ctx, oid, id, payload); err != nil {
				if isErrCode(err, 400) {
					return retry.NonRetryableError(err)
				}
				return retry.RetryableError(err)
			} else if integration != nil {
				// Try reading an integration after updating the label, retry if the label is not updated:
				if updInt, readErr := fetchPagerDutyEventOrchestrationIntegration(ctx, d, meta, oid, id, true); readErr != nil && updInt != nil {
					log.Printf("[WARN] Label for Integration '%s' on PagerDuty Event Orchestration '%s' was not updated. Expected: '%s', actual: '%s'. Retrying update...", id, oid, payload.Label, updInt.Label)
					return retry.RetryableError(readErr)
				}
			}
			return nil
		})

		if retryErr != nil {
			time.Sleep(2 * time.Second)
			return diag.FromErr(retryErr)
		}
	}

	return nil
}

func resourcePagerDutyEventOrchestrationIntegrationDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := meta.(*Config).Client()
	if err != nil {
		return diag.FromErr(err)
	}

	id := d.Id()
	oid, _ := getEventOrchestrationIntegrationPayloadData(d)

	retryErr := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		log.Printf("[INFO] Deleting Integration '%s' for PagerDuty Event Orchestration: %s", id, oid)
		if _, err := client.EventOrchestrationIntegrations.DeleteContext(ctx, oid, id); err != nil {
			if isErrCode(err, http.StatusBadRequest) {
				return retry.NonRetryableError(err)
			}

			return retry.RetryableError(err)
		} else {
			// Try reading an integration after deletion, retry if still found:
			if integr, _, readErr := client.EventOrchestrationIntegrations.GetContext(ctx, oid, id); readErr == nil && integr != nil {
				log.Printf("[WARN] Integration '%s' still exists on PagerDuty Event Orchestration '%s'. Retrying deletion...", id, oid)
				return retry.RetryableError(fmt.Errorf("Integration '%s' still exists on PagerDuty Event Orchestration '%s'.", id, oid))
			}
		}
		return nil
	})

	if retryErr != nil {
		time.Sleep(2 * time.Second)
		return diag.FromErr(retryErr)
	}

	d.SetId("")

	return nil
}

func resourcePagerDutyEventOrchestrationIntegrationImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	oid, id, err := resourcePagerDutyParseColonCompoundID(d.Id())
	if err != nil {
		return []*schema.ResourceData{}, err
	}

	if oid == "" || id == "" {
		return []*schema.ResourceData{}, fmt.Errorf("Error importing pagerduty_event_orchestration_integration. Expected import ID format: <orchestration_id>:<integration_id>")
	}

	retryErr := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		log.Printf("[INFO] Reading Integration '%s' for PagerDuty Event Orchestration: %s", id, oid)

		if _, err := fetchPagerDutyEventOrchestrationIntegration(ctx, d, meta, oid, id, false); err != nil {
			if isErrCode(err, http.StatusBadRequest) {
				return retry.NonRetryableError(err)
			}

			return retry.RetryableError(err)
		}
		return nil
	})

	if retryErr != nil {
		return []*schema.ResourceData{}, retryErr
	}

	return []*schema.ResourceData{d}, nil
}

func flattenEventOrchestrationIntegrationParameters(p *pagerduty.EventOrchestrationIntegrationParameters) []interface{} {
	result := map[string]interface{}{
		"routing_key": p.RoutingKey,
		"type":        p.Type,
	}

	return []interface{}{result}
}

func setEventOrchestrationIntegrationProps(d *schema.ResourceData, i *pagerduty.EventOrchestrationIntegration) error {
	d.Set("label", i.Label)
	d.Set("parameters", flattenEventOrchestrationIntegrationParameters(i.Parameters))

	return nil
}
*/
