package pagerduty

import (
	"context"
	"fmt"
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccDataSourcePagerDutyVendor_Basic(t *testing.T) {
	dataSourceName := "data.pagerduty_vendor.foo"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckPagerDutyScheduleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourcePagerDutyVendorConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "id", "PZQ6AUS"),
					resource.TestCheckResourceAttr(dataSourceName, "name", "Amazon CloudWatch"),
				),
			},
		},
	})
}

func TestAccDataSourcePagerDutyVendor_ExactMatch(t *testing.T) {
	dataSourceName := "data.pagerduty_vendor.foo"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckPagerDutyScheduleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourcePagerDutyExactMatchConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "id", "PAM4FGS"),
					resource.TestCheckResourceAttr(dataSourceName, "name", "Datadog"),
				),
			},
		},
	})
}

func TestAccDataSourcePagerDutyVendor_SpecialChars(t *testing.T) {
	dataSourceName := "data.pagerduty_vendor.foo"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckPagerDutyScheduleDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourcePagerDutySpecialCharsConfig,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "id", "PRYWPH4"),
					resource.TestCheckResourceAttr(dataSourceName, "name", "Slack to PagerDuty (Legacy)"),
				),
			},
		},
	})
}

func testAccCheckPagerDutyScheduleDestroy(s *terraform.State) error {
	ctx := context.Background()
	for _, r := range s.RootModule().Resources {
		if r.Type != "pagerduty_schedule" {
			continue
		}
		o := pagerduty.GetScheduleOptions{}
		_, err := testAccProvider.client.GetScheduleWithContext(ctx, r.Primary.ID, o)
		if err == nil {
			return fmt.Errorf("Schedule still exists")
		}
	}
	return nil
}

const testAccDataSourcePagerDutyVendorConfig = `
data "pagerduty_vendor" "foo" {
  name = "cloudwatch"
}
`

const testAccDataSourcePagerDutyExactMatchConfig = `
data "pagerduty_vendor" "foo" {
  name = "datadog"
}
`

const testAccDataSourcePagerDutySpecialCharsConfig = `
data "pagerduty_vendor" "foo" {
  name = "Slack to PagerDuty (Legacy)"
}
`
