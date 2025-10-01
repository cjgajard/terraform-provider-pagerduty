package pagerduty

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-querystring/query"
)

// Restriction limits on-call responsibility for a layer to certain times of the day or week.
type Restriction struct {
	Type            string `json:"type,omitempty"`
	StartTimeOfDay  string `json:"start_time_of_day,omitempty"`
	StartDayOfWeek  uint   `json:"start_day_of_week,omitempty"`
	DurationSeconds uint   `json:"duration_seconds,omitempty"`
}

// RenderedScheduleEntry represents the computed set of schedule layer entries that put users on call for a schedule, and cannot be modified directly.
type RenderedScheduleEntry struct {
	Start string    `json:"start,omitempty"`
	End   string    `json:"end,omitempty"`
	User  APIObject `json:"user,omitempty"`
}

// ScheduleLayer is an entry that puts users on call for a schedule.
type ScheduleLayer struct {
	APIObject
	Name                       string                  `json:"name,omitempty"`
	Start                      string                  `json:"start,omitempty"`
	End                        string                  `json:"end,omitempty"`
	RotationVirtualStart       string                  `json:"rotation_virtual_start,omitempty"`
	RotationTurnLengthSeconds  uint                    `json:"rotation_turn_length_seconds,omitempty"`
	Users                      []UserReference         `json:"users,omitempty"`
	Restrictions               []Restriction           `json:"restrictions,omitempty"`
	RenderedScheduleEntries    []RenderedScheduleEntry `json:"rendered_schedule_entries,omitempty"`
	RenderedCoveragePercentage float64                 `json:"rendered_coverage_percentage,omitempty"`
}

// Schedule determines the time periods that users are on call.
type Schedule struct {
	APIObject
	Name                string          `json:"name,omitempty"`
	TimeZone            string          `json:"time_zone,omitempty"`
	Description         string          `json:"description,omitempty"`
	EscalationPolicies  []APIObject     `json:"escalation_policies,omitempty"`
	Users               []APIObject     `json:"users,omitempty"`
	Teams               []APIObject     `json:"teams,omitempty"`
	ScheduleLayers      []ScheduleLayer `json:"schedule_layers,omitempty"`
	OverrideSubschedule ScheduleLayer   `json:"override_subschedule,omitempty"`
	FinalSchedule       ScheduleLayer   `json:"final_schedule,omitempty"`
}

// ListSchedulesOptions is the data structure used when calling the ListSchedules API endpoint.
type ListSchedulesOptions struct {
	// Limit is the pagination parameter that limits the number of results per
	// page. PagerDuty defaults this value to 25 if omitted, and sets an upper
	// bound of 100.
	Limit uint `url:"limit,omitempty"`

	// Offset is the pagination parameter that specifies the offset at which to
	// start pagination results. When trying to request the next page of
	// results, the new Offset value should be currentOffset + Limit.
	Offset uint `url:"offset,omitempty"`

	// Total is the pagination parameter to request that the API return the
	// total count of items in the response. If this field is omitted or set to
	// false, the total number of results will not be sent back from the PagerDuty API.
	//
	// Setting this to true will slow down the API response times, and so it's
	// recommended to omit it unless you've a specific reason for wanting the
	// total count of items in the collection.
	Total bool `url:"total,omitempty"`

	Query    string   `url:"query,omitempty"`
	Includes []string `url:"include,omitempty,brackets"`
}

// ListSchedulesResponse is the data structure returned from calling the ListSchedules API endpoint.
type ListSchedulesResponse struct {
	APIListObject
	Schedules []Schedule `json:"schedules"`
}

// UserReference is a reference to an authorized PagerDuty user.
type UserReference struct {
	User APIObject `json:"user"`
}

// ListSchedules lists the on-call schedules.
//
// Deprecated: Use ListSchedulesWithContext instead.
func (c *Client) ListSchedules(o ListSchedulesOptions) (*ListSchedulesResponse, error) {
	return c.ListSchedulesWithContext(context.Background(), o)
}

// ListSchedulesWithContext lists the on-call schedules.
func (c *Client) ListSchedulesWithContext(ctx context.Context, o ListSchedulesOptions) (*ListSchedulesResponse, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, "/schedules?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var result ListSchedulesResponse
	if err = c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CreateSchedule creates a new on-call schedule.
//
// Deprecated: Use CreateScheduleWithContext instead.
func (c *Client) CreateSchedule(s Schedule) (*Schedule, error) {
	return c.CreateScheduleWithContext(context.Background(), s)
}

// CreateScheduleWithContext creates a new on-call schedule.
func (c *Client) CreateScheduleWithContext(ctx context.Context, s Schedule) (*Schedule, error) {
	d := map[string]Schedule{
		"schedule": s,
	}

	resp, err := c.post(ctx, "/schedules", d, nil)
	return getScheduleFromResponse(c, resp, err)
}

// PreviewScheduleOptions is the data structure used when calling the PreviewSchedule API endpoint.
type PreviewScheduleOptions struct {
	Since    string `url:"since,omitempty"`
	Until    string `url:"until,omitempty"`
	Overflow bool   `url:"overflow,omitempty"`
}

// PreviewSchedule previews what an on-call schedule would look like without
// saving it.
//
// Deprecated: Use PreviewScheduleWithContext instead.
func (c *Client) PreviewSchedule(s Schedule, o PreviewScheduleOptions) error {
	return c.PreviewScheduleWithContext(context.Background(), s, o)
}

// PreviewScheduleWithContext previews what an on-call schedule would look like
// without saving it. Nothing is returned from this method, because the API
// should return the Schedule as we posted it. If this method call returns no
// error, the schedule should be valid and can be updated.
func (c *Client) PreviewScheduleWithContext(ctx context.Context, s Schedule, o PreviewScheduleOptions) error {
	v, err := query.Values(o)
	if err != nil {
		return err
	}

	d := map[string]Schedule{
		"schedule": s,
	}

	_, err = c.post(ctx, "/schedules/preview?"+v.Encode(), d, nil)
	return err
}

// DeleteSchedule deletes an on-call schedule.
//
// Deprecated: Use DeleteScheduleWithContext instead.
func (c *Client) DeleteSchedule(id string) error {
	return c.DeleteScheduleWithContext(context.Background(), id)
}

// DeleteScheduleWithContext deletes an on-call schedule.
func (c *Client) DeleteScheduleWithContext(ctx context.Context, id string) error {
	_, err := c.delete(ctx, "/schedules/"+id)
	return err
}

// GetScheduleOptions is the data structure used when calling the GetSchedule API endpoint.
type GetScheduleOptions struct {
	TimeZone string `url:"time_zone,omitempty"`
	Since    string `url:"since,omitempty"`
	Until    string `url:"until,omitempty"`
}

// GetSchedule shows detailed information about a schedule, including entries
// for each layer and sub-schedule.
//
// Deprecated: Use GetScheduleWithContext instead.
func (c *Client) GetSchedule(id string, o GetScheduleOptions) (*Schedule, error) {
	return c.GetScheduleWithContext(context.Background(), id, o)
}

// GetScheduleWithContext shows detailed information about a schedule, including
// entries for each layer and sub-schedule.
func (c *Client) GetScheduleWithContext(ctx context.Context, id string, o GetScheduleOptions) (*Schedule, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, fmt.Errorf("Could not parse values for query: %v", err)
	}

	resp, err := c.get(ctx, "/schedules/"+id+"?"+v.Encode(), nil)
	return getScheduleFromResponse(c, resp, err)
}

// UpdateScheduleOptions is the data structure used when calling the UpdateSchedule API endpoint.
type UpdateScheduleOptions struct {
	Overflow bool `url:"overflow,omitempty"`
}

// UpdateSchedule updates an existing on-call schedule.
//
// Deprecated: Use UpdateScheduleWithContext instead.
func (c *Client) UpdateSchedule(id string, s Schedule) (*Schedule, error) {
	return c.UpdateScheduleWithContext(context.Background(), id, s)
}

// UpdateScheduleWithContext updates an existing on-call schedule.
func (c *Client) UpdateScheduleWithContext(ctx context.Context, id string, s Schedule) (*Schedule, error) {
	d := map[string]Schedule{
		"schedule": s,
	}

	resp, err := c.put(ctx, "/schedules/"+id, d, nil)
	return getScheduleFromResponse(c, resp, err)
}

// ListOverridesOptions is the data structure used when calling the ListOverrides API endpoint.
type ListOverridesOptions struct {
	Since    string `url:"since,omitempty"`
	Until    string `url:"until,omitempty"`
	Editable bool   `url:"editable,omitempty"`
	Overflow bool   `url:"overflow,omitempty"`
}

// ListOverridesResponse is the data structure returned from calling the ListOverrides API endpoint.
type ListOverridesResponse struct {
	Overrides []Override `json:"overrides,omitempty"`
}

// Override are any schedule layers from the override layer.
type Override struct {
	ID      string    `json:"id,omitempty"`
	Type    string    `json:"type,omitempty"`
	Summary string    `json:"summary,omitempty"`
	Self    string    `json:"self,omitempty"`
	HTMLURL string    `json:"html_url,omitempty"`
	Start   string    `json:"start,omitempty"`
	End     string    `json:"end,omitempty"`
	User    APIObject `json:"user,omitempty"`
}

// ListOverrides lists overrides for a given time range.
//
// Deprecated: Use ListOverridesWithContext instead.
func (c *Client) ListOverrides(id string, o ListOverridesOptions) (*ListOverridesResponse, error) {
	return c.ListOverridesWithContext(context.Background(), id, o)
}

// ListOverridesWithContext lists overrides for a given time range.
func (c *Client) ListOverridesWithContext(ctx context.Context, id string, o ListOverridesOptions) (*ListOverridesResponse, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, "/schedules/"+id+"/overrides?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var result ListOverridesResponse
	if err = c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// CreateOverride creates an override for a specific user covering the specified
// time range.
//
// Deprecated: Use CreateOverrideWithContext instead.
func (c *Client) CreateOverride(id string, o Override) (*Override, error) {
	return c.CreateOverrideWithContext(context.Background(), id, o)
}

// CreateOverrideWithContext creates an override for a specific user covering
// the specified time range.
func (c *Client) CreateOverrideWithContext(ctx context.Context, id string, o Override) (*Override, error) {
	d := map[string]Override{
		"override": o,
	}

	resp, err := c.post(ctx, "/schedules/"+id+"/overrides", d, nil)
	if err != nil {
		return nil, err
	}

	return getOverrideFromResponse(c, resp)
}

// DeleteOverride removes an override.
//
// Deprecated: Use DeleteOverrideWithContext instead.
func (c *Client) DeleteOverride(scheduleID, overrideID string) error {
	return c.DeleteOverrideWithContext(context.Background(), scheduleID, overrideID)
}

// DeleteOverrideWithContext removes an override.
func (c *Client) DeleteOverrideWithContext(ctx context.Context, scheduleID, overrideID string) error {
	_, err := c.delete(ctx, "/schedules/"+scheduleID+"/overrides/"+overrideID)
	return err
}

// ListOnCallUsersOptions is the data structure used when calling the ListOnCallUsers API endpoint.
type ListOnCallUsersOptions struct {
	Since string `url:"since,omitempty"`
	Until string `url:"until,omitempty"`
}

// ListOnCallUsers lists all of the users on call in a given schedule for a
// given time range.
//
// Deprecated: Use ListOnCallUsersWithContext instead.
func (c *Client) ListOnCallUsers(id string, o ListOnCallUsersOptions) ([]User, error) {
	return c.ListOnCallUsersWithContext(context.Background(), id, o)
}

// ListOnCallUsersWithContext lists all of the users on call in a given schedule
// for a given time range.
func (c *Client) ListOnCallUsersWithContext(ctx context.Context, id string, o ListOnCallUsersOptions) ([]User, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, "/schedules/"+id+"/users?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var result map[string][]User
	if err := c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}

	u, ok := result["users"]
	if !ok {
		return nil, fmt.Errorf("JSON response does not have users field")
	}

	return u, nil
}

func getScheduleFromResponse(c *Client, resp *http.Response, err error) (*Schedule, error) {
	if err != nil {
		return nil, err
	}

	var target map[string]Schedule
	if dErr := c.decodeJSON(resp, &target); dErr != nil {
		return nil, fmt.Errorf("Could not decode JSON response: %v", dErr)
	}

	const rootNode = "schedule"

	t, nodeOK := target[rootNode]
	if !nodeOK {
		return nil, fmt.Errorf("JSON response does not have %s field", rootNode)
	}

	return &t, nil
}

func getOverrideFromResponse(c *Client, resp *http.Response) (*Override, error) {
	var target map[string]Override
	if dErr := c.decodeJSON(resp, &target); dErr != nil {
		return nil, fmt.Errorf("Could not decode JSON response: %v", dErr)
	}

	const rootNode = "override"
	o, nodeOK := target[rootNode]
	if !nodeOK {
		return nil, fmt.Errorf("JSON response does not have %s field", rootNode)
	}

	return &o, nil
}

// FlexibleSchedule represents the new v2 flexible schedule structure
type FlexibleSchedule struct {
	APIObject
	Name          string               `json:"name,omitempty"`
	Description   string               `json:"description,omitempty"`
	TimeZone      string               `json:"time_zone,omitempty"`
	Rotations     []ScheduleRotation   `json:"rotations,omitempty"`
	FinalSchedule *FinalSchedule       `json:"final_schedule,omitempty"`
	Teams         []APIObject          `json:"teams,omitempty"`
}

// ScheduleRotation represents a rotation in the flexible schedule
type ScheduleRotation struct {
	APIObject
	Name   string          `json:"name,omitempty"`
	Events []ScheduleEvent `json:"events,omitempty"`
}

// ScheduleEvent represents an event within a rotation
type ScheduleEvent struct {
	APIObject
	StartTime          *ZonedDateTime      `json:"start_time,omitempty"`
	EndTime            *ZonedDateTime      `json:"end_time,omitempty"`
	EffectiveSince     string              `json:"effective_since,omitempty"`
	EffectiveUntil     *string             `json:"effective_until,omitempty"`
	Recurrence         []string            `json:"recurrence,omitempty"`
	AssignmentStrategy *AssignmentStrategy `json:"assignment_strategy,omitempty"`
}

// ZonedDateTime represents a date-time with timezone
type ZonedDateTime struct {
	DateTime string `json:"date_time,omitempty"`
	TimeZone string `json:"time_zone,omitempty"`
}

// AssignmentStrategy represents how members are assigned to shifts
type AssignmentStrategy struct {
	Type            string        `json:"type,omitempty"`
	ShiftsPerMember int           `json:"shifts_per_member,omitempty"`
	Members         []ShiftMember `json:"members,omitempty"`
}

// ShiftMember represents a member in an assignment strategy
type ShiftMember struct {
	Type   string `json:"type,omitempty"`
	UserID string `json:"user_id,omitempty"`
	Name   string `json:"name,omitempty"`
	Color  string `json:"color,omitempty"`
}

// FinalSchedule represents the computed final schedule
type FinalSchedule struct {
	Type                       string                    `json:"type,omitempty"`
	RenderedCoveragePercentage float64                   `json:"rendered_coverage_percentage,omitempty"`
	ComputedShiftAssignments   []ComputedShiftAssignment `json:"computed_shift_assignments,omitempty"`
}

// ComputedShiftAssignment represents a computed shift assignment
type ComputedShiftAssignment struct {
	Type      string                         `json:"type,omitempty"`
	StartTime string                         `json:"start_time,omitempty"`
	EndTime   string                         `json:"end_time,omitempty"`
	Member    *ShiftMember                   `json:"member,omitempty"`
	Source    *ComputedShiftAssignmentSource `json:"source,omitempty"`
}

// ComputedShiftAssignmentSource represents the source of a computed shift assignment
type ComputedShiftAssignmentSource struct {
	Type              string `json:"type,omitempty"`
	RotationID        string `json:"rotation_id,omitempty"`
	ShiftID           string `json:"shift_id,omitempty"`
	ShiftAssignmentID string `json:"shift_assignment_id,omitempty"`
	CoverageID        string `json:"coverage_id,omitempty"`
}

// GetFlexibleScheduleOptions is the data structure used when calling the GetFlexibleSchedule API endpoint.
type GetFlexibleScheduleOptions struct {
	TimeZone string   `url:"time_zone,omitempty"`
	Since    string   `url:"since,omitempty"`
	Until    string   `url:"until,omitempty"`
	Includes []string `url:"include,omitempty,brackets"`
}

// CreateFlexibleScheduleOptions is the data structure used when calling the CreateFlexibleSchedule API endpoint.
type CreateFlexibleScheduleOptions struct {
	Overflow bool `url:"overflow,omitempty"`
}

// UpdateFlexibleScheduleOptions is the data structure used when calling the UpdateFlexibleSchedule API endpoint.
type UpdateFlexibleScheduleOptions struct {
	Overflow bool `url:"overflow,omitempty"`
}

// CreateFlexibleSchedule creates a new flexible schedule using the v2 API.
// This is a mock-up implementation for the new flexible schedule API.
func (c *Client) CreateFlexibleSchedule(s FlexibleSchedule) (*FlexibleSchedule, error) {
	return c.CreateFlexibleScheduleWithContext(context.Background(), s)
}

// CreateFlexibleScheduleWithContext creates a new flexible schedule using the v2 API.
// This is a mock-up implementation for the new flexible schedule API.
func (c *Client) CreateFlexibleScheduleWithContext(ctx context.Context, s FlexibleSchedule) (*FlexibleSchedule, error) {
	d := map[string]FlexibleSchedule{
		"schedule": s,
	}

	// Mock-up: In reality, this would use /api/v2/schedules endpoint
	resp, err := c.post(ctx, "/schedules", d, nil)
	return getFlexibleScheduleFromResponse(c, resp, err)
}

// GetFlexibleSchedule shows detailed information about a flexible schedule.
// This is a mock-up implementation for the new flexible schedule API.
func (c *Client) GetFlexibleSchedule(id string, o GetFlexibleScheduleOptions) (*FlexibleSchedule, error) {
	return c.GetFlexibleScheduleWithContext(context.Background(), id, o)
}

// GetFlexibleScheduleWithContext shows detailed information about a flexible schedule.
// This is a mock-up implementation for the new flexible schedule API.
func (c *Client) GetFlexibleScheduleWithContext(ctx context.Context, id string, o GetFlexibleScheduleOptions) (*FlexibleSchedule, error) {
	v, err := query.Values(o)
	if err != nil {
		return nil, fmt.Errorf("Could not parse values for query: %v", err)
	}

	// Mock-up: In reality, this would use /api/v2/schedules/{id} endpoint
	resp, err := c.get(ctx, "/schedules/"+id+"?"+v.Encode(), nil)
	return getFlexibleScheduleFromResponse(c, resp, err)
}

// UpdateFlexibleSchedule updates an existing flexible schedule.
// This is a mock-up implementation for the new flexible schedule API.
func (c *Client) UpdateFlexibleSchedule(id string, s FlexibleSchedule) (*FlexibleSchedule, error) {
	return c.UpdateFlexibleScheduleWithContext(context.Background(), id, s)
}

// UpdateFlexibleScheduleWithContext updates an existing flexible schedule.
// This is a mock-up implementation for the new flexible schedule API.
func (c *Client) UpdateFlexibleScheduleWithContext(ctx context.Context, id string, s FlexibleSchedule) (*FlexibleSchedule, error) {
	d := map[string]FlexibleSchedule{
		"schedule": s,
	}

	// Mock-up: In reality, this would use /api/v2/schedules/{id} endpoint
	resp, err := c.put(ctx, "/schedules/"+id, d, nil)
	return getFlexibleScheduleFromResponse(c, resp, err)
}

// DeleteFlexibleSchedule deletes a flexible schedule.
// This is a mock-up implementation for the new flexible schedule API.
func (c *Client) DeleteFlexibleSchedule(id string) error {
	return c.DeleteFlexibleScheduleWithContext(context.Background(), id)
}

// DeleteFlexibleScheduleWithContext deletes a flexible schedule.
// This is a mock-up implementation for the new flexible schedule API.
func (c *Client) DeleteFlexibleScheduleWithContext(ctx context.Context, id string) error {
	// Mock-up: In reality, this would use /api/v2/schedules/{id} endpoint
	_, err := c.delete(ctx, "/schedules/"+id)
	return err
}

func getFlexibleScheduleFromResponse(c *Client, resp *http.Response, err error) (*FlexibleSchedule, error) {
	if err != nil {
		return nil, err
	}

	var target map[string]FlexibleSchedule
	if dErr := c.decodeJSON(resp, &target); dErr != nil {
		return nil, fmt.Errorf("Could not decode JSON response: %v", dErr)
	}

	const rootNode = "schedule"

	t, nodeOK := target[rootNode]
	if !nodeOK {
		return nil, fmt.Errorf("JSON response does not have %s field", rootNode)
	}

	return &t, nil
}
