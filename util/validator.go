package util

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	v2diag "github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type stringDescriber struct{ s string }

func (d stringDescriber) MarkdownDescription(context.Context) string { return d.s }
func (d stringDescriber) Description(ctx context.Context) string     { return d.MarkdownDescription(ctx) }

type timezoneValidator struct{ stringDescriber }

func ValidateTimezone() validator.String {
	return &timezoneValidator{stringDescriber{"checks time zone is supported by the machine's tzdata"}}
}

func (v timezoneValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() {
		return
	}
	value := req.ConfigValue.ValueString()
	_, err := time.LoadLocation(value)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path, fmt.Sprintf("Timezone %q is invalid", value), err.Error(),
		)
	}
}

type validateIsAllowedString struct {
	validateFn func(s string) bool
	stringDescriber
}

func (v validateIsAllowedString) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if ok := v.validateFn(req.ConfigValue.ValueString()); !ok {
		resp.Diagnostics.AddError(v.stringDescriber.s, "")
	}
}

func IsAllowedStringValidator(mode StringContentValidationMode) validator.String {
	switch mode {
	case NoNonPrintableChars:
		return validateIsAllowedString{
			func(s string) bool {
				for _, char := range s {
					if !unicode.IsPrint(char) {
						return false
					}
				}
				return s != "" && !strings.HasSuffix(s, " ")
			},
			stringDescriber{"Name can not be blank, nor contain non-printable characters. Trailing white spaces are not allowed either."},
		}
	default:
		return validateIsAllowedString{
			func(s string) bool { return false },
			stringDescriber{"Invalid mode while using func IsAllowedStringValidator(mode StringContentValidationMode)"},
		}
	}
}

// ValidateIsAllowedString will always validate if string provided is not empty,
// neither has trailing white spaces. Additionally the string content validation
// will be done based on the `mode` set.
//
//	mode: NoContentValidation | NoNonPrintableChars | NoNonPrintableCharsOrSpecialChars
func ReValidateIsAllowedString(mode StringContentValidationMode) schema.SchemaValidateDiagFunc {
	return func(v interface{}, p cty.Path) v2diag.Diagnostics {
		var diags v2diag.Diagnostics

		fillDiags := func() {
			summary := "Name can not be blank. Trailing white spaces are not allowed either."
			switch mode {
			case NoNonPrintableChars:
				summary = "Name can not be blank, nor contain non-printable characters. Trailing white spaces are not allowed either."
			case NoNonPrintableCharsOrSpecialChars:
				summary = "Name can not be blank, nor contain the characters '\\', '/', '&', '<', '>', or any non-printable characters. Trailing white spaces are not allowed either."
			}
			diags = append(diags, v2diag.Diagnostic{
				Severity:      v2diag.Error,
				Summary:       summary,
				AttributePath: p,
			})
		}

		value := v.(string)
		if value == "" {
			fillDiags()
			return diags
		}

		for _, char := range value {
			if (mode == NoNonPrintableChars || mode == NoNonPrintableCharsOrSpecialChars) && !unicode.IsPrint(char) {
				fillDiags()
				return diags
			}
			if mode == NoNonPrintableCharsOrSpecialChars {
				switch char {
				case '\\', '/', '&', '<', '>':
					fillDiags()
					return diags
				}
			}
		}

		if strings.HasSuffix(value, " ") {
			fillDiags()
			return diags
		}

		return diags
	}
}

type alertGroupingParametersValidator struct {
	stringDescriber
	oneOfValidator validator.String
}

func (v alertGroupingParametersValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if len(req.ConfigValue.Elements()) < 1 {
		resp.Diagnostics.AddError("Expecting at least one element for alert_grouping_parameters", "")
		return
	}

	var target []struct {
		Type   types.String
		Config types.List
	}
	if d := req.ConfigValue.ElementsAs(ctx, &target, false); d.HasError() {
		resp.Diagnostics.Append(d...)
		return
	}
	obj := target[0]

	{
		vreq := validator.StringRequest{
			Path:        req.Path.AtName("type"),
			Config:      req.Config,
			ConfigValue: obj.Type,
		}
		vresp := &validator.StringResponse{}
		v.oneOfValidator.ValidateString(ctx, vreq, vresp)
		resp.Diagnostics.Append(vresp.Diagnostics...)
	}
}

func /*DEBUG */ ValidateOneOf(value types.String, allowed ...string) (diags diag.Diagnostics) {
	found := false
	for _, a := range allowed {
		if a == value.ValueString() {
			found = true
			break
		}
	}
	if !found {
		diags.AddError(fmt.Sprint("Expecting alert_grouping_parameters.type to be one of", allowed), "")
		return
	}
	return
}

func ValidateAlertGroupingParametersType(allowedTypes ...string) validator.List {
	// TODO
	// return &alertGroupingParametersValidator{stringDescriber{""}, allowedTypes}
	return &alertGroupingParametersValidator{stringDescriber{""}, stringvalidator.OneOf(allowedTypes...)}
}
