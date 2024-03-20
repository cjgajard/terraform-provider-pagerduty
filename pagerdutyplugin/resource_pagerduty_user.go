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
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

type resourceUser struct{ client *pagerduty.Client }

var (
	_ resource.ResourceWithConfigure   = (*resourceUser)(nil)
	_ resource.ResourceWithImportState = (*resourceUser)(nil)
)

func (r *resourceUser) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pagerduty_user"
}

func (r *resourceUser) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
				// TODO DiffSuppressFunc: suppressLeadTrailSpaceDiff,
			},
			"email": schema.StringAttribute{
				Required: true,
				// TODO DiffSuppressFunc: suppressCaseDiff,
			},
			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("Managed by Terraform"),
			},
			"role": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("user"),
				Validators: []validator.String{
					stringvalidator.OneOf(
						"admin",
						"limited_user",
						"observer",
						"owner",
						"read_only_user",
						"restricted_access",
						"read_only_limited_user",
						"user",
					),
				},
			},
			"teams": schema.SetAttribute{
				Optional:           true,
				Computed:           true,
				DeprecationMessage: "Use the 'pagerduty_team_membership' resource instead.",
				ElementType:        types.StringType,
				// TODO Set: schema.HashString,
			},
			"time_zone": schema.StringAttribute{
				Optional:   true,
				Computed:   true,
				Validators: []validator.String{util.ValidateTimezone()},
			},
			"avatar_url":      schema.StringAttribute{Computed: true},
			"color":           schema.StringAttribute{Optional: true, Computed: true},
			"html_url":        schema.StringAttribute{Computed: true},
			"id":              schema.StringAttribute{Computed: true},
			"invitation_sent": schema.BoolAttribute{Computed: true},
			"job_title":       schema.StringAttribute{Optional: true},
			"license":         schema.StringAttribute{Optional: true, Computed: true},
		},
	}
}

func (r *resourceUser) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var model resourceUserModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan := buildPagerdutyUser(ctx, &model)
	log.Printf("[INFO] Creating PagerDuty user %s", plan.Name)

	response, err := r.client.CreateUserWithContext(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error creating PagerDuty user %s", plan.Name),
			err.Error(),
		)
		return
	}

	model, err = requestGetUser(ctx, r.client, response.ID, util.RetryNotFound, &resp.Diagnostics)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading PagerDuty user %s", plan.ID),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *resourceUser) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var id types.String

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	log.Printf("[INFO] Reading PagerDuty user %s", id)

	state, err := requestGetUser(ctx, r.client, id.ValueString(), util.NonRetryNotFound, &resp.Diagnostics)
	if err != nil {
		if util.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading PagerDuty user %s", id),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *resourceUser) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var model resourceUserModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan := buildPagerdutyUser(ctx, &model)
	if plan.ID == "" {
		var id string
		req.State.GetAttribute(ctx, path.Root("id"), &id)
		plan.ID = id
	}
	log.Printf("[INFO] Updating PagerDuty user %s", plan.ID)

	var stateLicense types.String
	req.State.GetAttribute(ctx, path.Root("license"), &stateLicense)
	if plan.License != nil && plan.License.ID == stateLicense.ValueString() {
		plan.License = nil
	}

	// TODO delta teams
	// var teams types.Set
	// req.State.GetAttribute(ctx, path.Root("teams"), &teams)

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		user, err := r.client.UpdateUserWithContext(ctx, plan)
		if err != nil {
			return util.NewRetryError(err, util.RetryNotFound)
		}
		model = flattenUser(user)
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error updating PagerDuty user %s", plan.ID),
			err.Error(),
		)
		return
	}

	model, err = requestGetUser(ctx, r.client, plan.ID, util.NonRetryNotFound, &resp.Diagnostics)
	if err != nil {
		if util.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading PagerDuty user %s", plan.ID),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

func (r *resourceUser) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var id types.String

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	log.Printf("[INFO] Deleting PagerDuty user %s", id)

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		err := r.client.DeleteUserWithContext(ctx, id.ValueString())
		if err != nil {
			if util.IsBadRequestError(err) || util.IsNotFoundError(err) {
				return retry.NonRetryableError(err)
			}
			return retry.RetryableError(err)
		}
		return nil
	})
	if err != nil && !util.IsNotFoundError(err) {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error deleting PagerDuty user %s", id),
			err.Error(),
		)
		return
	}
	resp.State.RemoveResource(ctx)
}

func (r *resourceUser) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(ConfigurePagerdutyClient(&r.client, req.ProviderData)...)
}

func (r *resourceUser) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

type resourceUserModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Email          types.String `tfsdk:"email"`
	AvatarURL      types.String `tfsdk:"avatar_url"`
	Color          types.String `tfsdk:"color"`
	Description    types.String `tfsdk:"description"`
	HTMLURL        types.String `tfsdk:"html_url"`
	InvitationSent types.Bool   `tfsdk:"invitation_sent"`
	JobTitle       types.String `tfsdk:"job_title"`
	License        types.String `tfsdk:"license"`
	Role           types.String `tfsdk:"role"`
	Timezone       types.String `tfsdk:"time_zone"`
	Teams          types.Set    `tfsdk:"teams"`
}

func requestGetUser(ctx context.Context, client *pagerduty.Client, id string, retryNotFound util.RetryNotFoundType, diags *diag.Diagnostics) (resourceUserModel, error) {
	var model resourceUserModel
	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		user, err := client.GetUserWithContext(ctx, id, pagerduty.GetUserOptions{})
		if err != nil {
			return util.NewRetryError(err, retryNotFound)
		}
		model = flattenUser(user)
		return nil
	})
	return model, err
}

func buildPagerdutyUser(ctx context.Context, model *resourceUserModel) pagerduty.User {
	user := pagerduty.User{
		Name:     strings.TrimSpace(model.Name.ValueString()),
		Email:    model.Email.ValueString(),
		Color:    model.Color.ValueString(),
		Timezone: model.Timezone.ValueString(),
	}
	if !model.License.IsNull() && !model.License.IsUnknown() {
		user.License = &pagerduty.APIObject{
			Type: "license_reference",
			ID:   model.License.ValueString(),
		}
	}
	return user
}

func flattenUser(response *pagerduty.User) resourceUserModel {
	model := resourceUserModel{
		ID:             types.StringValue(response.ID),
		Name:           types.StringValue(response.Name),
		Email:          types.StringValue(response.Email),
		Timezone:       types.StringValue(response.Timezone),
		HTMLURL:        types.StringValue(response.HTMLURL),
		Color:          types.StringValue(response.Color),
		Role:           types.StringValue(response.Role),
		AvatarURL:      types.StringValue(response.AvatarURL),
		Description:    types.StringValue(response.Description),
		JobTitle:       types.StringValue(response.JobTitle),
		Teams:          flattenTeams(response.Teams),
		InvitationSent: types.BoolValue(response.InvitationSent),
	}
	if response.License != nil {
		model.License = types.StringValue(response.License.ID)
	}
	return model
}

func flattenTeams(teams []pagerduty.Team) types.Set {
	elements := make([]attr.Value, 0, len(teams))
	for _, t := range teams {
		elements = append(elements, types.StringValue(t.ID))
	}
	return types.SetValueMust(types.StringType, elements)
}

/*
func resourcePagerDutyUserUpdate(d *schema.ResourceData, meta interface{}) error {
	client, err := meta.(*Config).Client()
	if err != nil {
		return err
	}

	user := buildUserStruct(d)

	if ok := d.HasChangeExcept("license"); ok {
		// When not explicitely assigning a new license it's better to the backend
		// logic assign the license's id.
		user.License = nil
	}

	log.Printf("[INFO] Updating PagerDuty user %s", d.Id())

	// Retrying to give other resources (such as escalation policies) to delete
	retryErr := retry.Retry(2*time.Minute, func() *retry.RetryError {
		if _, _, err := client.Users.Update(d.Id(), user); err != nil {
			if isErrCode(err, 400) {
				return retry.RetryableError(err)
			}

			return retry.NonRetryableError(err)
		}
		return nil
	})
	if retryErr != nil {
		time.Sleep(2 * time.Second)
		return retryErr
	}

	if d.HasChange("teams") {
		o, n := d.GetChange("teams")

		if o == nil {
			o = new(schema.Set)
		}

		if n == nil {
			n = new(schema.Set)
		}

		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		remove := expandStringList(os.Difference(ns).List())
		add := expandStringList(ns.Difference(os).List())

		for _, t := range remove {

			if _, _, err := client.Teams.Get(t); err != nil {
				log.Printf("[INFO] PagerDuty team: %s not found, removing dangling team reference for user %s", t, d.Id())
				continue
			}

			log.Printf("[INFO] Removing PagerDuty user %s from team: %s", d.Id(), t)

			if _, err := client.Teams.RemoveUser(t, d.Id()); err != nil {
				return err
			}
		}

		for _, t := range add {
			log.Printf("[INFO] Adding PagerDuty user %s to team: %s", d.Id(), t)

			if _, err := client.Teams.AddUser(t, d.Id()); err != nil {
				return err
			}
		}
	}

	return resourcePagerDutyUserRead(d, meta)
}
*/
