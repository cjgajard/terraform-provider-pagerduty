package pagerduty

import (
	"context"

	"github.com/google/go-querystring/query"
)

type ListIncidentWorkflowActionsOptions struct {
	Cursor  string `url:"cursor,omitempty"`
	Limit   int    `url:"limit,omitempty"`
	Keyword string `url:"keyword,omitempty"`
}

type ListIncidentWorkflowActionsResponse struct {
	APIListObject
	IncidentWorkflowActions []IncidentWorkflowAction `json:"actions,omitempty"`
	Limit                   int                      `json:"limit,omitempty"`
	NextCursor              string                   `json:"next_cursor,omitempty"`
	More                    bool                     `json:"more,omitempty"`
}

type ListIncidentWorkflowActionResponse struct {
	IncidentWorkflowActions []IncidentWorkflowAction
	Limit                   int
	NextCursor              string
	More                    bool
}

func (c *Client) ListIncidentWorkflowActions(ctx context.Context, options ListIncidentWorkflowActionsOptions) (*ListIncidentWorkflowActionsResponse, error) {
	v, err := query.Values(options)
	if err != nil {
		return nil, err
	}

	resp, err := c.get(ctx, "/incident_workflows/actions?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var result ListIncidentWorkflowActionsResponse
	if err = c.decodeJSON(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

type IncidentWorkflowAction struct {
	ID              string                         `json:"id,omitempty"`
	Type            string                         `json:"type,omitempty"`
	Self            string                         `json:"self,omitempty"`
	ActionType      string                         `json:"action_type,omitempty"`
	CreatedAt       string                         `json:"created_at,omitempty"`
	CreatedByUserID string                         `json:"created_by_user_id,omitempty"`
	Description     string                         `json:"description,omitempty"`
	DomainName      string                         `json:"domain_name,omitempty"`
	FunctionName    string                         `json:"function_name,omitempty"`
	Inputs          []IncidentWorkflowActionInput  `json:"inputs,omitempty"`
	Metadata        string                         `json:"metadata,omitempty"`
	Name            string                         `json:"name,omitempty"`
	Outputs         []IncidentWorkflowActionOutput `json:"outputs,omitempty"`
	PackageName     string                         `json:"package_name,omitempty"`
	SearchKeywords  []string                       `json:"search_keywords,omitempty"`
	Tags            []string                       `json:"tags,omitempty"`
	Version         float64                        `json:"version,omitempty"`
}

type IncidentWorkflowActionInput struct {
	Name        string `json:"name,omitempty"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Default     string `json:"default,omitempty"`
}

type IncidentWorkflowActionOutput struct {
	Name        string `json:"name,omitempty"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}
