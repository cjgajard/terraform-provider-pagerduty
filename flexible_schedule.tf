resource "pagerduty_schedule_v2" "foo" {
  name      = "Daily Engineering Rotation"
  time_zone = "America/New_York"

  rotation {
    name         = "Night Shift"
    start_time   = "2015-11-06T20:00:00-05:00"
    turn_length  = 86400 # seconds
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

  teams = [pagerduty_team.example.id]
}

