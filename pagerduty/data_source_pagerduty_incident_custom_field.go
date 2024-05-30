package pagerduty

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/heimweh/go-pagerduty/pagerduty"
)

func dataSourcePagerDutyIncidentCustomField() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourcePagerDutyIncidentCustomFieldRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"display_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"data_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"field_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourcePagerDutyIncidentCustomFieldRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, err := meta.(*Config).Client()
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[INFO] Reading PagerDuty data source")

	searchName := d.Get("name").(string)

	err = retry.RetryContext(ctx, 5*time.Minute, func() *retry.RetryError {
		resp, _, err := client.IncidentCustomFields.ListContext(ctx, nil)
		if err != nil {
			if isErrCode(err, http.StatusBadRequest) {
				return retry.NonRetryableError(err)
			}

			// Delaying retry by 30s as recommended by PagerDuty
			// https://developer.pagerduty.com/docs/rest-api-v2/rate-limiting/#what-are-possible-workarounds-to-the-events-api-rate-limit
			time.Sleep(30 * time.Second)
			return retry.RetryableError(err)
		}

		var found *pagerduty.IncidentCustomField

		for _, field := range resp.Fields {
			if field.Name == searchName {
				found = field
				break
			}
		}

		if found == nil {
			return retry.NonRetryableError(
				fmt.Errorf("unable to locate any field with name: %s", searchName),
			)
		}

		err = flattenIncidentCustomField(d, found)
		if err != nil {
			return retry.NonRetryableError(err)
		}

		return nil
	})

	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func flattenIncidentCustomField(d *schema.ResourceData, field *pagerduty.IncidentCustomField) error {
	d.SetId(field.ID)
	d.Set("name", field.Name)
	if field.Description != nil {
		d.Set("description", *(field.Description))
	}
	d.Set("display_name", field.DisplayName)
	d.Set("data_type", field.DataType.String())
	d.Set("field_type", field.FieldType.String())

	if field.DefaultValue != nil {
		v, err := convertIncidentCustomFieldValueForFlatten(field.DefaultValue, field.FieldType.IsMultiValue())
		if err != nil {
			return err
		}
		d.Set("default_value", v)
	}
	return nil
}
