package pagerduty

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/PagerDuty/go-pagerduty"
)

// mockFlexibleScheduleServer creates a mock HTTP server that simulates the PagerDuty Flexible Schedule API
type mockFlexibleScheduleServer struct {
	server    *httptest.Server
	schedules map[string]*pagerduty.FlexibleSchedule
	mutex     sync.RWMutex
	idCounter int
}

// newMockFlexibleScheduleServer creates a new mock server instance
func newMockFlexibleScheduleServer() *mockFlexibleScheduleServer {
	mock := &mockFlexibleScheduleServer{
		schedules: make(map[string]*pagerduty.FlexibleSchedule),
		idCounter: 1,
	}

	mock.server = httptest.NewServer(http.HandlerFunc(mock.handler))
	return mock
}

// Close shuts down the mock server
func (m *mockFlexibleScheduleServer) Close() {
	m.server.Close()
}

// URL returns the mock server's URL
func (m *mockFlexibleScheduleServer) URL() string {
	return m.server.URL
}

// handler routes requests to the appropriate mock handler
func (m *mockFlexibleScheduleServer) handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Route based on method and path
	switch {
	case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/schedules"):
		m.handleCreate(w, r)
	case r.Method == "GET" && strings.Contains(r.URL.Path, "/schedules/"):
		m.handleGet(w, r)
	case r.Method == "PUT" && strings.Contains(r.URL.Path, "/schedules/"):
		m.handleUpdate(w, r)
	case r.Method == "DELETE" && strings.Contains(r.URL.Path, "/schedules/"):
		m.handleDelete(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Not Found"})
	}
}

// handleCreate handles POST /schedules
func (m *mockFlexibleScheduleServer) handleCreate(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Schedule pagerduty.FlexibleSchedule `json:"schedule"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Generate ID for the new schedule
	scheduleID := fmt.Sprintf("P%d", m.idCounter)
	m.idCounter++

	schedule := request.Schedule
	schedule.ID = scheduleID
	schedule.Type = "schedule"

	// Assign IDs to rotations and events
	for i := range schedule.Rotations {
		schedule.Rotations[i].ID = fmt.Sprintf("ROT%d%d", m.idCounter, i)
		schedule.Rotations[i].Type = "schedule_rotation"

		for j := range schedule.Rotations[i].Events {
			schedule.Rotations[i].Events[j].ID = fmt.Sprintf("EVT%d%d%d", m.idCounter, i, j)
			schedule.Rotations[i].Events[j].Type = "schedule_event"
		}
	}

	// Generate mock final schedule
	schedule.FinalSchedule = m.generateMockFinalSchedule(&schedule)

	m.schedules[scheduleID] = &schedule

	response := map[string]interface{}{
		"schedule": schedule,
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleGet handles GET /schedules/{id}
func (m *mockFlexibleScheduleServer) handleGet(w http.ResponseWriter, r *http.Request) {
	// Extract schedule ID from path
	parts := strings.Split(r.URL.Path, "/")
	scheduleID := parts[len(parts)-1]

	m.mutex.RLock()
	schedule, exists := m.schedules[scheduleID]
	m.mutex.RUnlock()

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Schedule not found"})
		return
	}

	response := map[string]interface{}{
		"schedule": schedule,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleUpdate handles PUT /schedules/{id}
func (m *mockFlexibleScheduleServer) handleUpdate(w http.ResponseWriter, r *http.Request) {
	// Extract schedule ID from path
	parts := strings.Split(r.URL.Path, "/")
	scheduleID := parts[len(parts)-1]

	var request struct {
		Schedule pagerduty.FlexibleSchedule `json:"schedule"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	existingSchedule, exists := m.schedules[scheduleID]
	if !exists {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Schedule not found"})
		return
	}

	// Update the schedule, preserving ID and generated IDs
	schedule := request.Schedule
	schedule.ID = scheduleID
	schedule.Type = "schedule"

	// Preserve or assign rotation/event IDs
	for i := range schedule.Rotations {
		if i < len(existingSchedule.Rotations) {
			schedule.Rotations[i].ID = existingSchedule.Rotations[i].ID
		} else {
			schedule.Rotations[i].ID = fmt.Sprintf("ROT%d%d", m.idCounter, i)
		}
		schedule.Rotations[i].Type = "schedule_rotation"

		for j := range schedule.Rotations[i].Events {
			if i < len(existingSchedule.Rotations) && j < len(existingSchedule.Rotations[i].Events) {
				schedule.Rotations[i].Events[j].ID = existingSchedule.Rotations[i].Events[j].ID
			} else {
				schedule.Rotations[i].Events[j].ID = fmt.Sprintf("EVT%d%d%d", m.idCounter, i, j)
			}
			schedule.Rotations[i].Events[j].Type = "schedule_event"
		}
	}

	// Regenerate final schedule
	schedule.FinalSchedule = m.generateMockFinalSchedule(&schedule)

	m.schedules[scheduleID] = &schedule

	response := map[string]interface{}{
		"schedule": schedule,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleDelete handles DELETE /schedules/{id}
func (m *mockFlexibleScheduleServer) handleDelete(w http.ResponseWriter, r *http.Request) {
	// Extract schedule ID from path
	parts := strings.Split(r.URL.Path, "/")
	scheduleID := parts[len(parts)-1]

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.schedules[scheduleID]; !exists {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Schedule not found"})
		return
	}

	delete(m.schedules, scheduleID)

	w.WriteHeader(http.StatusNoContent)
}

// generateMockFinalSchedule creates a mock final_schedule based on the flexible_schedule_response.json structure
func (m *mockFlexibleScheduleServer) generateMockFinalSchedule(schedule *pagerduty.FlexibleSchedule) *pagerduty.FinalSchedule {
	finalSchedule := &pagerduty.FinalSchedule{
		Type:                       "final_schedule",
		RenderedCoveragePercentage: 93.6, // Mock coverage percentage
		ComputedShiftAssignments:   []pagerduty.ComputedShiftAssignment{},
	}

	// Generate mock shift assignments based on rotations
	now := time.Now()
	assignmentID := 1

	for _, rotation := range schedule.Rotations {
		for _, event := range rotation.Events {
			if event.AssignmentStrategy == nil || len(event.AssignmentStrategy.Members) == 0 {
				continue
			}

			// Create a few mock shift assignments for each event
			for i := 0; i < 3; i++ {
				memberIdx := i % len(event.AssignmentStrategy.Members)
				member := event.AssignmentStrategy.Members[memberIdx]

				assignment := pagerduty.ComputedShiftAssignment{
					Type:      "computed_shift_assignment",
					StartTime: now.Add(time.Duration(i*24) * time.Hour).Format(time.RFC3339),
					EndTime:   now.Add(time.Duration((i+1)*24) * time.Hour).Format(time.RFC3339),
					Member: &pagerduty.ShiftMember{
						Type:   member.Type,
						UserID: member.UserID,
						Name:   member.Name,
						Color:  member.Color,
					},
					Source: &pagerduty.ComputedShiftAssignmentSource{
						Type:              "schedule_rotation_shift",
						RotationID:        rotation.ID,
						ShiftID:           fmt.Sprintf("SHIFT%d", assignmentID),
						ShiftAssignmentID: fmt.Sprintf("ASSIGN%d", assignmentID),
					},
				}

				finalSchedule.ComputedShiftAssignments = append(finalSchedule.ComputedShiftAssignments, assignment)
				assignmentID++
			}

			// Only generate a few assignments to keep response reasonable
			if len(finalSchedule.ComputedShiftAssignments) >= 10 {
				break
			}
		}

		if len(finalSchedule.ComputedShiftAssignments) >= 10 {
			break
		}
	}

	return finalSchedule
}

// generateMockScheduleFromJSON creates a mock schedule based on the flexible_schedule_response.json structure
func generateMockScheduleFromJSON() *pagerduty.FlexibleSchedule {
	return &pagerduty.FlexibleSchedule{
		APIObject: pagerduty.APIObject{
			ID:   "P65784",
			Type: "schedule",
		},
		Name:        "Support Team Schedule",
		Description: "Schedule for the support team covering weekdays and weekends.",
		TimeZone:    "America/Chicago",
		Rotations: []pagerduty.ScheduleRotation{
			{
				APIObject: pagerduty.APIObject{
					ID:   "AGL7U3W2IR53BMSLIC2L7F7IQM",
					Type: "schedule_rotation",
				},
				Name: "Week Days",
				Events: []pagerduty.ScheduleEvent{
					{
						APIObject: pagerduty.APIObject{
							ID:   "AGL7U3ZVM54URKAA4VBZ4USV3I",
							Type: "schedule_event",
						},
						StartTime: &pagerduty.ZonedDateTime{
							DateTime: "2025-07-14T08:00:00-05:00",
							TimeZone: "America/Chicago",
						},
						EndTime: &pagerduty.ZonedDateTime{
							DateTime: "2025-07-18T17:00:00-05:00",
							TimeZone: "America/Chicago",
						},
						EffectiveSince: "2025-07-09T12:00:00-05:00",
						EffectiveUntil: nil,
						Recurrence:     []string{"RRULE:FREQ=WEEKLY"},
						AssignmentStrategy: &pagerduty.AssignmentStrategy{
							Type: "every_member_assignment_strategy",
							Members: []pagerduty.ShiftMember{
								{
									Type:   "user_member",
									UserID: "P01234",
									Color:  "yellow",
									Name:   "sara",
								},
								{
									Type:   "user_member",
									UserID: "P98765",
									Color:  "green",
									Name:   "jill",
								},
							},
						},
					},
				},
			},
			{
				APIObject: pagerduty.APIObject{
					ID:   "AGL7U3W2IR53BMSLIC2L7F7IQN",
					Type: "schedule_rotation",
				},
				Name: "Weekends",
				Events: []pagerduty.ScheduleEvent{
					{
						APIObject: pagerduty.APIObject{
							ID:   "AGL7U4CAENZMTHDYP6XUOLIFTQ",
							Type: "schedule_event",
						},
						StartTime: &pagerduty.ZonedDateTime{
							DateTime: "2025-07-18T17:00:00-05:00",
							TimeZone: "America/Chicago",
						},
						EndTime: &pagerduty.ZonedDateTime{
							DateTime: "2025-07-21T08:00:00-05:00",
							TimeZone: "America/Chicago",
						},
						EffectiveSince: "2025-07-09T12:00:00-05:00",
						EffectiveUntil: stringPtr("2025-08-01T12:00:00-05:00"),
						Recurrence:     []string{"RRULE:FREQ=WEEKLY"},
						AssignmentStrategy: &pagerduty.AssignmentStrategy{
							Type:            "rotating_member_assignment_strategy",
							ShiftsPerMember: 1,
							Members: []pagerduty.ShiftMember{
								{
									Type:   "user_member",
									UserID: "P98765",
									Color:  "green",
									Name:   "jill",
								},
								{
									Type:   "user_member",
									UserID: "P01234",
									Color:  "yellow",
									Name:   "sara",
								},
							},
						},
					},
					{
						APIObject: pagerduty.APIObject{
							ID:   "AGL7U377EZ4EHAYRTQK5CES4DU",
							Type: "schedule_event",
						},
						StartTime: &pagerduty.ZonedDateTime{
							DateTime: "2025-07-18T17:00:00-05:00",
							TimeZone: "America/Chicago",
						},
						EndTime: &pagerduty.ZonedDateTime{
							DateTime: "2025-07-21T08:00:00-05:00",
							TimeZone: "America/Chicago",
						},
						EffectiveSince: "2025-08-01T12:00:00-05:00",
						EffectiveUntil: nil,
						Recurrence:     []string{"RRULE:FREQ=WEEKLY"},
						AssignmentStrategy: &pagerduty.AssignmentStrategy{
							Type: "every_member_assignment_strategy",
							Members: []pagerduty.ShiftMember{
								{
									Type:   "user_member",
									UserID: "P99999",
									Color:  "blue",
									Name:   "bob",
								},
							},
						},
					},
				},
			},
		},
		FinalSchedule: &pagerduty.FinalSchedule{
			Type:                       "final_schedule",
			RenderedCoveragePercentage: 93.6,
			ComputedShiftAssignments: []pagerduty.ComputedShiftAssignment{
				{
					Type:      "computed_shift_assignment",
					StartTime: "2025-07-14T08:00:00-05:00",
					EndTime:   "2025-07-14T17:00:00-05:00",
					Member: &pagerduty.ShiftMember{
						Type:   "user_member",
						UserID: "P77777",
						Color:  "purple",
						Name:   "dan",
					},
					Source: &pagerduty.ComputedShiftAssignmentSource{
						Type:              "schedule_rotation_shift_coverage",
						ShiftID:           "AGL7U34AOV5LDNOUYOV74BB6CU",
						RotationID:        "AGL7U3W2IR53BMSLIC2L7F7IQM",
						ShiftAssignmentID: "AGL7VFUA7Z4WNABOOBBDJNOHRY",
						CoverageID:        "AGL7VKTGVVYQDPOT6RKVCVT4UY",
					},
				},
				{
					Type:      "computed_shift_assignment",
					StartTime: "2025-07-14T17:00:00-05:00",
					EndTime:   "2025-07-15T08:00:00-05:00",
					Member: &pagerduty.ShiftMember{
						Type:   "user_member",
						UserID: "P01234",
						Color:  "yellow",
						Name:   "sara",
					},
					Source: &pagerduty.ComputedShiftAssignmentSource{
						Type:              "schedule_rotation_shift",
						ShiftID:           "AGL7U34AOV5LDNOUYOV74BB6CU",
						RotationID:        "AGL7U3W2IR53BMSLIC2L7F7IQM",
						ShiftAssignmentID: "AGL7VFUA7Z4WNABOOBBDJNOHRY",
					},
				},
				{
					Type:      "computed_shift_assignment",
					StartTime: "2025-07-18T17:00:00-05:00",
					EndTime:   "2025-07-21T08:00:00-05:00",
					Member: &pagerduty.ShiftMember{
						Type:   "user_member",
						UserID: "P98765",
						Color:  "green",
						Name:   "jill",
					},
					Source: &pagerduty.ComputedShiftAssignmentSource{
						Type:              "schedule_rotation_shift",
						ShiftID:           "AGL7U4EAFBYGJNLLXBJAA4QTUV",
						RotationID:        "AGL7U3W2IR53BMSLIC2L7F7IQN",
						ShiftAssignmentID: "AGL7VICVXR2BFP3JWLIODUX76Y",
					},
				},
			},
		},
		Teams: []pagerduty.APIObject{},
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

// MockScheduleTestCase represents a test case configuration for mock testing
type MockScheduleTestCase struct {
	Name            string
	InitialSchedule *pagerduty.FlexibleSchedule
	ExpectedError   bool
	ValidateFunc    func(*pagerduty.FlexibleSchedule) error
}

// createMockScheduleForTest creates a simple mock schedule for testing
func createMockScheduleForTest(name, timezone string, userID string) *pagerduty.FlexibleSchedule {
	return &pagerduty.FlexibleSchedule{
		APIObject: pagerduty.APIObject{
			Type: "schedule",
		},
		Name:        name,
		Description: "Test schedule",
		TimeZone:    timezone,
		Rotations: []pagerduty.ScheduleRotation{
			{
				APIObject: pagerduty.APIObject{
					Type: "schedule_rotation",
				},
				Name: "Test Rotation",
				Events: []pagerduty.ScheduleEvent{
					{
						APIObject: pagerduty.APIObject{
							Type: "schedule_event",
						},
						StartTime: &pagerduty.ZonedDateTime{
							DateTime: "2025-07-14T08:00:00-05:00",
							TimeZone: timezone,
						},
						EndTime: &pagerduty.ZonedDateTime{
							DateTime: "2025-07-18T17:00:00-05:00",
							TimeZone: timezone,
						},
						EffectiveSince: "2025-07-09T12:00:00-05:00",
						Recurrence:     []string{"RRULE:FREQ=WEEKLY"},
						AssignmentStrategy: &pagerduty.AssignmentStrategy{
							Type: "every_member_assignment_strategy",
							Members: []pagerduty.ShiftMember{
								{
									Type:   "user_member",
									UserID: userID,
									Name:   "Test User",
									Color:  "blue",
								},
							},
						},
					},
				},
			},
		},
	}
}
