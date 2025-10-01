package pagerduty

import (
	"fmt"
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccPagerDutyScheduleV2_Basic(t *testing.T) {
	username := fmt.Sprintf("tf-%s", acctest.RandString(5))
	email := fmt.Sprintf("%s@foo.test", username)
	scheduleName := fmt.Sprintf("tf-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyScheduleV2Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyScheduleV2Config(username, email, scheduleName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyScheduleV2Exists("pagerduty_schedule_v2.foo"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.foo", "name", scheduleName),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.foo", "description", "Managed by Terraform"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.foo", "time_zone", "America/New_York"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.foo", "rotation.#", "1"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.foo", "rotation.0.name", "Night Shift"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.foo", "rotation.0.assignment_strategy", "rotating_member_assignment_strategy"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.foo", "rotation.0.shifts_per_member", "1"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.foo", "rotation.0.member.#", "1"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.foo", "rotation.0.member.0.name", "Engineer On Call"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.foo", "rotation.0.member.0.color", "#FF5733"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.foo", "rotation.0.event.#", "1"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.foo", "rotation.0.event.0.start_time", "2015-11-06T20:00:00-05:00"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.foo", "rotation.0.event.0.end_time", "2015-11-07T20:00:00-05:00"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.foo", "rotation.0.event.0.effective_since", "2015-11-06T20:00:00-05:00"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.foo", "rotation.0.event.0.recurrence.#", "1"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.foo", "rotation.0.event.0.recurrence.0", "FREQ=DAILY"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.foo", "rotation.0.restriction.#", "1"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.foo", "rotation.0.restriction.0.type", "daily_restriction"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.foo", "rotation.0.restriction.0.start_time_of_day", "08:00:00"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.foo", "rotation.0.restriction.0.duration_seconds", "32400"),
				),
			},
			{
				Config: testAccCheckPagerDutyScheduleV2ConfigUpdated(username, email, scheduleName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyScheduleV2Exists("pagerduty_schedule_v2.foo"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.foo", "name", fmt.Sprintf("%s-updated", scheduleName)),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.foo", "description", "Updated by Terraform"),
				),
			},
		},
	})
}

func TestAccPagerDutyScheduleV2_EveryMemberStrategy(t *testing.T) {
	username1 := fmt.Sprintf("tf-%s", acctest.RandString(5))
	username2 := fmt.Sprintf("tf-%s", acctest.RandString(5))
	email1 := fmt.Sprintf("%s@foo.test", username1)
	email2 := fmt.Sprintf("%s@foo.test", username2)
	scheduleName := fmt.Sprintf("tf-%s", acctest.RandString(5))

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckPagerDutyScheduleV2Destroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCheckPagerDutyScheduleV2ConfigEveryMember(username1, username2, email1, email2, scheduleName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckPagerDutyScheduleV2Exists("pagerduty_schedule_v2.foo"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.foo", "rotation.0.assignment_strategy", "every_member_assignment_strategy"),
					resource.TestCheckResourceAttr("pagerduty_schedule_v2.foo", "rotation.0.member.#", "2"),
				),
			},
		},
	})
}

func testAccCheckPagerDutyScheduleV2Destroy(s *terraform.State) error {
	client, _ := testAccProvider.Meta().(*Config).Client()
	for _, r := range s.RootModule().Resources {
		if r.Type != "pagerduty_schedule_v2" {
			continue
		}

		if _, err := client.GetFlexibleSchedule(r.Primary.ID, pagerduty.GetFlexibleScheduleOptions{}); err == nil {
			return fmt.Errorf("Schedule still exists")
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
			return fmt.Errorf("No Schedule ID is set")
		}

		client, _ := testAccProvider.Meta().(*Config).Client()

		found, err := client.GetFlexibleSchedule(rs.Primary.ID, pagerduty.GetFlexibleScheduleOptions{})
		if err != nil {
			return err
		}

		if found.ID != rs.Primary.ID {
			return fmt.Errorf("Schedule not found: %v - %v", rs.Primary.ID, found)
		}

		return nil
	}
}

func testAccCheckPagerDutyScheduleV2Config(username, email, schedule string) string {
	return fmt.Sprintf(`
resource "pagerduty_user" "example" {
  name  = "%s"
  email = "%s"
}

resource "pagerduty_schedule_v2" "foo" {
  name      = "%s"
  time_zone = "America/New_York"

  rotation {
    name                = "Night Shift"
    assignment_strategy = "rotating_member_assignment_strategy"
    shifts_per_member   = 1

    member {
      user_id = pagerduty_user.example.id
      name    = "Engineer On Call"
      color   = "#FF5733"
    }

    event {
      start_time        = "2015-11-06T20:00:00-05:00"
      end_time          = "2015-11-07T20:00:00-05:00"
      recurrence        = ["FREQ=DAILY"]
      effective_since   = "2015-11-06T20:00:00-05:00"
      assignment_strategy = "rotating_member_assignment_strategy"
    }

    restriction {
      type              = "daily_restriction"
      start_time_of_day = "08:00:00"
      duration_seconds  = 32400
    }
  }
}
`, username, email, schedule)
}

func testAccCheckPagerDutyScheduleV2ConfigUpdated(username, email, schedule string) string {
	return fmt.Sprintf(`
resource "pagerduty_user" "example" {
  name  = "%s"
  email = "%s"
}

resource "pagerduty_schedule_v2" "foo" {
  name        = "%s-updated"
  description = "Updated by Terraform"
  time_zone   = "America/New_York"

  rotation {
    name                = "Night Shift"
    assignment_strategy = "rotating_member_assignment_strategy"
    shifts_per_member   = 1

    member {
      user_id = pagerduty_user.example.id
      name    = "Engineer On Call"
      color   = "#FF5733"
    }

    event {
      start_time        = "2015-11-06T20:00:00-05:00"
      end_time          = "2015-11-07T20:00:00-05:00"
      recurrence        = ["FREQ=DAILY"]
      effective_since   = "2015-11-06T20:00:00-05:00"
      assignment_strategy = "rotating_member_assignment_strategy"
    }

    restriction {
      type              = "daily_restriction"
      start_time_of_day = "08:00:00"
      duration_seconds  = 32400
    }
  }
}
`, username, email, schedule)
}

func testAccCheckPagerDutyScheduleV2ConfigEveryMember(username1, username2, email1, email2, schedule string) string {
	return fmt.Sprintf(`
resource "pagerduty_user" "example1" {
  name  = "%s"
  email = "%s"
}

resource "pagerduty_user" "example2" {
  name  = "%s"
  email = "%s"
}

resource "pagerduty_schedule_v2" "foo" {
  name      = "%s"
  time_zone = "America/New_York"

  rotation {
    name                = "Weekday Shift"
    assignment_strategy = "every_member_assignment_strategy"

    member {
      user_id = pagerduty_user.example1.id
      name    = "sara"
      color   = "yellow"
    }

    member {
      user_id = pagerduty_user.example2.id
      name    = "jill"
      color   = "green"
    }

    event {
      start_time        = "2025-07-14T08:00:00-05:00"
      end_time          = "2025-07-18T17:00:00-05:00"
      time_zone         = "America/Chicago"
      recurrence        = ["RRULE:FREQ=WEEKLY"]
      effective_since   = "2025-07-09T12:00:00-05:00"
      assignment_strategy = "every_member_assignment_strategy"
    }
  }
}
`, username1, email1, username2, email2, schedule)
}