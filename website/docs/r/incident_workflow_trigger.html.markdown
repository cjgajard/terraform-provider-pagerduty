---
layout: "pagerduty"
page_title: "PagerDuty: pagerduty_incident_workflow_trigger"
sidebar_current: "docs-pagerduty-resource-incident-workflow-trigger"
description: |-
  Creates and manages an incident workflow trigger in PagerDuty.
---

# pagerduty\_incident\_workflow\_trigger

An [Incident Workflow Trigger](https://support.pagerduty.com/docs/incident-workflows#triggers) defines when and if an [Incident Workflow](https://support.pagerduty.com/docs/incident-workflows) will be triggered.

## Example Usage

```hcl
resource "pagerduty_incident_workflow" "my_first_workflow" {
  name         = "Example Incident Workflow"
  description  = "This Incident Workflow is an example"
  step {
    name           = "Send Status Update"
    action         = "pagerduty.com:incident-workflows:send-status-update:1"
    input {
      name = "Message"
      value = "Example status message sent on {{current_date}}"
    }
  }
}

data "pagerduty_service" "first_service" {
  name = "My First Service"
}

resource "pagerduty_incident_workflow_trigger" "automatic_trigger" {
  type                       = "conditional"
  workflow                   = pagerduty_incident_workflow.my_first_workflow.id
  services                   = [pagerduty_service.first_service.id]
  condition                  = "incident.priority matches 'P1'"
  subscribed_to_all_services = false
}

data "pagerduty_team" "devops" {
  name = "devops"
}

resource "pagerduty_incident_workflow_trigger" "manual_trigger" {
  type       = "manual"
  workflow   = pagerduty_incident_workflow.my_first_workflow.id
  services   = [pagerduty_service.first_service.id]
}

resource "pagerduty_incident_type" "security" {
  name         = "security_incident"
  display_name = "Security Incident"
  parent_type  = "incident_default"
  enabled      = true
}

resource "pagerduty_incident_workflow_trigger" "incident_type_trigger" {
  type                       = "incident_type"
  workflow                   = pagerduty_incident_workflow.my_first_workflow.id
  incident_types             = [pagerduty_incident_type.security.id]
  subscribed_to_all_services = true
}

```

## Argument Reference

The following arguments are supported:

* `type` - (Required) [Updating causes resource replacement] May be `manual`, `conditional` or `incident_type`.
* `workflow` - (Required) The workflow ID for the workflow to trigger.
* `services` - (Optional) A list of service IDs. Incidents in any of the listed services are eligible to fire this trigger.
* `incident_types` - (Optional) A list of incident type IDs. Incidents of any of the listed types are eligible to fire this trigger. Required for, and only allowed on, `incident_type`-type triggers.
* `subscribed_to_all_services` - (Required) Set to `true` if the trigger should be eligible for firing on all services. Only allowed to be `true` if the services list is not defined or empty.
* `permissions` - (Optional) Indicates who can start this Trigger. Applicable only to `manual`-type triggers. Omitting this block entirely is different from an empty state: the block is only present in state when explicitly configured.
  * `restricted` - (Optional) If `true`, indicates that the Trigger can only be started by authorized Users. If `false` (default), any user can start this Trigger. Applicable only to `manual`-type triggers.
  * `team_id` - (Optional) The ID of the Team whose members can manually start this Trigger. Required and allowed only if `restricted` is `true`.
* `condition` - (Required for `conditional`-type triggers, not allowed for `manual`- or `incident_type`-type triggers) A [PCL](https://developer.pagerduty.com/docs/ZG9jOjM1NTE0MDc0-pcl-overview) condition string which must be satisfied for the trigger to fire.

## Attributes Reference

The following attributes are exported:

* `id` - The ID of the incident workflow trigger.

## Import

Incident workflow triggers can be imported using the `id`, e.g.

```
$ terraform import pagerduty_incident_workflow_trigger.main PLBP09X
```
