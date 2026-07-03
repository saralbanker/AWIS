package sdk

import "github.com/awis/awis/internal/core"

// Trigger declares one way a workflow may be started: a Type and a type-specific
// Config map. Its serialized field names are frozen at Gate G1.
type Trigger = core.Trigger

// TriggerType enumerates the kinds of workflow trigger.
type TriggerType = core.TriggerType

// The frozen trigger-type values.
const (
	// TriggerTypeManual starts the workflow by explicit submission.
	TriggerTypeManual = core.TriggerTypeManual
	// TriggerTypeSchedule starts the workflow on a schedule.
	TriggerTypeSchedule = core.TriggerTypeSchedule
	// TriggerTypeEvent starts the workflow on a domain event.
	TriggerTypeEvent = core.TriggerTypeEvent
	// TriggerTypeWebhook starts the workflow on an inbound webhook.
	TriggerTypeWebhook = core.TriggerTypeWebhook
)
