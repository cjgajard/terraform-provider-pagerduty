# PagerDuty Schedule V2 Testing Documentation

## Overview

This document describes the testing approach for the `pagerduty_schedule_v2` resource, which uses the new Terraform Plugin Framework and PagerDuty's Flexible Schedule API.

## Test Structure

The acceptance tests for `pagerduty_schedule_v2` are located in:
- `pagerdutyplugin/resource_pagerduty_schedule_v2_test.go`

## API Response Structure

The Flexible Schedule API returns responses structured like the example in `flexible_schedule_response.json`. Key components include:

### Schedule Structure
```json
{
  "schedule": {
    "id": "P65784",
    "type": "schedule",
    "name": "Support Team Schedule",
    "description": "Schedule description",
    "time_zone": "America/Chicago",
    "rotations": [...],
    "final_schedule": {...}
  }
}
```

### Rotation Structure
Each rotation contains:
- `id`, `type`, `name`: Basic identifiers
- `events`: Array of schedule events with timing and recurrence rules
- Assignment strategies and member configurations

### Final Schedule
The API automatically computes a `final_schedule` that includes:
- `rendered_coverage_percentage`: Overall coverage metric
- `computed_shift_assignments`: Calculated shift assignments based on rotations

## Test Cases

### 1. TestAccPagerDutyScheduleV2_Basic
Tests basic CRUD operations:
- Create a schedule with a single rotation
- Verify all attributes are set correctly
- Update the schedule name and description
- Verify updates are applied

### 2. TestAccPagerDutyScheduleV2_MultipleRotations
Tests a schedule with multiple rotations (weekdays and weekends):
- Two rotations with different assignment strategies
- Multiple members across rotations
- Mimics the structure from `flexible_schedule_response.json`

### 3. TestAccPagerDutyScheduleV2_RotatingStrategy
Tests the `rotating_member_assignment_strategy`:
- Multiple members rotating through shifts
- `shifts_per_member` configuration
- Proper member rotation calculation

### 4. TestAccPagerDutyScheduleV2_EffectiveUntil
Tests time-bounded rotations:
- Events with `effective_until` dates
- Rotation lifecycle management

### 5. TestAccPagerDutyScheduleV2_WithTeams
Tests schedule association with teams:
- Schedule linked to team resources
- Team ID references in schedule configuration

### 6. TestAccPagerDutyScheduleV2_FinalSchedule
Tests computed attributes:
- Verifies `final_schedule` is computed by API
- Checks `rendered_coverage_percentage`
- Validates `computed_shift_assignments`

## Mock Data Setup

### Current State
The tests are structured to work with the actual PagerDuty API once it's fully implemented. Currently:

1. **API Endpoints Used:**
   - `CreateFlexibleSchedule()` - Create new schedules
   - `GetFlexibleSchedule()` - Read schedule details
   - `UpdateFlexibleSchedule()` - Update existing schedules
   - `DeleteFlexibleSchedule()` - Delete schedules

2. **Mock Considerations:**
   When the API is not yet available, you can mock responses using the structure from `flexible_schedule_response.json`:

```go
// Example mock setup (not yet implemented)
func mockFlexibleScheduleAPI() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/schedules"):
			// Return mock create response
			response := loadMockResponse("flexible_schedule_response.json")
			json.NewEncoder(w).Encode(response)
		case r.Method == "GET" && strings.Contains(r.URL.Path, "/schedules/"):
			// Return mock get response
			response := loadMockResponse("flexible_schedule_response.json")
			json.NewEncoder(w).Encode(response)
		// ... other cases
		}
	}))
}
```

## Running Tests

### Prerequisites
Set the following environment variables:
```bash
export PAGERDUTY_TOKEN="your-test-token"
export PAGERDUTY_USER_TOKEN="your-user-token"
```

### Run Specific Test
```bash
go test -v -run TestAccPagerDutyScheduleV2_Basic ./pagerdutyplugin
```

### Run All Schedule V2 Tests
```bash
go test -v -run TestAccPagerDutyScheduleV2 ./pagerdutyplugin
```

### With Parallel Execution
```bash
PAGERDUTY_PARALLEL=1 go test -v -run TestAccPagerDutyScheduleV2 ./pagerdutyplugin
```

## Test Sweepers

The test includes a sweeper function (`testSweepScheduleV2`) to clean up test resources:
- Identifies schedules with names starting with "test" or "tf-"
- Automatically deletes them to prevent test pollution
- Currently a placeholder until API is fully implemented

Run sweepers:
```bash
go test -v -sweep=all -sweep-allow-failures ./pagerdutyplugin
```

## Key Differences from SDKv2 Tests

### Plugin Framework Patterns
1. **Provider Factories**: Uses `ProtoV5ProviderFactories` instead of `Providers`
2. **Context**: All API calls use `context.Context`
3. **Type Safety**: Strong typing throughout with Plugin Framework types
4. **Diagnostics**: Better error reporting with structured diagnostics

### Test Structure
- Tests are in `pagerdutyplugin/` instead of `pagerduty/`
- Uses newer `terraform-plugin-testing` package
- Implements proper resource sweepers for cleanup

## Example Test Data Mapping

The test configurations map to the API response structure:

| Terraform Config | API Response Field |
|-----------------|-------------------|
| `name` | `schedule.name` |
| `time_zone` | `schedule.time_zone` |
| `rotation.name` | `rotations[].name` |
| `rotation.event.start_time` | `rotations[].events[].start_time.date_time` |
| `rotation.member.user_id` | `rotations[].events[].assignment_strategy.members[].user_id` |
| `final_schedule` (computed) | `schedule.final_schedule` |

## Future Enhancements

1. **HTTP Mocking**: Add httptest server for offline testing
2. **Golden Files**: Use `flexible_schedule_response.json` as test fixtures
3. **State Migration Tests**: Test migration from old schedule format
4. **Import Tests**: Verify schedule import functionality
5. **Complex Scenarios**: Test edge cases like overlapping rotations, gaps in coverage

## Troubleshooting

### Common Issues

1. **API Not Found Errors**
   - Ensure the Flexible Schedule API is available in your test account
   - Check that feature flags are enabled

2. **Test Flakiness**
   - Use proper retry logic for eventual consistency
   - Implement appropriate wait times between operations

3. **Resource Cleanup Failures**
   - Run sweepers regularly to clean up orphaned resources
   - Check for dependencies that prevent deletion

## Related Files

- `resource_pagerduty_schedule_v2.go` - Resource implementation
- `flexible_schedule_response.json` - Example API response
- `flexible_schedule.tf` - Example Terraform configuration
- `provider_test.go` - Test provider setup
