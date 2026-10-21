package main

import (
	"net/url"
	"regexp"
	"strings"
)

var workflowNumber = regexp.MustCompile(`^[0-9]+$`)

func executionWorkflowID(value string) string {
	value = strings.TrimSpace(value)
	if workflowNumber.MatchString(value) {
		return value
	}
	u, err := url.Parse(value)
	need(err == nil && u.Scheme == "https" && (u.Host == "www.runninghub.ai" || u.Host == "runninghub.ai") && u.User == nil, "Provide an international RunningHub workflow URL or numeric workflow ID.")
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i := 0; i+1 < len(parts); i++ {
		if (parts[i] == "workflow" || parts[i] == "post") && workflowNumber.MatchString(parts[i+1]) && i+2 == len(parts) {
			return parts[i+1]
		}
	}
	fail("Workflow ID not found in the international workflow URL.")
	return ""
}

func (c *client) entryOwner() M {
	if c.accountID != "" {
		return mapOrEmpty(mapOrEmpty(c.config["accounts"])[c.accountID])
	}
	return c.profile
}

func (c *client) configuredWorkflowID() string {
	entry := mapOrEmpty(c.entryOwner()["execution_entry"])
	if entry["verified"] != true {
		return ""
	}
	if c.accountID != "" && str(entry["credential"]) != c.credentialName {
		return ""
	}
	return str(entry["workflow_id"])
}

func (c *client) setExecutionEntry(value string) M {
	id := executionWorkflowID(value)
	response := c.post("/api/openapi/getJsonApiFormat", M{"workflowId": id}, false)
	if str(response["code"]) == "810" {
		fail("RH 810 WORKFLOW_NOT_SAVED_OR_NOT_RUNNING: import the bundled rh-bootstrap.workflow.json, save and manually run it once on the website, then verify again. See references/integrations/runninghub/execution-entry.md. Do not repeatedly retry unchanged blank workflows; no task was submitted and the entry was not saved.")
	}
	data := obj(c.success(response))
	prompt, ok := data["prompt"]
	need(ok && prompt != nil, "Workflow access check did not return a graph; execution entry was not saved.")
	if text, ok := prompt.(string); ok {
		prompt = decode([]byte(text))
	}
	obj(prompt) // Accept the returned object; never infer a new blank canvas is initialized.
	owner := c.entryOwner()
	need(len(owner) > 0, "Configure this RH connection before its execution entry.")
	owner["execution_entry"] = M{"workflow_id": id, "verified": true, "checked_at": now(), "credential": c.credentialName}
	save(c.path, c.config)
	return M{"saved": true, "workflow_id": id, "access_verified": true, "check": "read-only workflow access; no generation task submitted", "account": c.accountID, "credential": c.credentialName}
}
