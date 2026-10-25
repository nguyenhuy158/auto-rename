package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fileRecord struct {
	ID           int    `json:"id"`
	OriginalName string `json:"original_name"`
	NewName      string `json:"new_name"`
	FilePath     string `json:"file_path"`
	Status       string `json:"status"`
}

type duplicateGroup struct {
	SHA256 string `json:"sha256"`
	Files  []struct {
		Path string `json:"path"`
	} `json:"files"`
}

type quarantineAction struct {
	ID             int    `json:"id"`
	OriginalPath   string `json:"original_path"`
	QuarantinePath string `json:"quarantine_path"`
	Status         string `json:"status"`
}

func request(t *testing.T, method, endpoint string, body any, result any) {
	t.Helper()
	var data io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		data = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, endpoint, data)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := (&http.Client{Timeout: 25 * time.Second}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		t.Fatalf("%s %s: HTTP %d: %s", method, endpoint, response.StatusCode, message)
	}
	if result != nil {
		if err := json.NewDecoder(response.Body).Decode(result); err != nil {
			t.Fatal(err)
		}
	}
}

func awaitRun(t *testing.T, base string) map[string]any {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		var response struct {
			Current map[string]any `json:"current"`
		}
		request(t, http.MethodGet, base+"/api/runs", nil, &response)
		if response.Current["status"] != "running" {
			if response.Current["status"] != "completed" {
				t.Fatalf("run did not complete: %+v", response.Current)
			}
			return response.Current
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("run timed out")
	return nil
}

func TestScanQuarantineAndRestore(t *testing.T) {
	base := os.Getenv("AUTO_RENAME_E2E_URL")
	if base == "" {
		t.Skip("run scripts/e2e.sh")
	}
	if os.Getenv("AUTO_RENAME_E2E_PHASE") != "flow" {
		t.Skip("different e2e phase")
	}
	source := os.Getenv("AUTO_RENAME_E2E_SOURCE")
	quarantine := os.Getenv("AUTO_RENAME_E2E_QUARANTINE")
	appSource := os.Getenv("AUTO_RENAME_E2E_APP_SOURCE")
	appQuarantine := os.Getenv("AUTO_RENAME_E2E_APP_QUARANTINE")
	resolvedSource, err := filepath.EvalSymlinks(source)
	if err != nil {
		t.Fatal(err)
	}
	if appSource == source {
		appSource = resolvedSource
	}

	var started map[string]any
	request(t, http.MethodPost, base+"/api/cron/trigger", nil, &started)
	run := awaitRun(t, base)
	if run["renamed_files"] != float64(3) {
		t.Fatalf("unexpected rename run: %+v", run)
	}
	for _, name := range []string{"first.mp4", "older.mp4", "newer.mp4"} {
		if _, err := os.Stat(filepath.Join(source, name)); !os.IsNotExist(err) {
			t.Fatalf("source name still present: %s", name)
		}
	}

	var records struct {
		Records []fileRecord `json:"records"`
	}
	request(t, http.MethodGet, base+"/api/records?pageSize=20", nil, &records)
	if len(records.Records) != 3 {
		t.Fatalf("expected three D1 records, got %d", len(records.Records))
	}
	var first fileRecord
	for _, record := range records.Records {
		if record.Status != "renamed" || record.FilePath != appSource {
			t.Fatalf("unexpected D1 record: %+v", record)
		}
		if _, err := os.Stat(filepath.Join(source, record.NewName)); err != nil {
			t.Fatal(err)
		}
		if record.OriginalName == "first.mp4" {
			first = record
		}
	}
	if first.ID == 0 {
		t.Fatal("first record not found")
	}
	request(t, http.MethodPost, base+"/api/records/restore", map[string]any{"id": first.ID}, nil)
	if _, err := os.Stat(filepath.Join(source, "first.mp4")); err != nil {
		t.Fatal(err)
	}

	request(t, http.MethodPost, base+"/api/duplicates/scan", nil, &started)
	awaitRun(t, base)
	var groups struct {
		Groups []duplicateGroup `json:"groups"`
	}
	request(t, http.MethodGet, base+"/api/duplicates", nil, &groups)
	if len(groups.Groups) != 1 || len(groups.Groups[0].Files) != 2 {
		t.Fatalf("unexpected duplicate groups: %+v", groups.Groups)
	}
	var oldest fileRecord
	for _, record := range records.Records {
		if record.OriginalName == "older.mp4" {
			oldest = record
		}
	}
	if groups.Groups[0].Files[0].Path != filepath.Join(appSource, oldest.NewName) {
		t.Fatal("oldest duplicate was not kept")
	}

	request(t, http.MethodPost, base+"/api/duplicates/quarantine", map[string]any{"sha256": groups.Groups[0].SHA256, "all": true, "confirm": true}, &started)
	awaitRun(t, base)
	var actions struct {
		Actions []quarantineAction `json:"actions"`
	}
	request(t, http.MethodGet, base+"/api/quarantine", nil, &actions)
	if len(actions.Actions) != 1 || actions.Actions[0].Status != "quarantined" {
		t.Fatalf("unexpected D1 quarantine: %+v", actions.Actions)
	}
	if !strings.HasPrefix(actions.Actions[0].QuarantinePath, appQuarantine+string(os.PathSeparator)) {
		t.Fatal("quarantine path outside fixture")
	}
	originalPath := filepath.Join(source, strings.TrimPrefix(actions.Actions[0].OriginalPath, appSource+string(os.PathSeparator)))
	quarantinePath := filepath.Join(quarantine, strings.TrimPrefix(actions.Actions[0].QuarantinePath, appQuarantine+string(os.PathSeparator)))
	if _, err := os.Stat(originalPath); !os.IsNotExist(err) {
		t.Fatal("duplicate source still present")
	}
	if _, err := os.Stat(quarantinePath); err != nil {
		t.Fatal(err)
	}
	request(t, http.MethodPost, base+"/api/quarantine/restore", map[string]any{"id": actions.Actions[0].ID}, nil)
	if _, err := os.Stat(originalPath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(quarantinePath); !os.IsNotExist(err) {
		t.Fatal("quarantine file still present")
	}

	worker := os.Getenv("AUTO_RENAME_E2E_WORKER")
	token := os.Getenv("AUTO_RENAME_E2E_TOKEN")
	for _, lookup := range []struct {
		Op     string
		ID     int
		Status string
	}{{"record.get", first.ID, "restored"}, {"quarantine.get", actions.Actions[0].ID, "restored"}} {
		payload, _ := json.Marshal(map[string]any{"op": lookup.Op, "input": map[string]any{"id": lookup.ID}})
		req, err := http.NewRequest(http.MethodPost, worker+"/rpc", bytes.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		response, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		var result struct {
			Result struct {
				Status string `json:"status"`
			} `json:"result"`
		}
		if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusOK || result.Result.Status != lookup.Status {
			t.Fatal(fmt.Errorf("D1 %s status = %s, HTTP %d", lookup.Op, result.Result.Status, response.StatusCode))
		}
	}
}

func TestReconcileInterruptedMoves(t *testing.T) {
	phase := os.Getenv("AUTO_RENAME_E2E_PHASE")
	if phase == "" || phase == "flow" {
		t.Skip("run scripts/e2e.sh")
	}
	base := os.Getenv("AUTO_RENAME_E2E_URL")
	source := os.Getenv("AUTO_RENAME_E2E_SOURCE")
	appSource := os.Getenv("AUTO_RENAME_E2E_APP_SOURCE")
	if appSource == source {
		resolved, err := filepath.EvalSymlinks(source)
		if err != nil {
			t.Fatal(err)
		}
		appSource = resolved
	}
	scenarios := map[string]struct {
		name    string
		newName string
	}{
		"plan_applied":   {"interrupted.mp4", "11111111-1111-4111-8111-111111111111.mp4"},
		"reconciled":     {"interrupted.mp4", "11111111-1111-4111-8111-111111111111.mp4"},
		"plan_ambiguous": {"ambiguous.mp4", "22222222-2222-4222-8222-222222222222.mp4"},
		"blocked":        {"ambiguous.mp4", "22222222-2222-4222-8222-222222222222.mp4"},
	}
	scenario, ok := scenarios[phase]
	if !ok {
		t.Fatalf("unknown e2e phase: %s", phase)
	}
	original := filepath.Join(source, scenario.name)
	target := filepath.Join(source, scenario.newName)
	if strings.HasPrefix(phase, "plan_") {
		content := []byte("interrupted operation fixture")
		if err := os.WriteFile(original, content, 0600); err != nil {
			t.Fatal(err)
		}
		var inserted struct {
			ID int `json:"id"`
		}
		workerRequest(t, "record.insert", map[string]any{
			"original_name": scenario.name,
			"new_name":      scenario.newName,
			"file_path":     appSource,
			"file_size":     len(content),
			"file_mode":     "-rw-------",
			"mod_time":      time.Now().UTC().Format(time.RFC3339),
			"success":       false,
			"error_msg":     "",
			"renamed_at":    time.Now().UTC().Format(time.RFC3339),
			"status":        "planned",
		}, &inserted)
		if inserted.ID == 0 {
			t.Fatal("D1 did not create a pending record")
		}
		if err := os.Link(original, target); err != nil {
			t.Fatal(err)
		}
		if phase == "plan_applied" {
			if err := os.Remove(original); err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	var records struct {
		Records []fileRecord `json:"records"`
	}
	request(t, http.MethodGet, base+"/api/records?search="+scenario.name, nil, &records)
	if len(records.Records) != 1 {
		t.Fatalf("expected one D1 record: %+v", records.Records)
	}
	want := "renamed"
	if phase == "blocked" {
		want = "needs_review"
	}
	if records.Records[0].Status != want {
		t.Fatalf("reconciliation status = %s, want %s", records.Records[0].Status, want)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatal(err)
	}
	if phase == "reconciled" {
		if _, err := os.Stat(original); !os.IsNotExist(err) {
			t.Fatal("source was not moved")
		}
		return
	}
	if _, err := os.Stat(original); err != nil {
		t.Fatal(err)
	}
	response, err := http.Post(base+"/api/cron/trigger", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("mutation was not blocked: HTTP %d", response.StatusCode)
	}
}

func workerRequest(t *testing.T, operation string, input any, output any) {
	t.Helper()
	payload, err := json.Marshal(map[string]any{"op": operation, "input": input})
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, os.Getenv("AUTO_RENAME_E2E_WORKER")+"/rpc", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("AUTO_RENAME_E2E_TOKEN"))
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("D1 %s returned HTTP %d", operation, response.StatusCode)
	}
	var result struct {
		Result json.RawMessage `json:"result"`
		Error  string          `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Error != "" {
		t.Fatal(result.Error)
	}
	if output != nil {
		if err := json.Unmarshal(result.Result, output); err != nil {
			t.Fatal(err)
		}
	}
}
