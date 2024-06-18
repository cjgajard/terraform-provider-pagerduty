package util

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// defaultGetenv is a default that sets the value for a types.StringType
// attribute to the value of an environment variable when it is not configured.
// The attribute must be marked as Optional and Computed.
type defaultGetenv struct{ Name string }

func (d defaultGetenv) Description(ctx context.Context) string {
	return d.MarkdownDescription(ctx)
}

func (d defaultGetenv) MarkdownDescription(_ context.Context) string {
	return fmt.Sprintf("If value is not configured, defaults to the value of an environment variable")
}

func (d defaultGetenv) DefaultString(_ context.Context, req defaults.StringRequest, resp *defaults.StringResponse) {
	resp.PlanValue = types.StringValue(os.Getenv(d.Name))
}

func DefaultGetenv(name string) defaults.String {
	return defaultGetenv{Name: name}
}
