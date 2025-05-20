# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This repository houses the Terraform Provider for PagerDuty, a plugin that allows for the management of PagerDuty resources using HashiCorp Configuration Language (HCL). The provider integrates with the PagerDuty API to create, manage and delete resources like teams, escalation policies, services, and more.

## Architecture

The codebase follows a Terraform provider structure with two parallel implementations:

1. **SDK-based Provider** (in `/pagerduty/` directory): Using HashiCorp's Terraform Plugin SDK v2
2. **Framework-based Provider** (in `/pagerdutyplugin/` directory): Using HashiCorp's newer Terraform Plugin Framework

Both providers are multiplexed together using the Terraform Plugin Mux. The main.go file sets up this multiplexing and serves the provider at the "registry.terraform.io/pagerduty/pagerduty" address.

Each implementation provides resources and data sources for PagerDuty entities. The codebase is transitioning towards the newer Framework-based implementation.

## Development Commands

### Building the Provider

```bash
make build
```

This will compile the provider and place the binary in the `$GOPATH/bin` directory.

### Running Tests

Run unit tests:

```bash
make test
```

Run acceptance tests (requires valid PagerDuty API tokens):

```bash
# For all tests
make testacc

# For specific tests
make testacc TESTARGS="-run TestAccPagerDutyTeam"
```

### Running Sweepers (Clean up test resources)

```bash
make sweep SWEEP_RESOURCE=pagerduty_service_custom_field
```

### Code Formatting and Validation

```bash
# Format code
make fmt

# Check code formatting
make fmtcheck

# Verify code for suspicious constructs
make vet

# Check for unchecked errors
make errcheck
```

### Testing with Local Changes

To test local changes with Terraform:

1. Build the provider: `make build`
2. Set up development overrides in `$HOME/.terraformrc` to use your local build
3. Run `terraform init -upgrade` in your Terraform module directory
4. Run Terraform commands as normal (`terraform plan`, `terraform apply`)

## Environment Variables

### Authentication:
- `PAGERDUTY_TOKEN`: API token for PagerDuty (used for provider authentication)
- `PAGERDUTY_USER_TOKEN`: User token for PagerDuty (used for certain operations)

### OAuth App Authentication:
- `PAGERDUTY_CLIENT_ID`: Client ID for PagerDuty OAuth app
- `PAGERDUTY_CLIENT_SECRET`: Client secret for PagerDuty OAuth app
- `PAGERDUTY_SUBDOMAIN`: PagerDuty subdomain for OAuth app

### Testing:
- `TF_ACC=1`: Enable acceptance tests
- `PAGERDUTY_ACC_*`: Various flags to enable specific acceptance tests

### Caching:
- `TF_PAGERDUTY_CACHE`: Enable caching (values: "memory" or MongoDB connection string)
- `TF_PAGERDUTY_CACHE_MAX_AGE`: Time in seconds for cached data to become stale (default: 10s)
- `TF_PAGERDUTY_CACHE_PREFILL`: Pre-fill cache with data

### Logging:
- `TF_LOG`: Terraform log level (TRACE, DEBUG, INFO, WARN, ERROR)
- `TF_LOG_PATH`: Path for Terraform logs
- `TF_LOG_PROVIDER_PAGERDUTY=SECURE`: Special log level that obfuscates API keys

## Notes for Implementation

When implementing new resources or data sources:

1. Consider which provider implementation to target (SDK or Framework)
2. Add appropriate documentation to `/website/docs/`
3. Include both resource and data source implementations if applicable
4. Add acceptance tests with appropriate configuration
5. Implement import functionality for resources
6. Update the CHANGELOG.md with your changes

For PagerDuty API interactions, the provider uses two different client libraries:
- `github.com/heimweh/go-pagerduty` (for SDK-based provider)
- `github.com/PagerDuty/go-pagerduty` (for Framework-based provider)

## Testing Specific Features

Some features require additional environment variables to enable testing due to account restrictions:

```bash
# Examples
PAGERDUTY_ACC_INCIDENT_WORKFLOWS=1 make testacc TESTARGS="-run PagerDutyIncidentWorkflow"
PAGERDUTY_ACC_SERVICE_INTEGRATION_GENERIC_EMAIL_NO_FILTERS="user@<your_domain>.pagerduty.com" make testacc TESTARGS="-run PagerDutyServiceIntegration_GenericEmailNoFilters"
```