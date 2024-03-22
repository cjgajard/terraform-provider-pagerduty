package pagerduty

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/PagerDuty/terraform-provider-pagerduty/util"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

type dataSourceUserContactMethod struct{ client *pagerduty.Client }

var _ datasource.DataSourceWithConfigure = (*dataSourceUserContactMethod)(nil)

func (*dataSourceUserContactMethod) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "pagerduty_user_contact_method"
}

func (*dataSourceUserContactMethod) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":               schema.StringAttribute{Computed: true},
			"user_id":          schema.StringAttribute{Required: true},
			"address":          schema.StringAttribute{Computed: true},
			"blacklisted":      schema.BoolAttribute{Computed: true},
			"country_code":     schema.Int64Attribute{Computed: true},
			"device_type":      schema.StringAttribute{Computed: true},
			"enabled":          schema.BoolAttribute{Computed: true},
			"send_short_email": schema.BoolAttribute{Computed: true},
			"label": schema.StringAttribute{
				Required:    true,
				Description: "The name of the contact method to find in the PagerDuty API",
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "The type of the contact method",
			},
		},
	}
}

func (d *dataSourceUserContactMethod) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(ConfigurePagerdutyClient(&d.client, req.ProviderData)...)
}

func (d *dataSourceUserContactMethod) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	log.Println("[INFO] Reading PagerDuty user's contact method")

	var userID types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("user_id"), &userID)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var searchLabel types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("label"), &searchLabel)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var searchType types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("type"), &searchType)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var found *pagerduty.ContactMethod
	err := retry.RetryContext(ctx, 5*time.Minute, func() *retry.RetryError {
		response, err := d.client.ListUserContactMethodsWithContext(ctx, userID.ValueString())
		if err != nil {
			if util.IsBadRequestError(err) || util.IsNotFoundError(err) {
				return retry.NonRetryableError(err)
			}
			return retry.RetryableError(err)
		}

		for _, cm := range response.ContactMethods {
			if cm.Label == searchLabel.ValueString() || cm.Type == searchType.ValueString() {
				found = &cm
				break
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
			fmt.Sprintf("Error reading PagerDuty user contact method with label: %s", searchLabel),
			err.Error(),
		)
		return
	}

	if found == nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to locate any user contact method with label: %s", searchLabel),
			"",
		)
		return
	}

	model := dataSourceUserContactMethodModel{
		ID:             types.StringValue(found.ID),
		UserID:         userID,
		Address:        types.StringValue(found.Address),
		Blacklisted:    types.BoolValue(found.Blacklisted),
		CountryCode:    types.Int64Value(int64(found.CountryCode)),
		Enabled:        types.BoolValue(found.Enabled),
		Label:          types.StringValue(found.Label),
		SendShortEmail: types.BoolValue(found.SendShortEmail),
		Type:           types.StringValue(found.Type),
	}
	if found.Type == "push_notification_contact_method" {
		model.DeviceType = types.StringValue(found.DeviceType)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

type dataSourceUserContactMethodModel struct {
	ID             types.String `tfsdk:"id"`
	UserID         types.String `tfsdk:"user_id"`
	Address        types.String `tfsdk:"address"`
	Blacklisted    types.Bool   `tfsdk:"blacklisted"`
	CountryCode    types.Int64  `tfsdk:"country_code"`
	DeviceType     types.String `tfsdk:"device_type"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	Label          types.String `tfsdk:"label"`
	SendShortEmail types.Bool   `tfsdk:"send_short_email"`
	Type           types.String `tfsdk:"type"`
}
