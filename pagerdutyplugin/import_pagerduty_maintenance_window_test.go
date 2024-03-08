package pagerduty

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPagerDutyMaintenanceWindow_import(t *testing.T) {
	window := fmt.Sprintf("tf-%s", acctest.RandString(5))
	windowStartTime := testAccTimeNow().Add(24 * time.Hour).Format(time.RFC3339)
	windowEndTime := testAccTimeNow().Add(48 * time.Hour).Format(time.RFC3339)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckPagerDutyMaintenanceWindowDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyMaintenanceWindowConfig(window, windowStartTime, windowEndTime),
			},

			{
				ResourceName:      "pagerduty_maintenance_window.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
