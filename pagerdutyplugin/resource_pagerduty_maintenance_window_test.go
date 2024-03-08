package pagerduty

import (
	"context"
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func init() {
	resource.AddTestSweepers("pagerduty_maintenance_window", &resource.Sweeper{
		Name: "pagerduty_maintenance_window",
		F:    testSweepMaintenanceWindow,
	})
}

func testSweepMaintenanceWindow(_ string) error {
	ctx := context.Background()

	resp, err := testAccProvider.client.ListMaintenanceWindowsWithContext(ctx, pagerduty.ListMaintenanceWindowsOptions{})
	if err != nil {
		return err
	}

	for _, window := range resp.MaintenanceWindows {
		if strings.HasPrefix(window.Description, "test") || strings.HasPrefix(window.Description, "tf-") {
			log.Printf("Destroying maintenance window %s (%s)", window.Description, window.ID)
			if err := testAccProvider.client.DeleteMaintenanceWindowWithContext(ctx, window.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

func TestAccPagerDutyMaintenanceWindow_Basic(t *testing.T) {
	window := fmt.Sprintf("tf-%s", acctest.RandString(5))
	windowStartTime := testAccTimeNow().Add(24 * time.Hour).Format(time.RFC3339)
	windowEndTime := testAccTimeNow().Add(48 * time.Hour).Format(time.RFC3339)
	windowUpdated := fmt.Sprintf("tf-%s", acctest.RandString(5))
	windowUpdatedStartTime := testAccTimeNow().Add(48 * time.Hour).Format(time.RFC3339)
	windowUpdatedEndTime := testAccTimeNow().Add(72 * time.Hour).Format(time.RFC3339)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckPagerDutyMaintenanceWindowDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyMaintenanceWindowConfig(window, windowStartTime, windowEndTime),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyMaintenanceWindowExists("pagerduty_maintenance_window.foo"),
				),
			},
			{
				Config: testAccCheckPagerDutyMaintenanceWindowConfigUpdated(windowUpdated, windowUpdatedStartTime, windowUpdatedEndTime),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyMaintenanceWindowExists("pagerduty_maintenance_window.foo"),
				),
			},
		},
	})
}

func testAccCheckPagerDutyMaintenanceWindowDestroy(s *terraform.State) error {
	ctx := context.Background()

	for _, r := range s.RootModule().Resources {
		if r.Type != "pagerduty_maintenance_window" {
			continue
		}

		opts := pagerduty.GetMaintenanceWindowOptions{}
		if _, err := testAccProvider.client.GetMaintenanceWindowWithContext(ctx, r.Primary.ID, opts); err == nil {
			return fmt.Errorf("maintenance window still exists")
		}

	}

	return nil
}

func testAccCheckPagerDutyMaintenanceWindowExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ctx := context.Background()

		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No maintenance window ID is set")
		}

		opts := pagerduty.GetMaintenanceWindowOptions{}
		found, err := testAccProvider.client.GetMaintenanceWindowWithContext(ctx, rs.Primary.ID, opts)
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("maintenance window not found: %v - %v", rs.Primary.ID, found)
		}

		return nil
	}
}

func testAccCheckPagerDutyMaintenanceWindowConfig(desc, start, end string) string {
	return fmt.Sprintf(`
resource "pagerduty_user" "foo" {
  name        = "%[1]v"
  email       = "%[1]v@foo.test"
  color       = "green"
  role        = "user"
  job_title   = "foo"
  description = "foo"
}

resource "pagerduty_escalation_policy" "foo" {
  name        = "%[1]v"
  description = "bar"
  num_loops   = 2

  rule {
    escalation_delay_in_minutes = 10

    target {
      type = "user_reference"
      id   = pagerduty_user.foo.id
    }
  }
}

resource "pagerduty_service" "foo" {
  name                    = "%[1]v"
  description             = "foo"
  auto_resolve_timeout    = 1800
  acknowledgement_timeout = 1800
  escalation_policy       = pagerduty_escalation_policy.foo.id

  incident_urgency_rule {
    type    = "constant"
    urgency = "high"
  }
}

resource "pagerduty_maintenance_window" "foo" {
  description = "%[1]v"
  start_time  = "%[2]v"
  end_time    = "%[3]v"
  services    = [pagerduty_service.foo.id]
}
`, desc, start, end)
}

func testAccCheckPagerDutyMaintenanceWindowConfigUpdated(desc, start, end string) string {
	return fmt.Sprintf(`
resource "pagerduty_user" "foo" {
  name        = "%[1]v"
  email       = "%[1]v@foo.test"
  color       = "green"
  role        = "user"
  job_title   = "foo"
  description = "foo"
}

resource "pagerduty_escalation_policy" "foo" {
  name        = "%[1]v"
  description = "bar"
  num_loops   = 2

  rule {
    escalation_delay_in_minutes = 10

    target {
      type = "user_reference"
      id   = pagerduty_user.foo.id
    }
  }
}

resource "pagerduty_service" "foo" {
  name                    = "%[1]v"
  description             = "foo"
  auto_resolve_timeout    = 1800
  acknowledgement_timeout = 1800
  escalation_policy       = pagerduty_escalation_policy.foo.id

  incident_urgency_rule {
    type    = "constant"
    urgency = "high"
  }
}

resource "pagerduty_service" "foo2" {
  name                    = "%[1]v2"
  description             = "foo2"
  auto_resolve_timeout    = 1800
  acknowledgement_timeout = 1800
  escalation_policy       = pagerduty_escalation_policy.foo.id

  incident_urgency_rule {
    type    = "constant"
    urgency = "high"
  }
}

resource "pagerduty_maintenance_window" "foo" {
  description = "%[1]v"
  start_time  = "%[2]v"
  end_time    = "%[3]v"
  services    = [pagerduty_service.foo.id, pagerduty_service.foo2.id]
}
`, desc, start, end)
}
