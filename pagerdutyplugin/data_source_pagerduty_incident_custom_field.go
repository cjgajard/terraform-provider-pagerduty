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

type dataSourceIncidentCustomField struct{ client *pagerduty.Client }

var _ datasource.DataSourceWithConfigure = (*dataSourceIncidentCustomField)(nil)

func (*dataSourceIncidentCustomField) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "pagerduty_incident_custom_field"
}

func (*dataSourceIncidentCustomField) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name":         schema.StringAttribute{Required: true},
			"id":           schema.StringAttribute{Computed: true},
			"display_name": schema.StringAttribute{Computed: true},
			"description":  schema.StringAttribute{Computed: true},
			"data_type":    schema.StringAttribute{Computed: true},
			"field_type":   schema.StringAttribute{Computed: true},
		},
	}
}

func (d *dataSourceIncidentCustomField) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(ConfigurePagerdutyClient(&d.client, req.ProviderData)...)
}

func (d *dataSourceIncidentCustomField) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	log.Println("[INFO] Reading PagerDuty incident custom field")

	var searchName types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("name"), &searchName)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var found *pagerduty.CustomField
	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		response, err := d.client.ListCustomFieldsWithContext(ctx, pagerduty.ListCustomFieldsOptions{})
		if err != nil {
			if util.IsBadRequestError(err) {
				return retry.NonRetryableError(err)
			}
			return retry.RetryableError(err)
		}

		for _, customField := range response.Fields {
			if customField.Name == searchName.ValueString() {
				found = &customField
				break
			}
		}
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Error reading PagerDuty incident custom field %s", searchName),
			err.Error(),
		)
		return
	}

	if found == nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Unable to locate any incident custom field with the name: %s", searchName),
			"",
		)
		return
	}

	resource := flattenIncidentCustomField(found, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	model := dataSourceIncidentCustomFieldModel{
		ID: resource.ID,
		Name: resource.Name,
		DisplayName: resource.DisplayName,
		Description: resource.Description,
		DataType: resource.DataType,
		FieldType: resource.FieldType,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}

type dataSourceIncidentCustomFieldModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	DisplayName types.String `tfsdk:"display_name"`
	Description types.String `tfsdk:"description"`
	DataType    types.String `tfsdk:"data_type"`
	FieldType   types.String `tfsdk:"field_type"`
}
