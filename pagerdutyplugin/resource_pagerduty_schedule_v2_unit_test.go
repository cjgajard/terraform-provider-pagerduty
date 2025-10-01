package pagerduty

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/PagerDuty/go-pagerduty"
)

// TestMockFlexibleScheduleServer_Create tests the mock server's create functionality
func TestMockFlexibleScheduleServer_Create(t *testing.T) {
	mock := newMockFlexibleScheduleServer()
	defer mock.Close()

	schedule := createMockScheduleForTest("Test Schedule", "America/Chicago", "PUSERID1")

	// Create request
	reqBody := map[string]interface{}{
		"schedule": schedule,
	}
	reqJSON, _ := json.Marshal(reqBody)

	resp, err := http.Post(
		fmt.Sprintf("%s/schedules", mock.URL()),
		"application/json",
		strings.NewReader(string(reqJSON)),
	)
	if err != nil {
		t.Fatalf("Failed to create schedule: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, resp.StatusCode)
	}

	var response map[string]*pagerduty.FlexibleSchedule
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	created := response["schedule"]
	if created.ID == "" {
		t.Error("Expected schedule ID to be set")
	}
	if created.Name != "Test Schedule" {
		t.Errorf("Expected name 'Test Schedule', got '%s'", created.Name)
	}
	if created.TimeZone != "America/Chicago" {
		t.Errorf("Expected timezone 'America/Chicago', got '%s'", created.TimeZone)
	}
	if created.FinalSchedule == nil {
		t.Error("Expected final_schedule to be set")
	}
	if len(created.Rotations) != 1 {
		t.Errorf("Expected 1 rotation, got %d", len(created.Rotations))
	}
	if created.Rotations[0].ID == "" {
		t.Error("Expected rotation ID to be set")
	}
	if created.Rotations[0].Events[0].ID == "" {
		t.Error("Expected event ID to be set")
	}
}

// TestMockFlexibleScheduleServer_Get tests the mock server's get functionality
func TestMockFlexibleScheduleServer_Get(t *testing.T) {
	mock := newMockFlexibleScheduleServer()
	defer mock.Close()

	// First create a schedule
	schedule := createMockScheduleForTest("Test Schedule", "America/Chicago", "PUSERID1")
	reqBody := map[string]interface{}{
		"schedule": schedule,
	}
	reqJSON, _ := json.Marshal(reqBody)

	createResp, err := http.Post(
		fmt.Sprintf("%s/schedules", mock.URL()),
		"application/json",
		strings.NewReader(string(reqJSON)),
	)
	if err != nil {
		t.Fatalf("Failed to create schedule: %v", err)
	}
	defer createResp.Body.Close()

	var createResponse map[string]*pagerduty.FlexibleSchedule
	json.NewDecoder(createResp.Body).Decode(&createResponse)
	scheduleID := createResponse["schedule"].ID

	// Now get the schedule
	getResp, err := http.Get(fmt.Sprintf("%s/schedules/%s", mock.URL(), scheduleID))
	if err != nil {
		t.Fatalf("Failed to get schedule: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, getResp.StatusCode)
	}

	var getResponse map[string]*pagerduty.FlexibleSchedule
	err = json.NewDecoder(getResp.Body).Decode(&getResponse)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	retrieved := getResponse["schedule"]
	if retrieved.ID != scheduleID {
		t.Errorf("Expected ID '%s', got '%s'", scheduleID, retrieved.ID)
	}
	if retrieved.Name != "Test Schedule" {
		t.Errorf("Expected name 'Test Schedule', got '%s'", retrieved.Name)
	}
}

// TestMockFlexibleScheduleServer_Update tests the mock server's update functionality
func TestMockFlexibleScheduleServer_Update(t *testing.T) {
	mock := newMockFlexibleScheduleServer()
	defer mock.Close()

	// Create initial schedule
	schedule := createMockScheduleForTest("Test Schedule", "America/Chicago", "PUSERID1")
	reqBody := map[string]interface{}{
		"schedule": schedule,
	}
	reqJSON, _ := json.Marshal(reqBody)

	createResp, err := http.Post(
		fmt.Sprintf("%s/schedules", mock.URL()),
		"application/json",
		strings.NewReader(string(reqJSON)),
	)
	if err != nil {
		t.Fatalf("Failed to create schedule: %v", err)
	}
	defer createResp.Body.Close()

	var createResponse map[string]*pagerduty.FlexibleSchedule
	json.NewDecoder(createResp.Body).Decode(&createResponse)
	scheduleID := createResponse["schedule"].ID

	// Update the schedule
	schedule.Name = "Updated Schedule"
	schedule.Description = "Updated description"
	updateReqBody := map[string]interface{}{
		"schedule": schedule,
	}
	updateReqJSON, _ := json.Marshal(updateReqBody)

	client := &http.Client{}
	req, _ := http.NewRequest(
		"PUT",
		fmt.Sprintf("%s/schedules/%s", mock.URL(), scheduleID),
		strings.NewReader(string(updateReqJSON)),
	)
	req.Header.Set("Content-Type", "application/json")

	updateResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to update schedule: %v", err)
	}
	defer updateResp.Body.Close()

	if updateResp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, updateResp.StatusCode)
	}

	var updateResponse map[string]*pagerduty.FlexibleSchedule
	err = json.NewDecoder(updateResp.Body).Decode(&updateResponse)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	updated := updateResponse["schedule"]
	if updated.ID != scheduleID {
		t.Errorf("Expected ID '%s', got '%s'", scheduleID, updated.ID)
	}
	if updated.Name != "Updated Schedule" {
		t.Errorf("Expected name 'Updated Schedule', got '%s'", updated.Name)
	}
	if updated.Description != "Updated description" {
		t.Errorf("Expected description 'Updated description', got '%s'", updated.Description)
	}
}

// TestMockFlexibleScheduleServer_Delete tests the mock server's delete functionality
func TestMockFlexibleScheduleServer_Delete(t *testing.T) {
	mock := newMockFlexibleScheduleServer()
	defer mock.Close()

	// Create a schedule
	schedule := createMockScheduleForTest("Test Schedule", "America/Chicago", "PUSERID1")
	reqBody := map[string]interface{}{
		"schedule": schedule,
	}
	reqJSON, _ := json.Marshal(reqBody)

	createResp, err := http.Post(
		fmt.Sprintf("%s/schedules", mock.URL()),
		"application/json",
		strings.NewReader(string(reqJSON)),
	)
	if err != nil {
		t.Fatalf("Failed to create schedule: %v", err)
	}
	defer createResp.Body.Close()

	var createResponse map[string]*pagerduty.FlexibleSchedule
	json.NewDecoder(createResp.Body).Decode(&createResponse)
	scheduleID := createResponse["schedule"].ID

	// Delete the schedule
	client := &http.Client{}
	req, _ := http.NewRequest(
		"DELETE",
		fmt.Sprintf("%s/schedules/%s", mock.URL(), scheduleID),
		nil,
	)

	deleteResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to delete schedule: %v", err)
	}
	defer deleteResp.Body.Close()

	if deleteResp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d", http.StatusNoContent, deleteResp.StatusCode)
	}

	// Verify schedule is deleted
	getResp, err := http.Get(fmt.Sprintf("%s/schedules/%s", mock.URL(), scheduleID))
	if err != nil {
		t.Fatalf("Failed to get schedule: %v", err)
	}
	defer getResp.Body.Close()

	if getResp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status %d for deleted schedule, got %d", http.StatusNotFound, getResp.StatusCode)
	}
}

// TestMockFlexibleScheduleServer_GetNotFound tests 404 handling
func TestMockFlexibleScheduleServer_GetNotFound(t *testing.T) {
	mock := newMockFlexibleScheduleServer()
	defer mock.Close()

	resp, err := http.Get(fmt.Sprintf("%s/schedules/NONEXISTENT", mock.URL()))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
}

// TestGenerateMockScheduleFromJSON tests the JSON-based mock schedule generator
func TestGenerateMockScheduleFromJSON(t *testing.T) {
	schedule := generateMockScheduleFromJSON()

	if schedule.ID != "P65784" {
		t.Errorf("Expected ID 'P65784', got '%s'", schedule.ID)
	}
	if schedule.Name != "Support Team Schedule" {
		t.Errorf("Expected name 'Support Team Schedule', got '%s'", schedule.Name)
	}
	if schedule.TimeZone != "America/Chicago" {
		t.Errorf("Expected timezone 'America/Chicago', got '%s'", schedule.TimeZone)
	}
	if len(schedule.Rotations) != 2 {
		t.Errorf("Expected 2 rotations, got %d", len(schedule.Rotations))
	}

	// Check first rotation
	weekdays := schedule.Rotations[0]
	if weekdays.Name != "Week Days" {
		t.Errorf("Expected rotation name 'Week Days', got '%s'", weekdays.Name)
	}
	if len(weekdays.Events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(weekdays.Events))
	}
	if weekdays.Events[0].AssignmentStrategy.Type != "every_member_assignment_strategy" {
		t.Errorf("Expected assignment strategy 'every_member_assignment_strategy', got '%s'",
			weekdays.Events[0].AssignmentStrategy.Type)
	}
	if len(weekdays.Events[0].AssignmentStrategy.Members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(weekdays.Events[0].AssignmentStrategy.Members))
	}

	// Check second rotation
	weekends := schedule.Rotations[1]
	if weekends.Name != "Weekends" {
		t.Errorf("Expected rotation name 'Weekends', got '%s'", weekends.Name)
	}
	if len(weekends.Events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(weekends.Events))
	}
	if weekends.Events[0].AssignmentStrategy.Type != "rotating_member_assignment_strategy" {
		t.Errorf("Expected assignment strategy 'rotating_member_assignment_strategy', got '%s'",
			weekends.Events[0].AssignmentStrategy.Type)
	}
	if weekends.Events[0].AssignmentStrategy.ShiftsPerMember != 1 {
		t.Errorf("Expected shifts_per_member 1, got %d", weekends.Events[0].AssignmentStrategy.ShiftsPerMember)
	}

	// Check final schedule
	if schedule.FinalSchedule == nil {
		t.Fatal("Expected final_schedule to be set")
	}
	if schedule.FinalSchedule.RenderedCoveragePercentage != 93.6 {
		t.Errorf("Expected coverage 93.6, got %.1f", schedule.FinalSchedule.RenderedCoveragePercentage)
	}
	if len(schedule.FinalSchedule.ComputedShiftAssignments) == 0 {
		t.Error("Expected computed shift assignments to be set")
	}
}

// TestMockFlexibleScheduleServer_FinalScheduleGeneration tests that final_schedule is generated
func TestMockFlexibleScheduleServer_FinalScheduleGeneration(t *testing.T) {
	mock := newMockFlexibleScheduleServer()
	defer mock.Close()

	schedule := createMockScheduleForTest("Test Schedule", "America/Chicago", "PUSERID1")

	reqBody := map[string]interface{}{"schedule": schedule}
	reqJSON, _ := json.Marshal(reqBody)

	resp, err := http.Post(
		fmt.Sprintf("%s/schedules", mock.URL()),
		"application/json",
		strings.NewReader(string(reqJSON)),
	)
	if err != nil {
		t.Fatalf("Failed to create schedule: %v", err)
	}
	defer resp.Body.Close()

	var response map[string]*pagerduty.FlexibleSchedule
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	created := response["schedule"]
	if created.FinalSchedule == nil {
		t.Fatal("Expected final_schedule to be set")
	}
	if created.FinalSchedule.Type != "final_schedule" {
		t.Errorf("Expected type 'final_schedule', got '%s'", created.FinalSchedule.Type)
	}
	if created.FinalSchedule.RenderedCoveragePercentage != 93.6 {
		t.Errorf("Expected coverage 93.6, got %.1f", created.FinalSchedule.RenderedCoveragePercentage)
	}
	if len(created.FinalSchedule.ComputedShiftAssignments) == 0 {
		t.Error("Expected computed shift assignments to be set")
	}

	// Check that computed shift assignments have the expected structure
	if len(created.FinalSchedule.ComputedShiftAssignments) > 0 {
		assignment := created.FinalSchedule.ComputedShiftAssignments[0]
		if assignment.Type != "computed_shift_assignment" {
			t.Errorf("Expected assignment type 'computed_shift_assignment', got '%s'", assignment.Type)
		}
		if assignment.StartTime == "" {
			t.Error("Expected start_time to be set")
		}
		if assignment.EndTime == "" {
			t.Error("Expected end_time to be set")
		}
		if assignment.Member == nil {
			t.Fatal("Expected member to be set")
		}
		if assignment.Source == nil {
			t.Fatal("Expected source to be set")
		}
		if assignment.Member.UserID != "PUSERID1" {
			t.Errorf("Expected member UserID 'PUSERID1', got '%s'", assignment.Member.UserID)
		}
	}
}

// TestMockFlexibleScheduleServer_InvalidJSON tests error handling for invalid JSON
func TestMockFlexibleScheduleServer_InvalidJSON(t *testing.T) {
	mock := newMockFlexibleScheduleServer()
	defer mock.Close()

	resp, err := http.Post(
		fmt.Sprintf("%s/schedules", mock.URL()),
		"application/json",
		strings.NewReader("invalid json"),
	)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, resp.StatusCode)
	}
}
