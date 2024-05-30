package pagerduty

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func init() {
	resource.AddTestSweepers("pagerduty_incident_custom_fields", &resource.Sweeper{
		Name: "pagerduty_incident_custom_fields",
		F:    testSweepIncidentCustomField,
	})
}

func testSweepIncidentCustomField(region string) error {
	ctx := context.Background()
	resp, err := testAccProvider.client.ListCustomFieldsWithContext(ctx, pagerduty.ListCustomFieldsOptions{})
	if err != nil {
		return err
	}

	for _, customField := range resp.Fields {
		if strings.HasPrefix(customField.Name, "tf_") {
			log.Printf("Destroying field %s (%s)", customField.Name, customField.ID)
			if err := testAccProvider.client.DeleteCustomFieldWithContext(ctx, customField.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

func TestAccPagerDutyIncidentCustomFields_Basic(t *testing.T) {
	fieldName := fmt.Sprintf("tf_%s", acctest.RandString(5))
	description1 := acctest.RandString(10)
	description2 := acctest.RandString(10)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckIncidentCustomFieldTests(t)
		},
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:      testAccCheckPagerDutyIncidentCustomFieldDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyIncidentCustomFieldConfig(fieldName, description1, "string"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyIncidentCustomFieldExists("pagerduty_incident_custom_field.input"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_custom_field.input", "name", fieldName),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_custom_field.input", "description", description1),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_custom_field.input", "data_type", "string"),
				),
			},
			{
				Config: testAccCheckPagerDutyIncidentCustomFieldConfig(fieldName, description2, "string"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyIncidentCustomFieldExists("pagerduty_incident_custom_field.input"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_custom_field.input", "description", description2),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_custom_field.input", "data_type", "string"),
				),
			},
		},
	})
}

func TestAccPagerDutyIncidentCustomField_BasicWithDescription(t *testing.T) {
	fieldName := fmt.Sprintf("tf_%s", acctest.RandString(5))
	description := acctest.RandString(30)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckIncidentCustomFieldTests(t)
		},
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:      testAccCheckPagerDutyIncidentCustomFieldDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyIncidentCustomFieldConfig(fieldName, description, "string"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyIncidentCustomFieldExists("pagerduty_incident_custom_field.input"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_custom_field.input", "name", fieldName),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_custom_field.input", "data_type", "string"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_custom_field.input", "description", description),
				),
			},
		},
	})
}

func TestAccPagerDutyIncidentCustomFields_UnknownDataType(t *testing.T) {
	fieldName := fmt.Sprintf("tf_%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckIncidentCustomFieldTests(t)
		},
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:      testAccCheckPagerDutyIncidentCustomFieldDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccCheckPagerDutyIncidentCustomFieldConfig(fieldName, "", "garbage"),
				ExpectError: regexp.MustCompile("Unknown data_type garbage"),
			},
		},
	})
}

func TestAccPagerDutyIncidentCustomFields_IllegalDataType(t *testing.T) {
	fieldName := fmt.Sprintf("tf_%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckIncidentCustomFieldTests(t)
		},
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:      testAccCheckPagerDutyIncidentCustomFieldDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testAccCheckPagerDutyIncidentCustomFieldConfig(fieldName, "", "unknown"),
				ExpectError: regexp.MustCompile("Unknown data_type unknown"),
			},
		},
	})
}

func testAccCheckPagerDutyIncidentCustomFieldConfig(name, description, datatype string) string {
	return fmt.Sprintf(`
resource "pagerduty_incident_custom_field" "input" {
  name = "%[1]s"
  display_name = "%[1]s"
  description = "%[2]s" 
  data_type = "%[3]s"
  field_type = "single_value_fixed"
}
`, name, description, datatype)
}

func testAccCheckPagerDutyIncidentCustomFieldConfigNoDescription(name, datatype string) string {
	return fmt.Sprintf(`
resource "pagerduty_incident_custom_field" "input" {
  name = "%[1]s"
  display_name = "%[1]s"
  data_type = "%[2]s"
  field_type = "single_value_fixed"
}
`, name, datatype)
}

func testAccCheckPagerDutyIncidentCustomFieldDestroy(s *terraform.State) error {
	ctx := context.Background()

	for _, r := range s.RootModule().Resources {
		if r.Type != "pagerduty_incident_custom_field" {
			continue
		}

		_, err := testAccProvider.client.GetCustomFieldWithContext(ctx, r.Primary.ID, pagerduty.GetCustomFieldOptions{})
		if err == nil {
			return fmt.Errorf("field still exists")
		}
	}

	return nil
}

func testAccCheckPagerDutyIncidentCustomFieldExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("no field ID is set")
		}

		found, err := testAccProvider.client.GetCustomFieldWithContext(context.Background(), rs.Primary.ID, pagerduty.GetCustomFieldOptions{})
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("field not found: %v - %v", rs.Primary.ID, found)
		}

		return nil
	}
}

func testAccPreCheckIncidentCustomFieldTests(t *testing.T) {
	if v := os.Getenv("PAGERDUTY_ACC_INCIDENT_CUSTOM_FIELDS"); v == "" {
		t.Skip("PAGERDUTY_ACC_INCIDENT_CUSTOM_FIELDS not set. Skipping Incident Custom Field-related test")
	}
}
