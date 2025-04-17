package pagerduty

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDataSourcePagerDutyIncidentWorkflowAction_Basic(t *testing.T) {
	name := "Send Status Update"
	version := 4

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourcePagerDutyIncidentWorkflowActionConfig(name, version),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "name", name),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "version", fmt.Sprintf("%d", version)),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "id", "pagerduty.com:incident-workflows:send-status-update:4"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "self", "https://api.pagerduty.com/incident_workflows/actions/pagerduty.com:incident-workflows:send-status-update:4"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "domain_name", "pagerduty.com"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "package_name", "incident-workflows"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "function_name", "send-status-update"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "description", "Posts a Status Update to the Internal Status Page and notifies subscribers"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "action_type", "action"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "tags.#", "3"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "tags.0", "incident"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "tags.1", "status"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "tags.2", "stunt"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "search_keywords.#", "3"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "search_keywords.0", "incident"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "search_keywords.1", "status"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "search_keywords.2", "stunt"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "metadata", "{\"entitlementUrl\":\"https://www.pagerduty.com/pricing/incident-management/\",\"entitlementDescription\":\"This functionality is available to customers on the Business and Enterprise plans for Incident Management.\",\"entitlementDisplayName\":\"The Business and Enterprise plans for Incident Management\",\"entitlementCallToActionText\":\"Learn More\"}"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "created_at", "2023-05-04T15:03:42.1+00:00"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "created_by_user_id", "PNH9XD4"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "inputs.#", "5"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "outputs.#", "3"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "outputs.0.name", "Result"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "outputs.0.description", "Value that shows if the action was successful or not. Either \"Success\" or \"Failed\""),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "outputs.0.type", "text"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "outputs.1.name", "Result Summary"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "outputs.1.description", "Brief description of what the action did or if it failed"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "outputs.1.type", "text"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "outputs.2.name", "Error"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "outputs.2.description", "Brief description that is populated if the action failed"),
					resource.TestCheckResourceAttr("data.pagerduty_incident_workflow_action.by_name", "outputs.2.type", "text"),
				),
			},
		},
	})
}

func testAccDataSourcePagerDutyIncidentWorkflowActionConfig(name string, version int) string {
	return fmt.Sprintf(`
data "pagerduty_incident_workflow_action" "by_name" {
    name = "%s"
    version = %d
}
`, name, version)
}
