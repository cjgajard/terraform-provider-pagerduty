package util

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

type isFutureTimeValidator struct{}

func (v isFutureTimeValidator) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}

func (v isFutureTimeValidator) MarkdownDescription(_ context.Context) string {
	return "Validates the field is a RFC3339 time string and its value is after system's clock"
}

// func (v isFutureTimeValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
// 	t, err := time.Parse(time.RFC3339, req.ConfigValue.ValueString())
// 	if err != nil {
// 		resp.Diagnostics.AddAttributeError(req.Path, "Not a valid RFC3339 time", err.Error())
// 		return
// 	}

// 	if t.Before(time.Now()) {
// 		resp.Diagnostics.AddAttributeError(req.Path, "Only future maintenance windows are allowed", "")
// 		return
// 	}
// }

func (v isFutureTimeValidator) ValidateResource(_ context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	t, err := time.Parse(time.RFC3339, req.Config.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Not a valid RFC3339 time", err.Error())
		return
	}

	if t.Before(time.Now()) {
		resp.Diagnostics.AddAttributeError(req.Path, "Only future maintenance windows are allowed", "")
		return
	}
}

func ValidateIsFutureMaintenanceWindow() resource.ConfigValidator {
	return isFutureTimeValidator{}
}
