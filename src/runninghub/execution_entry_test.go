package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecutionEntryURL(t *testing.T) {
	for _, value := range []string{"123", "https://www.runninghub.ai/workflow/123?from=workspace", "https://www.runninghub.ai/zh-cn/workflow/123", "https://runninghub.ai/post/123"} {
		if executionWorkflowID(value) != "123" {
			t.Fatal(value)
		}
	}
	for _, value := range []string{"https://www.runninghub.cn/workflow/123", "https://evil.invalid/workflow/123", "https://www.runninghub.ai/workflow/123/extra", "abc"} {
		rejects(t, func() { executionWorkflowID(value) })
	}
}

func TestExecutionEntrySavedAfterReadAndUsedForSubmit(t *testing.T) {
	c := testClient(t, apiHandler(func(w http.ResponseWriter, r *http.Request) {
		var body M
		json.NewDecoder(r.Body).Decode(&body)
		if body["workflowId"] != "987" {
			t.Error("did not use saved connection entry", body["workflowId"])
		}
		w.Write([]byte(`{"code":0,"data":{"taskId":"test-task"}}`))
	}, true))
	c.accountID = "team"
	c.credentialName = "member"
	c.config = M{"accounts": M{"team": M{"keys": M{}}}}
	c.path = filepath.Join(t.TempDir(), "runninghub.json")
	// Access verification returns an empty saved graph; must not submit a task.
	originalHTTP := c.http
	originalBase := c.base
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/openapi/getJsonApiFormat" {
			t.Error("unexpected network call during setup", r.URL.Path)
		}
		w.Write([]byte(`{"code":0,"data":{"prompt":"{}"}}`))
	}))
	defer server.Close()
	c.http = server.Client()
	c.base = server.URL
	c.setExecutionEntry("https://www.runninghub.ai/workflow/987")
	if c.configuredWorkflowID() != "987" {
		t.Fatal("entry not saved")
	}
	c.http = originalHTTP
	c.base = originalBase
	a := submitArgs(t)
	delete(a.opts, "workflow-id")
	c.submit(a)
	c.credentialName = "other"
	if c.configuredWorkflowID() != "" {
		t.Fatal("entry leaked across credentials")
	}
}

func TestExecutionEntry810DoesNotSaveOrSubmit(t *testing.T) {
	calls := 0
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/openapi/getJsonApiFormat" {
			t.Error("unexpected submission", r.URL.Path)
		}
		w.Write([]byte(`{"code":810,"msg":"WORKFLOW_NOT_SAVED_OR_NOT_RUNNING"}`))
	})
	c.accountID = "team"
	c.credentialName = "member"
	c.config = M{"accounts": M{"team": M{"keys": M{}}}}
	defer func() {
		value := recover()
		if value == nil || !strings.Contains(str(value), "rh-bootstrap.workflow.json") {
			t.Error("missing actionable 810 guidance", value)
		}
		if calls != 1 || exists(c.path) || c.configuredWorkflowID() != "" {
			t.Fatal("810 must not retry, submit or save")
		}
	}()
	c.setExecutionEntry("123")
}
