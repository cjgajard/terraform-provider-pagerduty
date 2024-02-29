package pagerduty

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/PagerDuty/terraform-provider-pagerduty/util"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

type resourceBusinessServiceSubscriber struct{ client *pagerduty.Client }

var (
	_ resource.ResourceWithConfigure   = (*resourceBusinessServiceSubscriber)(nil)
	_ resource.ResourceWithImportState = (*resourceBusinessServiceSubscriber)(nil)
)

func (r *resourceBusinessServiceSubscriber) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pagerduty_business_service_subscriber"
}

func (r *resourceBusinessServiceSubscriber) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"subscriber_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subscriber_type": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("team", "user"),
				},
			},
			"business_service_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *resourceBusinessServiceSubscriber) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceBusinessServiceSubscriberModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	businessServiceID := plan.BusinessServiceID.ValueString()
	log.Printf("[INFO] Creating business service subscriber for Business Service %v", businessServiceID)

	o := pagerduty.CreateBusinessServiceSubscriberOptions{
		Subscribers: []pagerduty.NotificationSubscriber{
			buildPagerdutyNotificationSubscriber(plan),
		},
	}

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		_, err := r.client.CreateBusinessServiceSubscriberWithContext(ctx, businessServiceID, o)
		if err != nil {
			if util.IsBadRequestError(err) {
				return retry.NonRetryableError(err)
			}
			return retry.RetryableError(err)
		}
		plan.ID = types.StringValue(buildBusinessServiceSubscriberID(plan))
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error creating business service subscriber for Business Service %v", businessServiceID),
			err.Error(),
		)
		return
	}

	model := requestGetBusinessServiceSubscriber(
		ctx, r.client,
		plan.BusinessServiceID.ValueString(),
		plan.SubscriberType.ValueString(),
		plan.SubscriberID.ValueString(),
		true,
		&resp.Diagnostics,
	)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *resourceBusinessServiceSubscriber) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceBusinessServiceSubscriberModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	log.Printf("[INFO] Reading PagerDuty business service %s subscriber %s type %s", state.BusinessServiceID, state.SubscriberID, state.SubscriberType)

	model := requestGetBusinessServiceSubscriber(
		ctx, r.client,
		state.BusinessServiceID.ValueString(),
		state.SubscriberType.ValueString(),
		state.SubscriberID.ValueString(),
		false,
		&resp.Diagnostics,
	)
	if model == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *resourceBusinessServiceSubscriber) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r *resourceBusinessServiceSubscriber) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var model resourceBusinessServiceSubscriberModel

	resp.Diagnostics.Append(req.State.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	businessServiceID := model.BusinessServiceID.ValueString()
	o := pagerduty.DeleteBusinessServiceSubscriberOptions{
		Subscribers: []pagerduty.NotificationSubscriber{
			buildPagerdutyNotificationSubscriber(model),
		},
	}

	log.Printf("[INFO] Deleting PagerDuty business service subscriber %s for %s", model.ID, businessServiceID)

	_, err := r.client.DeleteBusinessServiceSubscriberWithContext(ctx, businessServiceID, o)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error deleting PagerDuty business service subscriber %s for %s", model.ID, businessServiceID),
			err.Error(),
		)
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *resourceBusinessServiceSubscriber) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(ConfigurePagerdutyClient(&r.client, req.ProviderData)...)
}

func (r *resourceBusinessServiceSubscriber) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ids := strings.Split(req.ID, ".")
	if len(ids) != 3 {
		resp.Diagnostics.AddError(
			"Error importing pagerduty_business_service_subscriber. Expecting an importation ID formed as '<business_service_id>.<subscriber_type>.<subscriber_id>'",
			"Given Value: "+req.ID,
		)
	}

	model := resourceBusinessServiceSubscriberModel{
		ID:                types.StringValue(req.ID),
		BusinessServiceID: types.StringValue(ids[0]),
		SubscriberType:    types.StringValue(ids[1]),
		SubscriberID:      types.StringValue(ids[2]),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

type resourceBusinessServiceSubscriberModel struct {
	ID                types.String `tfsdk:"id"`
	BusinessServiceID types.String `tfsdk:"business_service_id"`
	SubscriberID      types.String `tfsdk:"subscriber_id"`
	SubscriberType    types.String `tfsdk:"subscriber_type"`
}

func requestGetBusinessServiceSubscriber(ctx context.Context, client *pagerduty.Client, businessServiceID, subscriberType, subscriberID string, retryNotFound bool, diags *diag.Diagnostics) *resourceBusinessServiceSubscriberModel {
	var model *resourceBusinessServiceSubscriberModel

	var found *pagerduty.NotificationSubscriber
	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		o := pagerduty.ListBusinessServiceSubscribersOptions{}
		list, err := client.ListBusinessServiceSubscribersWithContext(ctx, businessServiceID, o)
		if err != nil {
			if util.IsBadRequestError(err) {
				return retry.NonRetryableError(err)
			}
			if !retryNotFound && util.IsNotFoundError(err) {
				return retry.NonRetryableError(err)
			}
			return retry.RetryableError(err)
		}

		for _, sub := range list.Subscribers {
			if sub.SubscriberID == subscriberID && sub.SubscriberType == subscriberType {
				found = &sub
				break
			}
		}
		return nil
	})
	if err != nil {
		diags.AddError(
			fmt.Sprintf("Error reading PagerDuty business service %s subscriber %s type %s", businessServiceID, subscriberID, subscriberType),
			err.Error(),
		)
	}

	if found == nil {
		diags.AddError(
			"Unable to find resource",
			fmt.Sprintf("Reading PagerDuty business service %s subscriber %s type %s", businessServiceID, subscriberID, subscriberType),
		)
		return nil
	}

	model = flattenBusinessServiceSubscriber(businessServiceID, found)
	return model
}

func buildPagerdutyNotificationSubscriber(model resourceBusinessServiceSubscriberModel) pagerduty.NotificationSubscriber {
	return pagerduty.NotificationSubscriber{
		SubscriberID:   model.SubscriberID.ValueString(),
		SubscriberType: model.SubscriberType.ValueString(),
	}
}

func flattenBusinessServiceSubscriber(businessServiceID string, src *pagerduty.NotificationSubscriber) *resourceBusinessServiceSubscriberModel {
	model := resourceBusinessServiceSubscriberModel{
		ID:                types.StringValue(fmt.Sprintf("%v.%v.%v", businessServiceID, src.SubscriberType, src.SubscriberID)),
		BusinessServiceID: types.StringValue(businessServiceID),
		SubscriberID:      types.StringValue(src.SubscriberID),
		SubscriberType:    types.StringValue(src.SubscriberType),
	}
	return &model
}

func buildBusinessServiceSubscriberID(model resourceBusinessServiceSubscriberModel) string {
	businessServiceID := model.BusinessServiceID.ValueString()
	subscriberID := model.SubscriberID.ValueString()
	subscriberType := model.SubscriberType.ValueString()
	return fmt.Sprintf("%v.%v.%v", businessServiceID, subscriberType, subscriberID)
}
