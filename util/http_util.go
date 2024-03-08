package util

import (
	"errors"
	"net/http"
	"regexp"

	"github.com/PagerDuty/go-pagerduty"
)

func IsStatusCodeError(err error, errCode int) bool {
	var apiErr pagerduty.APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == errCode
	}
	return false
}

func IsBadRequestError(err error) bool {
	return IsStatusCodeError(err, http.StatusBadRequest)
}

var notFoundErrorRegexp = regexp.MustCompile(".*: 404 Not Found$")

func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var apiErr pagerduty.APIError
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode == http.StatusNotFound {
			return true
		}
	}

	// There are some errors that doesn't stick to expected error interface
	// and fallback to a simple text error message that can be capture by
	// this regexp.
	return notFoundErrorRegexp.MatchString(err.Error())
}
