package pagerduty

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/PagerDuty/terraform-provider-pagerduty/util"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccPagerDutyIncidentCustomFieldOption_Basic(t *testing.T) {
	fieldName := fmt.Sprintf("tf_%s", acctest.RandString(5))
	fieldOptionValue := fmt.Sprintf("tf_%s", acctest.RandString(5))
	fieldOptionValueUpdated := fmt.Sprintf("tf_%s", acctest.RandString(5))
	dataType := "string"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckPagerDutyIncidentCustomFieldOptionDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyIncidentCustomFieldOptionConfig(fieldName, dataType, fieldOptionValue),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyIncidentCustomFieldOptionExists("pagerduty_incident_custom_field_option.test"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_custom_field_option.test", "data_type", dataType),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_custom_field_option.test", "value", fieldOptionValue),
				),
			},
			{
				Config: testAccCheckPagerDutyIncidentCustomFieldOptionConfig(fieldName, dataType, fieldOptionValueUpdated),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyIncidentCustomFieldOptionExists("pagerduty_incident_custom_field_option.test"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_custom_field_option.test", "data_type", dataType),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_custom_field_option.test", "value", fieldOptionValueUpdated),
				),
			},
		},
	})
}

func TestAccPagerDutyIncidentCustomFieldOption_InvalidDataType(t *testing.T) {
	fieldName := fmt.Sprintf("tf_%s", acctest.RandString(5))
	fieldOptionValue := fmt.Sprintf("tf_%s", acctest.RandString(5))
	dataType := "integer"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyIncidentCustomFieldOptionConfig(fieldName, dataType, fieldOptionValue),
				ExpectError: regexp.MustCompile(
					`Attribute data_type value must be one of: \["string"\], got: "integer"`,
				),
			},
		},
	})
}

func testAccCheckPagerDutyIncidentCustomFieldOptionConfig(fieldName string, dataType string, fieldOptionValue string) string {
	fieldConfig := testAccCheckPagerDutyIncidentCustomFieldConfigNoDescription(fieldName, "string")
	return fieldConfig + "\n" + fmt.Sprintf(`
resource "pagerduty_incident_custom_field_option" "test" {
  field = pagerduty_incident_custom_field.input.id
  data_type = "%s"
  value = "%s"
}`, dataType, fieldOptionValue)
}

func testAccCheckPagerDutyIncidentCustomFieldOptionDestroy(s *terraform.State) error {
	for _, r := range s.RootModule().Resources {
		if r.Type != "pagerduty_incident_custom_field_option" {
			continue
		}

		found := false
		fieldID := r.Primary.Attributes["field"]
		fieldOptionID := r.Primary.ID

		list, err := testAccProvider.client.ListCustomFieldOptionsWithContext(context.Background(), fieldID)
		if err != nil {
			if util.IsNotFoundError(err) {
				return nil
			}
			return err
		}

		for _, o := range list.FieldOptions {
			if o.ID == fieldOptionID {
				found = true
				break
			}
		}

		if found {
			return fmt.Errorf("Field Option still exists")
		}
	}
	return nil
}

func testAccCheckPagerDutyIncidentCustomFieldOptionExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("no field option ID is set")
		}

		found := false
		fieldID := rs.Primary.Attributes["field"]
		fieldOptionID := rs.Primary.ID

		list, err := testAccProvider.client.ListCustomFieldOptionsWithContext(context.Background(), fieldID)
		if err != nil {
			return err
		}

		for _, o := range list.FieldOptions {
			if o.ID == fieldOptionID {
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("field option not found: %v/%v", fieldID, rs.Primary.ID)
		}

		return nil
	}
}
