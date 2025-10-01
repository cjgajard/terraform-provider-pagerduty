package pagerduty

import (
	"fmt"
	"log"
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func init() {
	resource.AddTestSweepers("pagerduty_schedule_v2", &resource.Sweeper{
		Name: "pagerduty_schedule_v2",
		F:    testSweepScheduleV2,
	})
}

func testSweepScheduleV2(region string) error {
	// Note: When the API is fully implemented, this will clean up test schedules
	// For now, this is a placeholder that would list and delete test schedules

	// Example implementation (commented out until API is ready):
	// response, err := testAccProvider.client.ListFlexibleSchedules(ctx, pagerduty.ListFlexibleScheduleOptions{})
	// if err != nil {
	// 	return err
	// }
	//
	// for _, schedule := range response.Schedules {
	// 	if strings.HasPrefix(schedule.Name, "test") || strings.HasPrefix(schedule.Name, "tf-") {
	// 		log.Printf("Destroying schedule %s (%s)", schedule.Name, schedule.ID)
	// 		if err := testAccProvider.client.DeleteFlexibleSchedule(schedule.ID); err != nil {
	// 			return err
	// 		}
	// 	}
	// }

	log.Printf("Schedule V2 sweep - API not fully implemented yet")
	return nil
}

// TestAccPagerDutyScheduleV2_Basic tests the basic create, read, and update operations
// for a flexible schedule using the every_member_assignment_strategy
func TestAccPagerDutyScheduleV2_Basic(t *testing.T) {
	username := fmt.Sprintf("tf-%s", acctest.RandString(5))
	email := fmt.Sprintf("%s@foo.test", username)
	scheduleName := fmt.Sprintf("tf-schedule-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckPagerDutyScheduleV2Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyScheduleV2ConfigBasic(username, email, scheduleName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyScheduleV2Exists("pagerduty_schedule_v2.test"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.test", "name", scheduleName),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.test", "description", "Managed by Terraform"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.test", "time_zone", "America/Chicago"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.test", "rotation.#", "1"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.test", "rotation.0.name", "Week Days"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.test", "rotation.0.assignment_strategy", "every_member_assignment_strategy"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.test", "rotation.0.member.#", "1"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.test", "rotation.0.event.#", "1"),
					resource.TestCheckResourceAttrSet("pagerduty_schedule_v2.test", "id"),
					resource.TestCheckResourceAttrSet("pagerduty_schedule_v2.test", "rotation.0.id"),
					resource.TestCheckResourceAttrSet("pagerduty_schedule_v2.test", "rotation.0.event.0.id"),
				),
			},
			{
				Config: testAccCheckPagerDutyScheduleV2ConfigUpdated(username, email, scheduleName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyScheduleV2Exists("pagerduty_schedule_v2.test"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.test", "name", fmt.Sprintf("%s-updated", scheduleName)),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.test", "description", "Updated by Terraform"),
				),
			},
		},
	})
}

// TestAccPagerDutyScheduleV2_MultipleRotations tests a schedule with multiple rotations
// similar to the flexible_schedule_response.json example
func TestAccPagerDutyScheduleV2_MultipleRotations(t *testing.T) {
	username1 := fmt.Sprintf("tf-%s", acctest.RandString(5))
	username2 := fmt.Sprintf("tf-%s", acctest.RandString(5))
	username3 := fmt.Sprintf("tf-%s", acctest.RandString(5))
	email1 := fmt.Sprintf("%s@foo.test", username1)
	email2 := fmt.Sprintf("%s@foo.test", username2)
	email3 := fmt.Sprintf("%s@foo.test", username3)
	scheduleName := fmt.Sprintf("tf-schedule-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckPagerDutyScheduleV2Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyScheduleV2ConfigMultipleRotations(
					username1, username2, username3,
					email1, email2, email3,
					scheduleName,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyScheduleV2Exists("pagerduty_schedule_v2.test"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.test", "name", scheduleName),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.test", "rotation.#", "2"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.test", "rotation.0.name", "Week Days"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.test", "rotation.1.name", "Weekends"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.test", "rotation.0.assignment_strategy", "every_member_assignment_strategy"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.test", "rotation.1.assignment_strategy", "rotating_member_assignment_strategy"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.test", "rotation.0.member.#", "2"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.test", "rotation.1.member.#", "1"),
				),
			},
		},
	})
}

// TestAccPagerDutyScheduleV2_RotatingStrategy tests a schedule using rotating_member_assignment_strategy
func TestAccPagerDutyScheduleV2_RotatingStrategy(t *testing.T) {
	username1 := fmt.Sprintf("tf-%s", acctest.RandString(5))
	username2 := fmt.Sprintf("tf-%s", acctest.RandString(5))
	email1 := fmt.Sprintf("%s@foo.test", username1)
	email2 := fmt.Sprintf("%s@foo.test", username2)
	scheduleName := fmt.Sprintf("tf-schedule-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckPagerDutyScheduleV2Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyScheduleV2ConfigRotatingStrategy(
					username1, username2,
					email1, email2,
					scheduleName,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyScheduleV2Exists("pagerduty_schedule_v2.test"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.test", "rotation.0.assignment_strategy", "rotating_member_assignment_strategy"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.test", "rotation.0.shifts_per_member", "2"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.test", "rotation.0.member.#", "2"),
				),
			},
		},
	})
}

// TestAccPagerDutyScheduleV2_EffectiveUntil tests a schedule with effective_until date
func TestAccPagerDutyScheduleV2_EffectiveUntil(t *testing.T) {
	username := fmt.Sprintf("tf-%s", acctest.RandString(5))
	email := fmt.Sprintf("%s@foo.test", username)
	scheduleName := fmt.Sprintf("tf-schedule-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckPagerDutyScheduleV2Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyScheduleV2ConfigEffectiveUntil(username, email, scheduleName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyScheduleV2Exists("pagerduty_schedule_v2.test"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.test", "rotation.0.event.0.effective_until", "2025-12-31T23:59:59-05:00"),
				),
			},
		},
	})
}

// TestAccPagerDutyScheduleV2_WithTeams tests a schedule associated with teams
func TestAccPagerDutyScheduleV2_WithTeams(t *testing.T) {
	username := fmt.Sprintf("tf-%s", acctest.RandString(5))
	email := fmt.Sprintf("%s@foo.test", username)
	scheduleName := fmt.Sprintf("tf-schedule-%s", acctest.RandString(5))
	teamName := fmt.Sprintf("tf-team-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckPagerDutyScheduleV2Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyScheduleV2ConfigWithTeams(username, email, scheduleName, teamName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyScheduleV2Exists("pagerduty_schedule_v2.test"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.test", "teams.#", "1"),
				),
			},
		},
	})
}

// TestAccPagerDutyScheduleV2_FinalSchedule tests that final_schedule is computed correctly
func TestAccPagerDutyScheduleV2_FinalSchedule(t *testing.T) {
	username := fmt.Sprintf("tf-%s", acctest.RandString(5))
	email := fmt.Sprintf("%s@foo.test", username)
	scheduleName := fmt.Sprintf("tf-schedule-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV5ProviderFactories: testAccProtoV5ProviderFactories(),
		CheckDestroy:             testAccCheckPagerDutyScheduleV2Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyScheduleV2ConfigBasic(username, email, scheduleName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyScheduleV2Exists("pagerduty_schedule_v2.test"),
					resource.TestCheckResourceAttrSet("pagerduty_schedule_v2.test", "final_schedule.0.rendered_coverage_percentage"),
					resource.TestCheckResourceAttrSet("pagerduty_schedule_v2.test", "final_schedule.0.computed_shift_assignment.#"),
				),
			},
		},
	})
}

func testAccCheckPagerDutyScheduleV2Destroy(s *terraform.State) error {
	for _, r := range s.RootModule().Resources {
		if r.Type != "pagerduty_schedule_v2" {
			continue
		}

		if _, err := testAccProvider.client.GetFlexibleSchedule(r.Primary.ID, pagerduty.GetFlexibleScheduleOptions{}); err == nil {
			return fmt.Errorf("Flexible Schedule still exists")
		}
	}
	return nil
}

func testAccCheckPagerDutyScheduleV2Exists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Flexible Schedule ID is set")
		}

		found, err := testAccProvider.client.GetFlexibleSchedule(rs.Primary.ID, pagerduty.GetFlexibleScheduleOptions{})
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Flexible Schedule not found: %v - %v", rs.Primary.ID, found)
		}

		return nil
	}
}

func testAccExternallyDestroyScheduleV2(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Flexible Schedule ID is set")
		}

		if err := testAccProvider.client.DeleteFlexibleSchedule(rs.Primary.ID); err != nil {
			return err
		}

		return nil
	}
}

// Configuration templates

func testAccCheckPagerDutyScheduleV2ConfigBasic(username, email, scheduleName string) string {
	return fmt.Sprintf(`
resource "pagerduty_user" "test" {
  name  = "%s"
  email = "%s"
}

resource "pagerduty_schedule_v2" "test" {
  name      = "%s"
  time_zone = "America/Chicago"

  rotation {
    name                = "Week Days"
    assignment_strategy = "every_member_assignment_strategy"

    member {
      user_id = pagerduty_user.test.id
      name    = "sara"
      color   = "yellow"
    }

    event {
      start_time      = "2025-07-14T08:00:00-05:00"
      end_time        = "2025-07-18T17:00:00-05:00"
      time_zone       = "America/Chicago"
      effective_since = "2025-07-09T12:00:00-05:00"
      recurrence      = ["RRULE:FREQ=WEEKLY"]
      assignment_strategy = "every_member_assignment_strategy"
    }
  }
}
`, username, email, scheduleName)
}

func testAccCheckPagerDutyScheduleV2ConfigUpdated(username, email, scheduleName string) string {
	return fmt.Sprintf(`
resource "pagerduty_user" "test" {
  name  = "%s"
  email = "%s"
}

resource "pagerduty_schedule_v2" "test" {
  name        = "%s-updated"
  description = "Updated by Terraform"
  time_zone   = "America/Chicago"

  rotation {
    name                = "Week Days"
    assignment_strategy = "every_member_assignment_strategy"

    member {
      user_id = pagerduty_user.test.id
      name    = "sara"
      color   = "yellow"
    }

    event {
      start_time      = "2025-07-14T08:00:00-05:00"
      end_time        = "2025-07-18T17:00:00-05:00"
      time_zone       = "America/Chicago"
      effective_since = "2025-07-09T12:00:00-05:00"
      recurrence      = ["RRULE:FREQ=WEEKLY"]
      assignment_strategy = "every_member_assignment_strategy"
    }
  }
}
`, username, email, scheduleName)
}

func testAccCheckPagerDutyScheduleV2ConfigMultipleRotations(user1, user2, user3, email1, email2, email3, scheduleName string) string {
	return fmt.Sprintf(`
resource "pagerduty_user" "test1" {
  name  = "%s"
  email = "%s"
}

resource "pagerduty_user" "test2" {
  name  = "%s"
  email = "%s"
}

resource "pagerduty_user" "test3" {
  name  = "%s"
  email = "%s"
}

resource "pagerduty_schedule_v2" "test" {
  name        = "%s"
  description = "Support Team Schedule covering weekdays and weekends"
  time_zone   = "America/Chicago"

  rotation {
    name                = "Week Days"
    assignment_strategy = "every_member_assignment_strategy"

    member {
      user_id = pagerduty_user.test1.id
      name    = "sara"
      color   = "yellow"
    }

    member {
      user_id = pagerduty_user.test2.id
      name    = "jill"
      color   = "green"
    }

    event {
      start_time      = "2025-07-14T08:00:00-05:00"
      end_time        = "2025-07-18T17:00:00-05:00"
      time_zone       = "America/Chicago"
      effective_since = "2025-07-09T12:00:00-05:00"
      recurrence      = ["RRULE:FREQ=WEEKLY"]
      assignment_strategy = "every_member_assignment_strategy"
    }
  }

  rotation {
    name                = "Weekends"
    assignment_strategy = "rotating_member_assignment_strategy"
    shifts_per_member   = 1

    member {
      user_id = pagerduty_user.test3.id
      name    = "bob"
      color   = "blue"
    }

    event {
      start_time      = "2025-07-18T17:00:00-05:00"
      end_time        = "2025-07-21T08:00:00-05:00"
      time_zone       = "America/Chicago"
      effective_since = "2025-07-09T12:00:00-05:00"
      recurrence      = ["RRULE:FREQ=WEEKLY"]
      assignment_strategy = "rotating_member_assignment_strategy"
    }
  }
}
`, user1, email1, user2, email2, user3, email3, scheduleName)
}

func testAccCheckPagerDutyScheduleV2ConfigRotatingStrategy(user1, user2, email1, email2, scheduleName string) string {
	return fmt.Sprintf(`
resource "pagerduty_user" "test1" {
  name  = "%s"
  email = "%s"
}

resource "pagerduty_user" "test2" {
  name  = "%s"
  email = "%s"
}

resource "pagerduty_schedule_v2" "test" {
  name      = "%s"
  time_zone = "America/Chicago"

  rotation {
    name                = "Rotating Shift"
    assignment_strategy = "rotating_member_assignment_strategy"
    shifts_per_member   = 2

    member {
      user_id = pagerduty_user.test1.id
      name    = "jill"
      color   = "green"
    }

    member {
      user_id = pagerduty_user.test2.id
      name    = "sara"
      color   = "yellow"
    }

    event {
      start_time      = "2025-07-18T17:00:00-05:00"
      end_time        = "2025-07-21T08:00:00-05:00"
      time_zone       = "America/Chicago"
      effective_since = "2025-07-09T12:00:00-05:00"
      recurrence      = ["RRULE:FREQ=WEEKLY"]
      assignment_strategy = "rotating_member_assignment_strategy"
    }
  }
}
`, user1, email1, user2, email2, scheduleName)
}

func testAccCheckPagerDutyScheduleV2ConfigEffectiveUntil(username, email, scheduleName string) string {
	return fmt.Sprintf(`
resource "pagerduty_user" "test" {
  name  = "%s"
  email = "%s"
}

resource "pagerduty_schedule_v2" "test" {
  name      = "%s"
  time_zone = "America/Chicago"

  rotation {
    name                = "Limited Time Rotation"
    assignment_strategy = "every_member_assignment_strategy"

    member {
      user_id = pagerduty_user.test.id
      name    = "jill"
      color   = "green"
    }

    event {
      start_time       = "2025-07-18T17:00:00-05:00"
      end_time         = "2025-07-21T08:00:00-05:00"
      time_zone        = "America/Chicago"
      effective_since  = "2025-07-09T12:00:00-05:00"
      effective_until  = "2025-12-31T23:59:59-05:00"
      recurrence       = ["RRULE:FREQ=WEEKLY"]
      assignment_strategy = "every_member_assignment_strategy"
    }
  }
}
`, username, email, scheduleName)
}

func testAccCheckPagerDutyScheduleV2ConfigWithTeams(username, email, scheduleName, teamName string) string {
	return fmt.Sprintf(`
resource "pagerduty_user" "test" {
  name  = "%s"
  email = "%s"
}

resource "pagerduty_team" "test" {
  name = "%s"
}

resource "pagerduty_schedule_v2" "test" {
  name      = "%s"
  time_zone = "America/Chicago"
  teams     = [pagerduty_team.test.id]

  rotation {
    name                = "Week Days"
    assignment_strategy = "every_member_assignment_strategy"

    member {
      user_id = pagerduty_user.test.id
      name    = "sara"
      color   = "yellow"
    }

    event {
      start_time      = "2025-07-14T08:00:00-05:00"
      end_time        = "2025-07-18T17:00:00-05:00"
      time_zone       = "America/Chicago"
      effective_since = "2025-07-09T12:00:00-05:00"
      recurrence      = ["RRULE:FREQ=WEEKLY"]
      assignment_strategy = "every_member_assignment_strategy"
    }
  }
}
`, username, email, teamName, scheduleName)
}
