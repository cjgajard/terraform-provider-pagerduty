package pagerduty

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/PagerDuty/go-pagerduty"
	"github.com/PagerDuty/terraform-provider-pagerduty/util"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
)

// Config defines the configuration options for the PagerDuty client
type Config struct {
	// The PagerDuty API URL
	APIURL string

	// Override default PagerDuty API URL
	APIURLOverride string

	// The PagerDuty APP URL
	AppURL string

	// The PagerDuty API V2 token
	Token string

	// The PagerDuty User level token for Slack
	UserToken string

	// Skip validation of the token against the PagerDuty API
	SkipCredsValidation bool

	// Target version for terraform
	TerraformVersion string

	// Region where the server of the service is deployed
	ServiceRegion string

	// Do not verify TLS certs for HTTPS requests - useful if you're behind a corporate proxy
	InsecureTls bool

	// Parameters for fine-grained access control
	AppOauthScopedToken *AppOauthScopedToken
}

type AppOauthScopedToken struct {
	ClientID, ClientSecret, Subdomain string
}

const invalidCreds = `
No valid credentials found for PagerDuty provider.
Please see https://www.terraform.io/docs/providers/pagerduty/index.html
for more information on providing credentials for this provider.
`

var (
	RetryTime     = 2 * time.Minute
	RetryTimeLong = 5 * time.Minute
)

// Client returns a PagerDuty client, initializing when necessary.
func (c *Config) Client(ctx context.Context) (*pagerduty.Client, error) {
	clientOpts := []pagerduty.ClientOptions{}
	if c.AppOauthScopedToken != nil {
		tokenFile := getTokenFilepath()
		account := fmt.Sprintf("as_account-%s.%s", c.ServiceRegion, c.AppOauthScopedToken.Subdomain)
		accountAndScopes := []string{account}
		accountAndScopes = append(accountAndScopes, availableOauthScopes()...)
		opt := pagerduty.WithScopedOAuthAppTokenSource(pagerduty.NewFileTokenSource(
			ctx,
			c.AppOauthScopedToken.ClientID,
			c.AppOauthScopedToken.ClientSecret,
			accountAndScopes,
			tokenFile,
		))
		clientOpts = append(clientOpts, opt)
	}

	// Validate that the PagerDuty token is set
	if c.Token == "" && c.AppOauthScopedToken == nil {
		return nil, fmt.Errorf(invalidCreds)
	}

	client, err := c.getClient(ctx, c.Token, clientOpts)
	if err != nil {
		return nil, err
	}

	log.Printf("[INFO] PagerDuty plugin client configured")
	return client, err
}

// SlackClient returns a PagerDuty client using an special token for Slack.
func (c *Config) SlackClient(ctx context.Context) (*pagerduty.Client, error) {
	if c.UserToken == "" {
		return nil, fmt.Errorf(invalidCreds)
	}
	slackClient, err := c.getClient(ctx, c.UserToken, []pagerduty.ClientOptions{})
	if err != nil {
		return nil, err
	}
	return slackClient, nil
}

func (c *Config) getClient(ctx context.Context, token string, opts []pagerduty.ClientOptions) (*pagerduty.Client, error) {
	httpClient := http.DefaultClient
	httpClient.Timeout = 1 * time.Minute

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if c.InsecureTls {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	httpClient.Transport = logging.NewTransport("PagerDuty", transport)

	apiURL := c.APIURL
	if c.APIURLOverride != "" {
		apiURL = c.APIURLOverride
	}

	maxRetries := 1
	retryInterval := 60 // seconds

	clientOpts := []pagerduty.ClientOptions{
		WithHTTPClient(httpClient),
		pagerduty.WithAPIEndpoint(apiURL),
		pagerduty.WithTerraformProvider(c.TerraformVersion),
		pagerduty.WithRetryPolicy(maxRetries, retryInterval),
	}
	clientOpts = append(clientOpts, opts...)
	client := pagerduty.NewClient(token, clientOpts...)

	if !c.SkipCredsValidation {
		// Validate the credentials by calling the abilities endpoint,
		// if we get a 401 response back we return an error to the user
		if _, err := client.ListAbilitiesWithContext(ctx); err != nil {
			return nil, fmt.Errorf(fmt.Sprintf("%s\n%s", err, invalidCreds))
		}
	}
	return client, nil
}

func WithHTTPClient(httpClient pagerduty.HTTPClient) pagerduty.ClientOptions {
	return func(c *pagerduty.Client) {
		if util.IsNilFunc(httpClient) {
			return
		}
		c.HTTPClient = httpClient
	}
}

func getTokenFilepath() string {
	dir, err := os.UserHomeDir()
	if err == nil {
		dir = filepath.Join(dir, ".pagerduty")
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			os.Mkdir(dir, os.ModeDir|0o755)
		}
	} else {
		dir = ""
	}
	return filepath.Join(dir, "token.json")
}

func availableOauthScopes() []string {
	return []string{
		"abilities.read",
		"addons.read",
		"addons.write",
		"analytics.read",
		"audit_records.read",
		"change_events.read",
		"change_events.write",
		"custom_fields.read",
		"custom_fields.write",
		"escalation_policies.read",
		"escalation_policies.write",
		"event_orchestrations.read",
		"event_orchestrations.write",
		"event_rules.read",
		"event_rules.write",
		"extension_schemas.read",
		"extensions.read",
		"extensions.write",
		"incident_workflows.read",
		"incident_workflows.write",
		"incident_workflows:instances.write",
		"incidents.read",
		"incidents.write",
		"licenses.read",
		"notifications.read",
		"oncalls.read",
		"priorities.read",
		"response_plays.read",
		"response_plays.write",
		"schedules.read",
		"schedules.write",
		"services.read",
		"services.write",
		"standards.read",
		"standards.write",
		"status_dashboards.read",
		"status_pages.read",
		"status_pages.write",
		"subscribers.read",
		"subscribers.write",
		"tags.read",
		"tags.write",
		"teams.read",
		"teams.write",
		"templates.read",
		"templates.write",
		"users.read",
		"users.write",
		"users:contact_methods.read",
		"users:contact_methods.write",
		"users:sessions.read",
		"users:sessions.write",
		"vendors.read",
	}
}

// ConfigurePagerdutyClient sets a pagerduty API client in a pointer `dst` to
// the property of any datasource or resource struct from the general
// configuration of the provider.
func ConfigurePagerdutyClient(dst **pagerduty.Client, providerData any) diag.Diagnostics {
	data, d := getPagerdutyProviderData(dst, providerData)
	if d.HasError() {
		return d
	}
	*dst = data.client
	return d
}

func ConfigurePagerdutySlackClient(dst **pagerduty.Client, providerData any) diag.Diagnostics {
	data, d := getPagerdutyProviderData(dst, providerData)
	if d.HasError() {
		return d
	}
	*dst = data.slackClient
	return d
}

func getPagerdutyProviderData(dst any, providerData any) (ProviderData, diag.Diagnostics) {
	var diags diag.Diagnostics

	if providerData == nil {
		return ProviderData{}, diags
	}

	data, ok := providerData.(ProviderData)
	if !ok {
		diags.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf(
				"Expected *github.com/PagerDuty/go-pagerduty.Client, got: %T."+
					"Please report this issue to the provider developers.",
				providerData,
			),
		)
		return data, diags
	}

	if util.IsNilFunc(dst) {
		diags.AddError(
			"Bad usage of getPagerdutyProviderData",
			"Received a null client destination",
		)
		return data, diags
	}

	return data, diags
}
