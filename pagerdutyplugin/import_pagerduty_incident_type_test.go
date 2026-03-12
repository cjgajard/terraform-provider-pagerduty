package pagerduty

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPagerDutyIncidentType_import(t *testing.T) {
	name := fmt.Sprintf("tf_%s", acctest.RandString(5))
	displayName := fmt.Sprintf("Terraform Test Incident Type %s", acctest.RandString(5))
	parentType := "incident_default"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckPagerDutyIncidentTypeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyIncidentTypeConfig(name, displayName, parentType),
			},
			{
				ResourceName:      "pagerduty_incident_type.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
