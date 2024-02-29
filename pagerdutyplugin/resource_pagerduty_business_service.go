package pagerduty

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	helperResource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

type resourceBusinessService struct {
	client *pagerduty.Client
}

var (
	_ resource.ResourceWithConfigure   = (*resourceBusinessService)(nil)
	_ resource.ResourceWithImportState = (*resourceBusinessService)(nil)
)

func (r *resourceBusinessService) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pagerduty_business_service"
}

func (r *resourceBusinessService) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name":     schema.StringAttribute{Required: true},
			"id":       schema.StringAttribute{Computed: true},
			"html_url": schema.StringAttribute{Computed: true},
			"self":     schema.StringAttribute{Computed: true},
			"summary":  schema.StringAttribute{Computed: true},
			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("Managed by Terraform"),
			},
			"type": schema.StringAttribute{
				Optional:           true,
				Computed:           true,
				Default:            stringdefault.StaticString("business_service"),
				DeprecationMessage: "This will become a computed attribute in the next major release.",
				Validators:         []validator.String{stringvalidator.OneOf("business_service")},
			},
			"point_of_contact": schema.StringAttribute{Optional: true},
			"team":             schema.StringAttribute{Optional: true},
		},
	}
}

func (r *resourceBusinessService) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan resourceBusinessServiceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	businessServicePlan := buildPagerdutyBusinessService(ctx, &plan)
	log.Printf("[INFO] Creating PagerDuty business service %s", plan.Name)

	err := helperResource.RetryContext(ctx, 5*time.Minute, func() *helperResource.RetryError {
		bs, err := r.client.CreateBusinessServiceWithContext(ctx, businessServicePlan)
		if err != nil {
			return helperResource.NonRetryableError(err)
		} else if bs != nil {
			businessServicePlan.ID = bs.ID
		}
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error creating Business Service %s", plan.Name),
			err.Error(),
		)
		return
	}

	plan = requestGetBusinessService(ctx, r.client, businessServicePlan.ID, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *resourceBusinessService) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state resourceBusinessServiceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	log.Printf("[INFO] Reading PagerDuty business service %s", state.ID)

	state = requestGetBusinessService(ctx, r.client, state.ID.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *resourceBusinessService) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan resourceBusinessServiceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	businessServicePlan := buildPagerdutyBusinessService(ctx, &plan)
	if businessServicePlan.ID == "" {
		var id string
		req.State.GetAttribute(ctx, path.Root("id"), &id)
		businessServicePlan.ID = id
	}
	log.Printf("[INFO] Updating PagerDuty business service %s", businessServicePlan.ID)

	businessService, err := r.client.UpdateBusinessServiceWithContext(ctx, businessServicePlan)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error updating Business Service %s", businessServicePlan.ID),
			err.Error(),
		)
		return
	}
	plan = flattenBusinessService(businessService)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *resourceBusinessService) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var id types.String

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	log.Printf("[INFO] Deleting PagerDuty business service %s", id.String())

	err := r.client.DeleteBusinessServiceWithContext(ctx, id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error deleting Business Service %s", id),
			err.Error(),
		)
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *resourceBusinessService) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(ConfigurePagerdutyClient(&r.client, req.ProviderData)...)
}

func (r *resourceBusinessService) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type resourceBusinessServiceModel struct {
	ID             types.String `tfsdk:"id"`
	Description    types.String `tfsdk:"description"`
	HTMLUrl        types.String `tfsdk:"html_url"`
	Name           types.String `tfsdk:"name"`
	PointOfContact types.String `tfsdk:"point_of_contact"`
	Self           types.String `tfsdk:"self"`
	Summary        types.String `tfsdk:"summary"`
	Team           types.String `tfsdk:"team"`
	Type           types.String `tfsdk:"type"`
}

func requestGetBusinessService(ctx context.Context, client *pagerduty.Client, id string, diags *diag.Diagnostics) resourceBusinessServiceModel {
	var model resourceBusinessServiceModel

	err := helperResource.RetryContext(ctx, 5*time.Minute, func() *helperResource.RetryError {
		businessService, err := client.GetBusinessServiceWithContext(ctx, id)
		if err != nil {
			return helperResource.RetryableError(err)
		}
		model = flattenBusinessService(businessService)
		return nil
	})
	if err != nil {
		diags.AddError(
			fmt.Sprintf("Error reading Business Service %s", id),
			err.Error(),
		)
	}

	return model
}

func buildPagerdutyBusinessService(_ context.Context, model *resourceBusinessServiceModel) *pagerduty.BusinessService {
	businessService := pagerduty.BusinessService{
		ID:             model.ID.ValueString(),
		Description:    model.Description.ValueString(),
		HTMLUrl:        model.HTMLUrl.ValueString(),
		Name:           model.Name.ValueString(),
		PointOfContact: model.PointOfContact.ValueString(),
		Self:           model.Self.ValueString(),
		Summary:        model.Summary.ValueString(),
		Team:           &pagerduty.BusinessServiceTeam{ID: model.Team.ValueString()},
		Type:           model.Type.ValueString(),
	}
	return &businessService
}

func flattenBusinessService(src *pagerduty.BusinessService) resourceBusinessServiceModel {
	model := resourceBusinessServiceModel{
		ID:             types.StringValue(src.ID),
		Description:    types.StringValue(src.Description),
		HTMLUrl:        types.StringValue(src.HTMLUrl),
		Name:           types.StringValue(src.Name),
		Self:           types.StringValue(src.Self),
		Summary:        types.StringValue(src.Summary),
		Type:           types.StringValue(src.Type),
		PointOfContact: types.StringNull(),
		Team:           types.StringNull(),
	}
	if src.PointOfContact != "" {
		model.PointOfContact = types.StringValue(src.PointOfContact)
	}
	if src.Team != nil {
		model.Team = types.StringValue(src.Team.ID)
	}
	return model
}
