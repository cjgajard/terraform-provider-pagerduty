package pagerduty

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/PagerDuty/terraform-provider-pagerduty/util"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

type dataSourceService struct{ client *pagerduty.Client }

var _ datasource.DataSourceWithConfigure = (*dataSourceService)(nil)

func (d *dataSourceService) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "pagerduty_service"
}

func (d *dataSourceService) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":                      schema.StringAttribute{Computed: true},
			"name":                    schema.StringAttribute{Required: true},
			"auto_resolve_timeout":    schema.Int64Attribute{Computed: true},
			"acknowledgement_timeout": schema.Int64Attribute{Computed: true},
			"alert_creation":          schema.StringAttribute{Computed: true},
			"description":             schema.StringAttribute{Computed: true},
			"escalation_policy":       schema.StringAttribute{Computed: true},
			"type":                    schema.StringAttribute{Computed: true},
			"teams": schema.ListAttribute{
				Computed:    true,
				Description: "The set of teams associated with the service",
				ElementType: types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"id":   types.StringType,
						"name": types.StringType,
					},
				},
			},
		},
	}
}

func (d *dataSourceService) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(ConfigurePagerdutyClient(&d.client, req.ProviderData)...)
}

func (d *dataSourceService) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	log.Printf("[INFO] Reading PagerDuty service")

	var searchName types.String
	if d := req.Config.GetAttribute(ctx, path.Root("name"), &searchName); d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	}

	var found *pagerduty.Service
	var offset uint = 0
	more := true

	for more {
		err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
			resp, err := d.client.ListServicesWithContext(ctx, pagerduty.ListServiceOptions{
				Query:  searchName.ValueString(),
				Limit:  10,
				Offset: offset,
			})
			if err != nil {
				if util.IsBadRequestError(err) {
					return retry.NonRetryableError(err)
				}
				return retry.RetryableError(err)
			}

			more = resp.More
			offset += uint(len(resp.Services))

			for _, service := range resp.Services {
				if service.Name == searchName.ValueString() {
					found = &service
					more = false
					break
				}
			}

			return nil
		})
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Error searching Service %s", searchName),
				err.Error(),
			)
			return
		}
	}

	if found == nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to locate any service with the name: %s", searchName),
			"",
		)
		return
	}
	model := flattenServiceData(ctx, found, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

type dataSourceServiceModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	AutoResolveTimeout     types.Int64  `tfsdk:"auto_resolve_timeout"`
	AcknowledgementTimeout types.Int64  `tfsdk:"acknowledgement_timeout"`
	AlertCreation          types.String `tfsdk:"alert_creation"`
	Description            types.String `tfsdk:"description"`
	EscalationPolicy       types.String `tfsdk:"escalation_policy"`
	Type                   types.String `tfsdk:"type"`
	Teams                  types.List   `tfsdk:"teams"`
}

func flattenServiceData(ctx context.Context, service *pagerduty.Service, diags *diag.Diagnostics) dataSourceServiceModel {
	teamObjectType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"id":   types.StringType,
			"name": types.StringType,
		},
	}

	teams, d := types.ListValueFrom(ctx, teamObjectType, service.Teams)
	diags.Append(d...)
	if d.HasError() {
		return dataSourceServiceModel{}
	}

	model := dataSourceServiceModel{
		ID:                     types.StringValue(service.ID),
		Name:                   types.StringValue(service.Name),
		Type:                   types.StringValue(service.Type),
		AutoResolveTimeout:     types.Int64Null(),
		AcknowledgementTimeout: types.Int64Null(),
		AlertCreation:          types.StringValue(service.AlertCreation),
		Description:            types.StringValue(service.Description),
		EscalationPolicy:       types.StringValue(service.EscalationPolicy.ID),
		Teams:                  teams,
	}

	if service.AutoResolveTimeout != nil {
		model.AutoResolveTimeout = types.Int64Value(int64(*service.AutoResolveTimeout))
	}
	if service.AcknowledgementTimeout != nil {
		model.AcknowledgementTimeout = types.Int64Value(int64(*service.AcknowledgementTimeout))
	}
	return model
}