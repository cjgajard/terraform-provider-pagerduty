package pagerduty

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	heimweh "github.com/heimweh/go-pagerduty/pagerduty"
)

func init() {
	resource.AddTestSweepers("pagerduty_incident_workflow_triggers", &resource.Sweeper{
		Name: "pagerduty_incident_workflow_triggers",
		F:    testSweepIncidentWorkflowTrigger,
	})
}

// testSweepIncidentWorkflowTrigger enumerates test-created workflows through
// the legacy heimweh client (the pagerduty_incident_workflow resource has not
// been migrated yet, so that is still the only client that knows how to list
// workflows by name) and deletes their triggers through the official client
// that backs the now-migrated pagerduty_incident_workflow_trigger resource.
func testSweepIncidentWorkflowTrigger(_ string) error {
	token := os.Getenv("PAGERDUTY_TOKEN")
	if token == "" {
		return fmt.Errorf("PAGERDUTY_TOKEN must be set")
	}

	legacyClient, err := heimweh.NewClient(&heimweh.Config{Token: token})
	if err != nil {
		return err
	}

	workflowsResp, _, err := legacyClient.IncidentWorkflows.List(&heimweh.ListIncidentWorkflowOptions{})
	if err != nil {
		return err
	}

	ctx := context.Background()
	client := testAccProvider.client

	for _, iw := range workflowsResp.IncidentWorkflows {
		opts := pagerduty.ListIncidentWorkflowTriggersOptions{WorkflowID: iw.ID, Limit: 100}
		for {
			page, err := client.ListIncidentWorkflowTriggers(ctx, opts)
			if err != nil {
				return err
			}
			for _, t := range page.Triggers {
				log.Printf("Destroying incident workflow trigger %s", t.ID)
				if err := client.DeleteIncidentWorkflowTrigger(ctx, t.ID); err != nil {
					return err
				}
			}
			if page.NextPageToken == "" {
				break
			}
			opts.PageToken = page.NextPageToken
		}
	}

	return nil
}

func TestAccPagerDutyIncidentWorkflowTrigger_BadType(t *testing.T) {
	config := `
resource "pagerduty_incident_workflow_trigger" "my_first_workflow_trigger" {
  type             = "dummy"
  workflow         = "ignored"
  subscribed_to_all_services = true
}
`
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckIncidentWorkflows(t)
		},
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`(?s)value must be one of.*"dummy"`),
			},
		},
	})
}

func TestAccPagerDutyIncidentWorkflowTrigger_ConditionWithManualType(t *testing.T) {
	config := `
resource "pagerduty_incident_workflow_trigger" "my_first_workflow_trigger" {
  type             = "manual"
  workflow         = "ignored"
  condition        = "something"
  subscribed_to_all_services = true
}
`
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckIncidentWorkflows(t)
		},
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("when trigger type manual is used, condition must not be specified"),
			},
		},
	})
}

func TestAccPagerDutyIncidentWorkflowTrigger_SubscribedToAllWithInvalidServices(t *testing.T) {
	config := `
resource "pagerduty_incident_workflow_trigger" "my_first_workflow_trigger" {
  type       = "conditional"
  workflow   = "ignored"
  condition  = "something"
  subscribed_to_all_services = true
  services = ["abc-123"]
}
`
	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckIncidentWorkflows(t)
		},
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("when subscribed_to_all_services is true, services must either be not defined or empty"),
			},
		},
	})
}

func TestAccPagerDutyIncidentWorkflowTrigger_IncidentTypesOnWrongType(t *testing.T) {
	workflow := fmt.Sprintf("tf-%s", acctest.RandString(5))
	config := fmt.Sprintf(`
%s

resource "pagerduty_incident_workflow_trigger" "test" {
  type                       = "conditional"
  workflow                   = pagerduty_incident_workflow.test.id
  condition                  = "incident.priority matches 'P1'"
  incident_types             = ["PLACEHOLDER"]
  subscribed_to_all_services = true
}
`, testAccCheckPagerDutyIncidentWorkflowTriggerConfigWorkflow(workflow))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckIncidentWorkflows(t)
		},
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("incident_types can only be specified when trigger type is incident_type"),
			},
		},
	})
}

func TestAccPagerDutyIncidentWorkflowTrigger_IncidentTypeMissingTypes(t *testing.T) {
	workflow := fmt.Sprintf("tf-%s", acctest.RandString(5))
	config := fmt.Sprintf(`
%s

resource "pagerduty_incident_workflow_trigger" "test" {
  type                       = "incident_type"
  workflow                   = pagerduty_incident_workflow.test.id
  subscribed_to_all_services = true
}
`, testAccCheckPagerDutyIncidentWorkflowTriggerConfigWorkflow(workflow))

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckIncidentWorkflows(t)
		},
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("incident_types must be specified when trigger type is incident_type"),
			},
		},
	})
}

func TestAccPagerDutyIncidentWorkflowTrigger_BasicManual(t *testing.T) {
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
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyIncidentWorkflowTriggerExists("pagerduty_incident_workflow_trigger.test"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "type", "manual"),
				),
			},
		},
	})
}

// testAccCheckPagerDutyIncidentWorkflowTriggerConfigWorkflow returns a
// minimal pagerduty_incident_workflow (still backed by the legacy provider,
// which is unaffected by the trigger resource's migration and remains
// available through the same muxed provider factory).
func testAccCheckPagerDutyIncidentWorkflowTriggerConfigWorkflow(name string) string {
	return fmt.Sprintf(`
resource "pagerduty_incident_workflow" "test" {
  name = "%s"
}
`, name)
}

func testAccCheckPagerDutyIncidentWorkflowTriggerConfigManualSingleService(service, workflow string) string {
	return fmt.Sprintf(`
data "pagerduty_escalation_policy" "default" {
  name = "Default"
}

resource "pagerduty_service" "test" {
  name              = "%s"
  escalation_policy = data.pagerduty_escalation_policy.default.id
}

%s

resource "pagerduty_incident_workflow_trigger" "test" {
  type       = "manual"
  workflow   = pagerduty_incident_workflow.test.id
  services   = [pagerduty_service.test.id]
  subscribed_to_all_services = false
}
`, service, testAccCheckPagerDutyIncidentWorkflowTriggerConfigWorkflow(workflow))
}

func TestAccPagerDutyIncidentWorkflowTrigger_BasicConditionalAllServices(t *testing.T) {
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
				Config: testAccCheckPagerDutyIncidentWorkflowTriggerConfigConditionalAllServices(workflow, ""),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyIncidentWorkflowTriggerExists("pagerduty_incident_workflow_trigger.test"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "type", "conditional"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "condition", ""),
					resource.TestCheckResourceAttr("pagerduty_incident_workflow_trigger.test", "subscribed_to_all_services", "true"),
				),
			},
			{
				Config: testAccCheckPagerDutyIncidentWorkflowTriggerConfigConditionalAllServices(workflow, "incident.priority matches 'P1'"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyIncidentWorkflowTriggerExists("pagerduty_incident_workflow_trigger.test"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "type", "conditional"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "condition", "incident.priority matches 'P1'"),
					resource.TestCheckResourceAttr("pagerduty_incident_workflow_trigger.test", "subscribed_to_all_services", "true"),
				),
			},
			{
				Config: testAccCheckPagerDutyIncidentWorkflowTriggerConfigConditionalAllServices(workflow, "incident.priority matches 'P2'"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyIncidentWorkflowTriggerExists("pagerduty_incident_workflow_trigger.test"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "type", "conditional"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "condition", "incident.priority matches 'P2'"),
				),
			},
		},
	})
}

func testAccCheckPagerDutyIncidentWorkflowTriggerConfigConditionalAllServices(workflow, condition string) string {
	return fmt.Sprintf(`
%s

resource "pagerduty_incident_workflow_trigger" "test" {
  type       = "conditional"
  workflow   = pagerduty_incident_workflow.test.id
  services   = []
  condition  = "%s"
  subscribed_to_all_services = true
}
`, testAccCheckPagerDutyIncidentWorkflowTriggerConfigWorkflow(workflow), condition)
}

func testAccCheckPagerDutyIncidentWorkflowTriggerConfigManualAllServices(workflow string) string {
	return fmt.Sprintf(`
%s

resource "pagerduty_incident_workflow_trigger" "test" {
  type       = "manual"
  workflow   = pagerduty_incident_workflow.test.id
  services   = []
  subscribed_to_all_services = true
}
`, testAccCheckPagerDutyIncidentWorkflowTriggerConfigWorkflow(workflow))
}

func TestAccPagerDutyIncidentWorkflowTrigger_ManualWithTeamPermissions(t *testing.T) {
	service := fmt.Sprintf("tf-%s", acctest.RandString(5))
	workflow := fmt.Sprintf("tf-%s", acctest.RandString(5))
	teamName := fmt.Sprintf("tf-%s", acctest.RandString(5))
	teamIDTFRef := "pagerduty_team.foo.id"
	emptyCondition := ""
	dummyCondition := "event.summary matches 'foo'"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckIncidentWorkflows(t)
		},
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckPagerDutyIncidentWorkflowTriggerDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyIncidentWorkflowTriggerConfigManualWithPermissions(service, teamName, workflow),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyIncidentWorkflowTriggerExists("pagerduty_incident_workflow_trigger.test"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "type", "manual"),
					// permissions is a block: when not configured it is an
					// empty list, not a computed default value, unlike the
					// legacy SDKv2 resource. See CHANGELOG for v3.36.0.
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "permissions.#", "0"),
				),
			},
			{
				Config: testAccCheckPagerDutyIncidentWorkflowTriggerConfigManualWithPermissionsUpdated(service, teamName, workflow, "manual", emptyCondition, "true", teamIDTFRef),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyIncidentWorkflowTriggerExists("pagerduty_incident_workflow_trigger.test"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "type", "manual"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "permissions.0.restricted", "true"),
					testAccCheckPagerDutyIncidentWorkflowTriggerCheckPermissionsTeamId("pagerduty_incident_workflow_trigger.test", "pagerduty_team.foo"),
				),
			},
			// Check input validation conditions for permissions configuration
			{
				Config:      testAccCheckPagerDutyIncidentWorkflowTriggerConfigManualWithPermissionsUpdated(service, teamName, workflow, "conditional", dummyCondition, "true", teamIDTFRef),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("restricted can only be true when trigger type is manual"),
			},
			{
				Config:      testAccCheckPagerDutyIncidentWorkflowTriggerConfigManualWithPermissionsUpdated(service, teamName, workflow, "manual", emptyCondition, "false", teamIDTFRef),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("team_id not allowed when restricted is false"),
			},
			{
				Config:      testAccCheckPagerDutyIncidentWorkflowTriggerConfigManualWithPermissionsUpdated(service, teamName, workflow, "manual", emptyCondition, "true", `""`),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("team_id must be specified when restricted is true"),
			},
		},
	})
}

func testAccCheckPagerDutyIncidentWorkflowTriggerConfigManualWithPermissions(service, team, workflow string) string {
	return fmt.Sprintf(`
data "pagerduty_escalation_policy" "default" {
  name = "Default"
}

resource "pagerduty_service" "test" {
  name              = "%s"
  escalation_policy = data.pagerduty_escalation_policy.default.id
}

%s

resource "pagerduty_team" "foo" {
  name = %q
}

resource "pagerduty_incident_workflow_trigger" "test" {
  type                       = "manual"
  workflow                   = pagerduty_incident_workflow.test.id
  services                   = [pagerduty_service.test.id]
  subscribed_to_all_services = false
}
`, service, testAccCheckPagerDutyIncidentWorkflowTriggerConfigWorkflow(workflow), team)
}

func testAccCheckPagerDutyIncidentWorkflowTriggerConfigManualWithPermissionsUpdated(service, team, workflow, triggerType, condition, isRestricted, teamId string) string {
	return fmt.Sprintf(`
data "pagerduty_escalation_policy" "default" {
  name = "Default"
}

resource "pagerduty_service" "test" {
  name              = "%s"
  escalation_policy = data.pagerduty_escalation_policy.default.id
}

%s

resource "pagerduty_team" "foo" {
  name = "%s"
}

resource "pagerduty_incident_workflow_trigger" "test" {
  type                       = "%s"
  condition                  = "%s"
  workflow                   = pagerduty_incident_workflow.test.id
  services                   = [pagerduty_service.test.id]
  subscribed_to_all_services = false
  permissions {
    restricted = %s
    team_id    = %s
  }
}
`, service, testAccCheckPagerDutyIncidentWorkflowTriggerConfigWorkflow(workflow), team, triggerType, condition, isRestricted, teamId)
}

func testAccCheckPagerDutyIncidentWorkflowTriggerCheckPermissionsTeamId(iwtName, teamName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rsIWT, ok := s.RootModule().Resources[iwtName]
		if !ok {
			return fmt.Errorf("not found: %s", iwtName)
		}
		if rsIWT.Primary.ID == "" {
			return fmt.Errorf("no incident workflow trigger ID is set")
		}

		rsTeam, ok := s.RootModule().Resources[teamName]
		if !ok {
			return fmt.Errorf("not found: %s", teamName)
		}
		if rsTeam.Primary.ID == "" {
			return fmt.Errorf("no team ID is set")
		}

		ctx := context.Background()
		found, err := testAccProvider.client.GetIncidentWorkflowTrigger(ctx, rsIWT.Primary.ID, pagerduty.GetIncidentWorkflowTriggerOptions{})
		if err != nil {
			return err
		}

		if found.Permissions == nil || found.Permissions.TeamID != rsTeam.Primary.ID {
			return fmt.Errorf("incident workflow trigger team restriction wanted %q, but got %+v", rsTeam.Primary.ID, found.Permissions)
		}

		return nil
	}
}

func TestAccPagerDutyIncidentWorkflowTrigger_ChangeTypeCausesReplace(t *testing.T) {
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
				Config: testAccCheckPagerDutyIncidentWorkflowTriggerConfigConditionalAllServices(workflow, ""),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyIncidentWorkflowTriggerExists("pagerduty_incident_workflow_trigger.test"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "type", "conditional"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "condition", ""),
					resource.TestCheckResourceAttr("pagerduty_incident_workflow_trigger.test", "subscribed_to_all_services", "true"),
				),
			},
			{
				Config: testAccCheckPagerDutyIncidentWorkflowTriggerConfigConditionalAllServices(workflow, "incident.priority matches 'P1'"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyIncidentWorkflowTriggerExists("pagerduty_incident_workflow_trigger.test"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "type", "conditional"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "condition", "incident.priority matches 'P1'"),
					resource.TestCheckResourceAttr("pagerduty_incident_workflow_trigger.test", "subscribed_to_all_services", "true"),
				),
			},
			{
				Config: testAccCheckPagerDutyIncidentWorkflowTriggerConfigManualAllServices(workflow),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyIncidentWorkflowTriggerExists("pagerduty_incident_workflow_trigger.test"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "type", "manual"),
					resource.TestCheckResourceAttr("pagerduty_incident_workflow_trigger.test", "subscribed_to_all_services", "true"),
				),
			},
		},
	})
}

func TestAccPagerDutyIncidentWorkflowTrigger_CannotChangeType(t *testing.T) {
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
				Config: testAccCheckPagerDutyIncidentWorkflowTriggerConfigConditionalAllServices(workflow, ""),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyIncidentWorkflowTriggerExists("pagerduty_incident_workflow_trigger.test"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "type", "conditional"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "condition", ""),
					resource.TestCheckResourceAttr("pagerduty_incident_workflow_trigger.test", "subscribed_to_all_services", "true"),
				),
			},
			{
				Config: testAccCheckPagerDutyIncidentWorkflowTriggerConfigConditionalAllServices(workflow, "incident.priority matches 'P1'"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyIncidentWorkflowTriggerExists("pagerduty_incident_workflow_trigger.test"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "type", "conditional"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "condition", "incident.priority matches 'P1'"),
					resource.TestCheckResourceAttr("pagerduty_incident_workflow_trigger.test", "subscribed_to_all_services", "true"),
				),
			},
			{
				Config: testAccCheckPagerDutyIncidentWorkflowTriggerConfigConditionalAllServices(workflow, "incident.priority matches 'P2'"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyIncidentWorkflowTriggerExists("pagerduty_incident_workflow_trigger.test"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "type", "conditional"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "condition", "incident.priority matches 'P2'"),
				),
			},
		},
	})
}

func TestAccPagerDutyIncidentWorkflowTrigger_UpdateToEmptyCondition(t *testing.T) {
	name := fmt.Sprintf("tf-%s", acctest.RandString(5))

	configFn := func(condition string) string {
		return fmt.Sprintf(`
data "pagerduty_escalation_policy" "default" {
  name = "Default"
}

resource "pagerduty_service" "test" {
  name              = "%[1]s"
  escalation_policy = data.pagerduty_escalation_policy.default.id
}

resource "pagerduty_incident_workflow" "test" {
  name = "%[1]s-incident-workflow"
}

resource "pagerduty_incident_workflow_trigger" "test" {
  type       = "conditional"
  workflow   = pagerduty_incident_workflow.test.id
  condition  = "%[2]s"
  subscribed_to_all_services = false
  services = [pagerduty_service.test.id]
}
`, name, condition)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckIncidentWorkflows(t)
		},
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckPagerDutyIncidentWorkflowTriggerDestroy,
		Steps: []resource.TestStep{
			{
				Config: configFn("incident.priority matches 'P1'"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyIncidentWorkflowTriggerExists("pagerduty_incident_workflow_trigger.test"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "type", "conditional"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "condition", "incident.priority matches 'P1'"),
				),
			},
			{
				Config: configFn(""),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyIncidentWorkflowTriggerExists("pagerduty_incident_workflow_trigger.test"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "type", "conditional"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "condition", ""),
				),
			},
		},
	})
}

// TestAccPagerDutyIncidentWorkflowTrigger_BasicIncidentType exercises the new
// incident_type trigger type and its incident_types argument, added to the
// PagerDuty API on 2025-01-24.
//
// NOTE: the exact wire shape of incident_types (a list of bare IDs vs. a list
// of {id, type} references) was assumed, not verified against a live
// account — see the plan doc. If the API rejects this configuration, only
// the SDK struct and this resource's expand/flatten need to change; this
// test's shape (a flat list of incident type IDs in HCL) should not need to.
func TestAccPagerDutyIncidentWorkflowTrigger_BasicIncidentType(t *testing.T) {
	ref := fmt.Sprintf("tf-%s", acctest.RandString(5))
	workflow := fmt.Sprintf("tf-%s", acctest.RandString(5))

	config := func(incidentTypeRefs string) string {
		return fmt.Sprintf(`
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
  incident_types             = [%[3]s]
  subscribed_to_all_services = true
}
`, ref, testAccCheckPagerDutyIncidentWorkflowTriggerConfigWorkflow(workflow), incidentTypeRefs)
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			testAccPreCheckIncidentWorkflows(t)
		},
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckPagerDutyIncidentWorkflowTriggerDestroy,
		Steps: []resource.TestStep{
			{
				Config: config("pagerduty_incident_type.test.id"),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyIncidentWorkflowTriggerExists("pagerduty_incident_workflow_trigger.test"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "type", "incident_type"),
					resource.TestCheckResourceAttr(
						"pagerduty_incident_workflow_trigger.test", "incident_types.#", "1"),
					resource.TestCheckResourceAttrSet(
						"pagerduty_incident_workflow_trigger.test", "incident_types.0"),
				),
			},
		},
	})
}

func testAccCheckPagerDutyIncidentWorkflowTriggerDestroy(s *terraform.State) error {
	ctx := context.Background()
	client := testAccProvider.client
	for _, r := range s.RootModule().Resources {
		if r.Type != "pagerduty_incident_workflow_trigger" {
			continue
		}

		if _, err := client.GetIncidentWorkflowTrigger(ctx, r.Primary.ID, pagerduty.GetIncidentWorkflowTriggerOptions{}); err == nil {
			return fmt.Errorf("incident workflow trigger still exists")
		}
	}
	return nil
}

func testAccCheckPagerDutyIncidentWorkflowTriggerExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("no incident workflow trigger ID is set")
		}

		ctx := context.Background()
		found, err := testAccProvider.client.GetIncidentWorkflowTrigger(ctx, rs.Primary.ID, pagerduty.GetIncidentWorkflowTriggerOptions{})
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("incident workflow trigger not found: %v - %v", rs.Primary.ID, found)
		}

		return nil
	}
}
