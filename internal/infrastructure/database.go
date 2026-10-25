package infrastructure

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"auto-rename/internal/domain"
)

type Database struct {
	endpoint string
	token    string
	client   *http.Client
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  string          `json:"error"`
}

type recordStats struct {
	Total      int `json:"total"`
	Successful int `json:"successful"`
	Failed     int `json:"failed"`
	Recent     int `json:"recent"`
}

type Page[T any] struct {
	Rows  []T `json:"rows"`
	Total int `json:"total"`
}

type SearchResults struct {
	Records []domain.FileRecord       `json:"records"`
	Runs    []domain.ScanRun          `json:"runs"`
	Actions []domain.QuarantineAction `json:"actions"`
}

func NewDatabase(endpoint, token string) (*Database, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && !(parsed.Scheme == "http" && (parsed.Hostname() == "localhost" || parsed.Hostname() == "127.0.0.1"))) {
		return nil, errors.New("D1 API URL must use HTTPS or local HTTP")
	}
	if token == "" {
		return nil, errors.New("AUTO_RENAME_API_TOKEN is required")
	}
	return &Database{endpoint: strings.TrimRight(endpoint, "/") + "/rpc", token: token, client: &http.Client{Timeout: 20 * time.Second}}, nil
}

func (d *Database) call(op string, input any, output any) error {
	body, err := json.Marshal(map[string]any{"op": op, "input": input})
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPost, d.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+d.token)
	request.Header.Set("Content-Type", "application/json")
	response, err := d.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("D1 API returned HTTP %d", response.StatusCode)
	}
	var result rpcResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	if result.Error != "" {
		return errors.New(result.Error)
	}
	if output == nil {
		return nil
	}
	if len(result.Result) == 0 || string(result.Result) == "null" {
		return errors.New("D1 record not found")
	}
	return json.Unmarshal(result.Result, output)
}

func (d *Database) Ping() error {
	var result recordStats
	return d.call("record.stats", nil, &result)
}

func (d *Database) HasOriginalName(name string) (bool, error) {
	var result struct {
		Count int `json:"count"`
	}
	err := d.call("record.hasOriginal", map[string]any{"name": name}, &result)
	return result.Count > 0, err
}

func (d *Database) RestoredNames() (map[string]bool, error) {
	var rows []struct {
		Path string `json:"file_path"`
		Name string `json:"original_name"`
	}
	if err := d.call("record.restored", nil, &rows); err != nil {
		return nil, err
	}
	names := make(map[string]bool, len(rows))
	for _, row := range rows {
		names[filepath.Join(row.Path, row.Name)] = true
	}
	return names, nil
}

func (d *Database) CreateFileRecord(record domain.FileRecord) (int, error) {
	var result struct {
		ID int `json:"id"`
	}
	err := d.call("record.insert", record, &result)
	return result.ID, err
}

func (d *Database) InsertFileRecord(record domain.FileRecord) error {
	_, err := d.CreateFileRecord(record)
	return err
}

func (d *Database) GetFileRecord(id int) (domain.FileRecord, error) {
	var record domain.FileRecord
	err := d.call("record.get", map[string]any{"id": id}, &record)
	return record, err
}

func (d *Database) PendingFileRecords() ([]domain.FileRecord, error) {
	var records []domain.FileRecord
	err := d.call("record.pending", nil, &records)
	return records, err
}

func (d *Database) UpdateFileRecord(id int, status string, success bool, message, restoredAt string) error {
	return d.call("record.update", map[string]any{"id": id, "status": status, "success": success, "error_msg": message, "restored_at": restoredAt}, nil)
}

func (d *Database) GetRecordsByFilter(page, pageSize int, search, status, dateFrom, dateTo string) ([]domain.FileRecord, error) {
	return d.ListRecordsByFilter(page, pageSize, search, status, dateFrom, dateTo, "id", "desc")
}

func (d *Database) ListRecordsByFilter(page, pageSize int, search, status, dateFrom, dateTo, sort, order string) ([]domain.FileRecord, error) {
	var records []domain.FileRecord
	err := d.call("record.list", map[string]any{"limit": pageSize, "offset": (page - 1) * pageSize, "search": search, "status": status, "dateFrom": dateFrom, "dateTo": dateTo, "sort": sort, "order": order}, &records)
	return records, err
}

func (d *Database) GetRecordCountByFilter(search, status, dateFrom, dateTo string) (int, error) {
	var result struct {
		Count int `json:"count"`
	}
	err := d.call("record.count", map[string]any{"search": search, "status": status, "dateFrom": dateFrom, "dateTo": dateTo}, &result)
	return result.Count, err
}

func (d *Database) GetRecentOperations(limit int) ([]domain.FileRecord, error) {
	return d.GetRecordsByFilter(1, limit, "", "", "", "")
}

func (d *Database) stats() (recordStats, error) {
	var result recordStats
	err := d.call("record.stats", nil, &result)
	return result, err
}

func (d *Database) GetTotalRecords() (int, error) {
	result, err := d.stats()
	return result.Total, err
}

func (d *Database) GetSuccessfulRenames() (int, error) {
	result, err := d.stats()
	return result.Successful, err
}

func (d *Database) GetFailedRenames() (int, error) {
	result, err := d.stats()
	return result.Failed, err
}

func (d *Database) GetRecentActivity(hours int) (int, error) {
	result, err := d.stats()
	return result.Recent, err
}

func (d *Database) SaveSetting(key, value string) error {
	return d.call("settings.save", map[string]any{"key": key, "value": value}, nil)
}

func (d *Database) StartRun(source string) (int, error) {
	var result struct {
		ID int `json:"id"`
	}
	err := d.call("run.start", map[string]any{"source": source, "started_at": time.Now().UTC().Format(time.RFC3339)}, &result)
	return result.ID, err
}

func (d *Database) UpdateRun(run domain.ScanRun) error {
	return d.call("run.update", run, nil)
}

func (d *Database) GetRecentRuns() ([]domain.ScanRun, error) {
	var runs []domain.ScanRun
	err := d.call("run.recent", nil, &runs)
	return runs, err
}

func (d *Database) ListRuns(page, pageSize int, search, sort, order string) (Page[domain.ScanRun], error) {
	var result Page[domain.ScanRun]
	err := d.call("run.page", map[string]any{"limit": pageSize, "offset": (page - 1) * pageSize, "search": search, "sort": sort, "order": order}, &result)
	return result, err
}

func (d *Database) CreateQuarantine(action domain.QuarantineAction) (int, error) {
	var result struct {
		ID int `json:"id"`
	}
	err := d.call("quarantine.insert", action, &result)
	return result.ID, err
}

func (d *Database) GetQuarantine(id int) (domain.QuarantineAction, error) {
	var action domain.QuarantineAction
	err := d.call("quarantine.get", map[string]any{"id": id}, &action)
	return action, err
}

func (d *Database) PendingQuarantines() ([]domain.QuarantineAction, error) {
	var actions []domain.QuarantineAction
	err := d.call("quarantine.pending", nil, &actions)
	return actions, err
}

func (d *Database) GetQuarantines() ([]domain.QuarantineAction, error) {
	var actions []domain.QuarantineAction
	err := d.call("quarantine.list", nil, &actions)
	return actions, err
}

func (d *Database) ListQuarantines(page, pageSize int, search, sort, order string) (Page[domain.QuarantineAction], error) {
	var result Page[domain.QuarantineAction]
	err := d.call("quarantine.page", map[string]any{"limit": pageSize, "offset": (page - 1) * pageSize, "search": search, "sort": sort, "order": order}, &result)
	return result, err
}

func (d *Database) SearchAll(query string) (SearchResults, error) {
	var result SearchResults
	err := d.call("search.all", map[string]any{"query": query}, &result)
	return result, err
}

func (d *Database) UpdateQuarantine(id int, status string) error {
	return d.call("quarantine.update", map[string]any{"id": id, "status": status}, nil)
}

func (d *Database) RestoreQuarantine(id int) error {
	return d.call("quarantine.restore", map[string]any{"id": id, "restored_at": time.Now().UTC().Format(time.RFC3339)}, nil)
}

func (d *Database) Close() error { return nil }
