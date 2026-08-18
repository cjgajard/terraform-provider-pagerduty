package pagerduty

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPagerDutyIncidentWorkflowTrigger_import(t *testing.T) {
	service := fmt.Sprintf("tf-%s", acctest.RandString(5))
	workflow := fmt.Sprintf("tf-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckIncidentWorkflows(t)
		},
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckPagerDutyIncidentWorkflowTriggerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyIncidentWorkflowTriggerConfigManualSingleService(service, workflow),
			},
			{
				ResourceName:      "pagerduty_incident_workflow_trigger.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPagerDutyIncidentWorkflowTrigger_importIncidentType(t *testing.T) {
	ref := fmt.Sprintf("tf-%s", acctest.RandString(5))
	workflow := fmt.Sprintf("tf-%s", acctest.RandString(5))

	config := fmt.Sprintf(`
resource "pagerduty_incident_type" "test" {
  name         = "%[1]s_type"
  display_name = "%[1]s Type"
  parent_type  = "incident_default"
  enabled      = true
}

%[2]s

resource "pagerduty_incident_workflow_trigger" "test" {
  type                       = "incident_type"
  workflow                   = pagerduty_incident_workflow.test.id
  incident_types             = [pagerduty_incident_type.test.id]
  subscribed_to_all_services = true
}
`, ref, testAccCheckPagerDutyIncidentWorkflowTriggerConfigWorkflow(workflow))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckIncidentWorkflows(t)
		},
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckPagerDutyIncidentWorkflowTriggerDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
			},
			{
				ResourceName:      "pagerduty_incident_workflow_trigger.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
