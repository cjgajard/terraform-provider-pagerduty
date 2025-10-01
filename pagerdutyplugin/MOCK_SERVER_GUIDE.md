# PagerDuty Schedule V2 Mock Server Guide

## Overview

This guide explains how to use the mock HTTP server for testing the `pagerduty_schedule_v2` resource without making actual API calls to PagerDuty. The mock server simulates the Flexible Schedule API based on the structure defined in `flexible_schedule_response.json`.

## Files

- `resource_pagerduty_schedule_v2_mock_test.go` - Mock server implementation
- `resource_pagerduty_schedule_v2_unit_test.go` - Unit tests using the mock
- `resource_pagerduty_schedule_v2_mock_integration_test.go` - Integration examples
- `flexible_schedule_response.json` - Reference API response structure

## Mock Server Architecture

### Core Components

```go
type mockFlexibleScheduleServer struct {
    server    *httptest.Server       // HTTP test server
    schedules map[string]*FlexibleSchedule  // In-memory storage
    mutex     sync.RWMutex          // Thread-safe access
    idCounter int                   // ID generation
}
```

### Supported Operations

| HTTP Method | Endpoint | Description |
|------------|----------|-------------|
| POST | `/schedules` | Create a new schedule |
| GET | `/schedules/{id}` | Retrieve a schedule |
| PUT | `/schedules/{id}` | Update a schedule |
| DELETE | `/schedules/{id}` | Delete a schedule |

## Quick Start

### 1. Basic Mock Server Usage

```go
import "testing"

func TestYourFeature(t *testing.T) {
    // Create mock server
    mock := newMockFlexibleScheduleServer()
    defer mock.Close()

    // Use mock.URL() as the API endpoint
    client := createClientWithBaseURL(mock.URL())

    // Make API calls - they hit the mock server
    schedule, err := client.CreateFlexibleSchedule(...)
}
```

### 2. Unit Testing with Mock

```go
func TestMockFlexibleScheduleServer_Create(t *testing.T) {
    mock := newMockFlexibleScheduleServer()
    defer mock.Close()

    schedule := createMockScheduleForTest("Test", "America/Chicago", "USER123")

    // Make HTTP request to mock
    reqBody := map[string]interface{}{"schedule": schedule}
    reqJSON, _ := json.Marshal(reqBody)

    resp, err := http.Post(
        fmt.Sprintf("%s/schedules", mock.URL()),
        "application/json",
        strings.NewReader(string(reqJSON)),
    )

    // Assert responses
    assert.Equal(t, http.StatusCreated, resp.StatusCode)
}
```

### 3. Integration Testing

```go
func TestAccPagerDutyScheduleV2_WithMock(t *testing.T) {
    mock := newMockFlexibleScheduleServer()
    defer mock.Close()

    // Override API URL
    os.Setenv("PAGERDUTY_API_URL_OVERRIDE", mock.URL())
    defer os.Unsetenv("PAGERDUTY_API_URL_OVERRIDE")

    // Run Terraform acceptance tests
    resource.Test(t, resource.TestCase{
        // ... test configuration
    })
}
```

## Mock Response Generation

### Automatic ID Generation

The mock server automatically generates IDs for:
- Schedules: `P1`, `P2`, etc.
- Rotations: `ROT{schedule_id}{rotation_index}`
- Events: `EVT{schedule_id}{rotation_index}{event_index}`

### Final Schedule Generation

The mock automatically generates a `final_schedule` for each schedule:

```go
{
  "type": "final_schedule",
  "rendered_coverage_percentage": 93.6,
  "computed_shift_assignments": [
    // Generated shift assignments based on rotations
  ]
}
```

This mimics the real API's behavior of computing shift assignments.

### Example Generated Response

```json
{
  "schedule": {
    "id": "P1",
    "type": "schedule",
    "name": "Test Schedule",
    "time_zone": "America/Chicago",
    "rotations": [
      {
        "id": "ROT11",
        "type": "schedule_rotation",
        "name": "Test Rotation",
        "events": [
          {
            "id": "EVT111",
            "type": "schedule_event",
            // ... event details
          }
        ]
      }
    ],
    "final_schedule": {
      // Automatically generated
    }
  }
}
```

## Helper Functions

### createMockScheduleForTest

Creates a simple schedule for testing:

```go
schedule := createMockScheduleForTest(
    "My Schedule",      // name
    "America/Chicago",  // timezone
    "PUSER123",        // user_id
)
```

### generateMockScheduleFromJSON

Returns a complete schedule matching `flexible_schedule_response.json`:

```go
schedule := generateMockScheduleFromJSON()
// Returns exact structure from the JSON file
```

## Running Tests

### Unit Tests Only

```bash
go test -v -run TestMockFlexibleScheduleServer ./pagerdutyplugin
```

### Specific Mock Test

```bash
go test -v -run TestMockFlexibleScheduleServer_Create ./pagerdutyplugin
```

### Integration Tests with Mock

```bash
PAGERDUTY_USE_MOCK=1 go test -v -run TestAccPagerDutyScheduleV2_WithMock ./pagerdutyplugin
```

### All Mock Tests

```bash
go test -v -run "Mock" ./pagerdutyplugin
```

### Benchmark Mock Performance

```bash
go test -bench=BenchmarkMockScheduleOperations ./pagerdutyplugin
```

## Test Coverage

The mock server covers all CRUD operations:

### Create Tests
- ✅ Basic schedule creation
- ✅ Multiple rotations
- ✅ Different assignment strategies
- ✅ ID generation
- ✅ Final schedule computation

### Read Tests
- ✅ Retrieve existing schedule
- ✅ 404 for non-existent schedules
- ✅ All fields preserved

### Update Tests
- ✅ Name and description updates
- ✅ Rotation changes
- ✅ ID preservation
- ✅ Final schedule regeneration

### Delete Tests
- ✅ Successful deletion
- ✅ 404 after deletion
- ✅ Idempotency

### Advanced Tests
- ✅ Concurrent operations
- ✅ Thread safety
- ✅ Invalid JSON handling
- ✅ Complex scenarios

## Mock vs Real API

### Differences

| Feature | Mock | Real API |
|---------|------|----------|
| Authentication | Not validated | Required |
| Rate limiting | None | Yes |
| Validation | Basic | Comprehensive |
| Shift computation | Simple generation | Complex calculation |
| Coverage percentage | Fixed (93.6%) | Actually computed |

### What Mock DOES

✅ Handle all CRUD operations
✅ Generate appropriate IDs
✅ Maintain state in memory
✅ Return proper HTTP status codes
✅ Thread-safe concurrent access
✅ Generate final_schedule structure
✅ Preserve all schedule fields

### What Mock DOESN'T

❌ Validate recurrence rules
❌ Check timezone validity
❌ Compute actual shift assignments
❌ Handle query parameters
❌ Validate user references
❌ Check for conflicts

## Advanced Usage

### Custom Mock Configuration

```go
mock := newMockFlexibleScheduleServer()

// Pre-populate schedules
mock.mutex.Lock()
mock.schedules["CUSTOM1"] = &pagerduty.FlexibleSchedule{
    ID: "CUSTOM1",
    // ... schedule details
}
mock.mutex.Unlock()

// Use in tests
```

### Inspecting Mock State

```go
func TestWithStateInspection(t *testing.T) {
    mock := newMockFlexibleScheduleServer()
    defer mock.Close()

    // ... perform operations

    // Check mock state
    mock.mutex.RLock()
    scheduleCount := len(mock.schedules)
    mock.mutex.RUnlock()

    assert.Equal(t, 5, scheduleCount)
}
```

### Concurrent Testing

```go
func TestConcurrentOperations(t *testing.T) {
    mock := newMockFlexibleScheduleServer()
    defer mock.Close()

    // Spawn multiple goroutines
    for i := 0; i < 100; i++ {
        go func(idx int) {
            // Each goroutine creates a schedule
            // Mock handles concurrency safely
        }(i)
    }
}
```

## Debugging Tips

### Enable Verbose Logging

```bash
go test -v -run TestMock ./pagerdutyplugin 2>&1 | tee test.log
```

### Inspect HTTP Traffic

The mock server logs can be enhanced:

```go
func (m *mockFlexibleScheduleServer) handler(w http.ResponseWriter, r *http.Request) {
    log.Printf("Mock Request: %s %s", r.Method, r.URL.Path)
    // ... handler logic
}
```

### Check Response Bodies

```go
var response map[string]*pagerduty.FlexibleSchedule
json.NewDecoder(resp.Body).Decode(&response)
spew.Dump(response)  // Pretty print response
```

## Migration Path

### Phase 1: Development (Current)
Use mock server for all development and unit testing.

### Phase 2: API Preview
When API becomes available:
1. Keep mock tests for fast feedback
2. Add real API tests with feature flags
3. Compare mock vs real responses

### Phase 3: Production
1. Keep mock tests for CI/CD speed
2. Use real API for nightly integration tests
3. Maintain mock for offline development

## Example Test Scenarios

### Scenario 1: Basic CRUD Cycle

```go
func TestBasicCRUD(t *testing.T) {
    mock := newMockFlexibleScheduleServer()
    defer mock.Close()

    // CREATE
    schedule := createMockScheduleForTest("Test", "America/Chicago", "U123")
    created := createSchedule(mock.URL(), schedule)
    assert.NotEmpty(t, created.ID)

    // READ
    retrieved := getSchedule(mock.URL(), created.ID)
    assert.Equal(t, created.Name, retrieved.Name)

    // UPDATE
    retrieved.Name = "Updated"
    updated := updateSchedule(mock.URL(), created.ID, retrieved)
    assert.Equal(t, "Updated", updated.Name)

    // DELETE
    deleteSchedule(mock.URL(), created.ID)

    // VERIFY DELETED
    _, err := getSchedule(mock.URL(), created.ID)
    assert.Error(t, err)
}
```

### Scenario 2: Complex Multi-Rotation Schedule

```go
func TestMultipleRotations(t *testing.T) {
    mock := newMockFlexibleScheduleServer()
    defer mock.Close()

    // Use pre-built complex schedule
    schedule := generateMockScheduleFromJSON()

    created := createSchedule(mock.URL(), schedule)

    assert.Len(t, created.Rotations, 2)
    assert.Equal(t, "Week Days", created.Rotations[0].Name)
    assert.Equal(t, "Weekends", created.Rotations[1].Name)

    // Verify final schedule
    assert.NotNil(t, created.FinalSchedule)
    assert.Greater(t, created.FinalSchedule.RenderedCoveragePercentage, 0.0)
}
```

## Troubleshooting

### Test Hangs
- Ensure `mock.Close()` is called with `defer`
- Check for deadlocks in concurrent tests

### Unexpected 404
- Verify schedule was created successfully
- Check ID is being passed correctly
- Inspect mock.schedules map

### JSON Decode Errors
- Verify request/response structure
- Check for nil pointers in nested structures
- Use proper JSON tags in structs

### Race Conditions
- Always use `mock.mutex` for direct state access
- Mock server handles its own locking for API calls

## Best Practices

1. **Always defer Close()**: `defer mock.Close()`
2. **Use helper functions**: `createMockScheduleForTest()` for consistency
3. **Test failure paths**: Not just happy paths
4. **Verify state**: Check mock.schedules when needed
5. **Concurrent safety**: Mock handles this, but test it
6. **Clean fixtures**: Use consistent test data
7. **Assert responses**: Don't just check status codes
8. **Benchmark**: Use benchmarks for performance testing

## Contributing

When adding new features to the mock:

1. Update all CRUD handlers
2. Add corresponding unit tests
3. Update this documentation
4. Ensure thread safety
5. Match real API behavior
6. Add integration examples

## Resources

- [PagerDuty API Documentation](https://developer.pagerduty.com/)
- [Terraform Plugin Testing](https://developer.hashicorp.com/terraform/plugin/testing)
- [Go httptest Package](https://pkg.go.dev/net/http/httptest)
- `flexible_schedule_response.json` - API response reference
