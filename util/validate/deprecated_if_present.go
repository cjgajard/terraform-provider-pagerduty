package validate

import (
	"context"

	"github.com/PagerDuty/terraform-provider-pagerduty/util"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func DeprecatedIfPresent(msg string) *deprecatedIfPresentValidator {
	return &deprecatedIfPresentValidator{
		StringDescriber: util.StringDescriber{Value: "Shows a warning message if the user sets a known and not-empty value"},
		Message:         msg,
	}
}

type deprecatedIfPresentValidator struct {
	util.StringDescriber
	Message string
}

var _ validator.String = (*deprecatedIfPresentValidator)(nil)

func (v *deprecatedIfPresentValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
}

/*
ValidateDiagFunc: func(i interface{}, path cty.Path) diag.Diagnostics {
	v, ok := i.(string)
	if !ok {
		return diag.Diagnostics{
			{
				Severity:      diag.Error,
				Summary:       "Expected String",
				AttributePath: path,
			},
		}
	}

	if v != "" {
		return diag.Diagnostics{
			{
				Severity:      diag.Warning,
				Summary:       "Argument is deprecated. Assignments or updates to this attribute are not supported by Service Integrations API, it is a read-only value. Input support will be dropped in upcomming major release",
				AttributePath: path,
			},
		}
	}
	return diag.Diagnostics{}
},
*/
