package pagerduty

import (
	"context"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func init() {
	resource.AddTestSweepers("pagerduty_event_orchestration", &resource.Sweeper{
		Name: "pagerduty_event_orchestration",
		F:    testSweepEventOrchestration,
		Dependencies: []string{
			"pagerduty_schedule",
			"pagerduty_team",
			"pagerduty_user",
			"pagerduty_escalation_policy",
			"pagerduty_service",
		},
	})
}

func testSweepEventOrchestration(region string) error {
	ctx := context.Background()
	resp, err := testAccProvider.client.ListOrchestrationsWithContext(ctx, pagerduty.ListOrchestrationsOptions{})
	if err != nil {
		return err
	}

	for _, orchestration := range resp.Orchestrations {
		if strings.HasPrefix(orchestration.Name, "tf-orchestration-") {
			log.Printf("Destroying Event Orchestration %s (%s)", orchestration.Name, orchestration.ID)
			if err := testAccProvider.client.DeleteOrchestrationWithContext(ctx, orchestration.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

func TestAccPagerDutyEventOrchestration_Basic(t *testing.T) {
	name := fmt.Sprintf("tf-orchestration-%s", acctest.RandString(5))
	description := fmt.Sprintf("tf-description-%s", acctest.RandString(5))
	nameUpdated := fmt.Sprintf("tf-name-%s", acctest.RandString(5))
	descriptionUpdated := fmt.Sprintf("tf-description-%s", acctest.RandString(5))
	team1 := fmt.Sprintf("tf-team-%s", acctest.RandString(5))
	team2 := fmt.Sprintf("tf-team-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckPagerDutyEventOrchestrationDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyEventOrchestrationConfigNameOnly(name),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyEventOrchestrationExists("pagerduty_event_orchestration.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_event_orchestration.foo", "name", name,
					),
					resource.TestCheckResourceAttr(
						"pagerduty_event_orchestration.foo", "description", "",
					),
					resource.TestCheckResourceAttr(
						"pagerduty_event_orchestration.foo", "team.#", "0",
					),
				),
			},
			{
				Config: testAccCheckPagerDutyEventOrchestrationConfig(name, description, team1, team2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyEventOrchestrationExists("pagerduty_event_orchestration.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_event_orchestration.foo", "name", name,
					),
					resource.TestCheckResourceAttr(
						"pagerduty_event_orchestration.foo", "description", description,
					),
					testAccCheckPagerDutyEventOrchestrationTeamMatch("pagerduty_event_orchestration.foo", "pagerduty_team.foo"),
				),
			},
			{
				Config: testAccCheckPagerDutyEventOrchestrationConfigUpdated(nameUpdated, descriptionUpdated, team1, team2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyEventOrchestrationExists("pagerduty_event_orchestration.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_event_orchestration.foo", "name", nameUpdated,
					),
					resource.TestCheckResourceAttr(
						"pagerduty_event_orchestration.foo", "description", descriptionUpdated,
					),
					testAccCheckPagerDutyEventOrchestrationTeamMatch("pagerduty_event_orchestration.foo", "pagerduty_team.bar"),
				),
			},
			{
				Config: testAccCheckPagerDutyEventOrchestrationConfigDescriptionTeamDeleted(nameUpdated, team1, team2),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyEventOrchestrationExists("pagerduty_event_orchestration.foo"),
					resource.TestCheckResourceAttr(
						"pagerduty_event_orchestration.foo", "name", nameUpdated,
					),
					resource.TestCheckResourceAttr(
						"pagerduty_event_orchestration.foo", "description", "",
					),
					resource.TestCheckResourceAttr(
						"pagerduty_event_orchestration.foo", "team.#", "0",
					),
				),
			},
		},
	})
}

func testAccCheckPagerDutyEventOrchestrationDestroy(s *terraform.State) error {
	ctx := context.Background()
	for _, r := range s.RootModule().Resources {
		if r.Type != "pagerduty_event_orchestration" {
			continue
		}
		opts := &pagerduty.GetOrchestrationOptions{}
		if _, err := testAccProvider.client.GetOrchestrationWithContext(ctx, r.Primary.ID, opts); err == nil {
			return fmt.Errorf("Event Orchestration still exists")
		}
	}
	return nil
}

func testAccCheckPagerDutyEventOrchestrationExists(rn string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ctx := context.Background()

		orch, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("Not found: %s", rn)
		}
		if orch.Primary.ID == "" {
			return fmt.Errorf("No Event Orchestration ID is set")
		}

		opts := &pagerduty.GetOrchestrationOptions{}
		found, err := testAccProvider.client.GetOrchestrationWithContext(ctx, orch.Primary.ID, opts)
		if err != nil {
			return err
		}
		if found.ID != orch.Primary.ID {
			return fmt.Errorf("Event Orchrestration not found: %v - %v", orch.Primary.ID, found)
		}

		return nil
	}
}

func testAccCheckPagerDutyEventOrchestrationTeamMatch(orchName, teamName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		o, orchOk := s.RootModule().Resources[orchName]

		if !orchOk {
			return fmt.Errorf("Not found: %s", orchName)
		}

		t, tOk := s.RootModule().Resources[teamName]
		if !tOk {
			return fmt.Errorf("Not found: %s", teamName)
		}

		var otId = o.Primary.Attributes["team"]
		var tId = t.Primary.Attributes["id"]

		if otId != tId {
			return fmt.Errorf("Event Orchestration team ID (%v) not matching provided team ID: %v", otId, tId)
		}

		return nil
	}
}

func testAccCheckPagerDutyEventOrchestrationConfigNameOnly(n string) string {
	return fmt.Sprintf(`

resource "pagerduty_event_orchestration" "foo" {
	name = "%s"
}
`, n)
}

func testAccCheckPagerDutyEventOrchestrationConfig(name, description, team1, team2 string) string {
	return fmt.Sprintf(`

resource "pagerduty_team" "foo" {
	name = "%s"
}
resource "pagerduty_team" "bar" {
	name = "%s"
}
resource "pagerduty_event_orchestration" "foo" {
	name = "%s"
	description = "%s"
	team = pagerduty_team.foo.id
}
`, team1, team2, name, description)
}

func testAccCheckPagerDutyEventOrchestrationConfigUpdated(name, description, team1, team2 string) string {
	return fmt.Sprintf(`

resource "pagerduty_team" "foo" {
	name = "%s"
}
resource "pagerduty_team" "bar" {
	name = "%s"
}
resource "pagerduty_event_orchestration" "foo" {
	name = "%s"
	description = "%s"
	team = pagerduty_team.bar.id
}
`, team1, team2, name, description)
}

func testAccCheckPagerDutyEventOrchestrationConfigDescriptionTeamDeleted(name, team1, team2 string) string {
	return fmt.Sprintf(`

resource "pagerduty_team" "foo" {
	name = "%s"
}
resource "pagerduty_team" "bar" {
	name = "%s"
}
resource "pagerduty_event_orchestration" "foo" {
	name = "%s"
}
`, team1, team2, name)
}
