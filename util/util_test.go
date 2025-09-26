package util

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"testing"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

func TestValidateTZValueDiagFunc(t *testing.T) {
	notValidTZ1 := "not a valid TZ"

	cases := []struct {
		given string
		want  diag.Diagnostics
		path  cty.Path
	}{
		{
			given: "America/Montevideo",
			want:  nil,
			path:  cty.Path{},
		},
		{
			given: "America/Indiana/Indianapolis",
			want:  nil,
			path:  cty.Path{},
		},
		{
			given: notValidTZ1,
			want:  diag.Diagnostics{diag.Diagnostic{Severity: 0, Summary: fmt.Sprintf("\"%s\" is a not valid input. Please refer to the list of allowed Time Zone values at https://developer.pagerduty.com/docs/1afe25e9c94cb-types#time-zone", notValidTZ1), Detail: "", AttributePath: cty.Path{cty.GetAttrStep{Name: "time_zone"}}}},
			path:  cty.Path{cty.GetAttrStep{Name: "time_zone"}},
		},
	}

	for _, c := range cases {
		got := ValidateTZValueDiagFunc(c.given, c.path)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("want %v; got %v", c.want, got)
		}
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-API error",
			err:      errors.New("some other error"),
			expected: false,
		},
		{
			name: "401 Unauthorized API error",
			err: pagerduty.APIError{
				StatusCode: http.StatusUnauthorized,
				Message:    "Unauthorized",
			},
			expected: true,
		},
		{
			name: "403 Forbidden API error",
			err: pagerduty.APIError{
				StatusCode: http.StatusForbidden,
				Message:    "Forbidden",
			},
			expected: true,
		},
		{
			name: "400 Bad Request API error",
			err: pagerduty.APIError{
				StatusCode: http.StatusBadRequest,
				Message:    "Bad Request",
			},
			expected: false,
		},
		{
			name: "404 Not Found API error",
			err: pagerduty.APIError{
				StatusCode: http.StatusNotFound,
				Message:    "Not Found",
			},
			expected: false,
		},
		{
			name: "500 Internal Server Error API error",
			err: pagerduty.APIError{
				StatusCode: http.StatusInternalServerError,
				Message:    "Internal Server Error",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsAuthError(tt.err)
			if result != tt.expected {
				t.Errorf("IsAuthError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}
