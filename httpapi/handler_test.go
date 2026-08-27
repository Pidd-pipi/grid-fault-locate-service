package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"example.com/grid-fault-locate-service/config"
	"example.com/grid-fault-locate-service/store"
)

// newTestApp 构造完整应用（内存仓储）。
func newTestApp(t *testing.T) (*App, *store.Store) {
	t.Helper()
	st, err := store.New("")
	if err != nil {
		t.Fatal(err)
	}
	app := NewApp(config.Default(), st, os.DirFS("testdata"))
	return app, st
}

func doJSON(t *testing.T, h http.Handler, method, path, body string) (*http.Response, map[string]any) {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = bytes.NewBufferString(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var payload map[string]any
	if strings.Contains(rec.Header().Get("Content-Type"), "json") || len(rec.Body.Bytes()) > 0 {
		_ = json.Unmarshal(rec.Body.Bytes(), &payload)
	}
	return rec.Result(), payload
}

func dataOf(payload map[string]any) map[string]any {
	if d, ok := payload["data"].(map[string]any); ok {
		return d
	}
	return nil
}

func TestHealthz(t *testing.T) {
	app, _ := newTestApp(t)
	h := app.Handler()
	for _, path := range []string{"/healthz", "/api/healthz"} {
		resp, payload := doJSON(t, h, http.MethodGet, path, "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s should be 200, got %d", path, resp.StatusCode)
		}
		if code, _ := payload["code"].(float64); code != 0 {
			t.Fatalf("unexpected code %v", payload["code"])
		}
	}
}

func TestIndexPageServed(t *testing.T) {
	app, _ := newTestApp(t)
	h := app.Handler()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / should be 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "test page") {
		t.Fatal("index page not served")
	}
	// 未知 API 路径应返回 JSON 404
	resp, payload := doJSON(t, h, http.MethodGet, "/api/nope", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown api should 404, got %d", resp.StatusCode)
	}
	_ = payload
}

// seedFullChain 通过 HTTP 创建线路-开关-区段-指示器并上报信号。
func seedFullChain(t *testing.T, h http.Handler) (feederID string, sectionID string, indicatorID string) {
	t.Helper()
	_, p := doJSON(t, h, http.MethodPost, "/api/feeders", `{"name":"集成线","substation":"站A","voltageLevel":"10kV"}`)
	feederID = dataOf(p)["id"].(string)

	_, p = doJSON(t, h, http.MethodPost, "/api/feeders/"+feederID+"/switches", `{"name":"出线开关","switchType":"feeder_outlet"}`)
	outletID := dataOf(p)["id"].(string)
	_, p = doJSON(t, h, http.MethodPost, "/api/feeders/"+feederID+"/switches", `{"name":"分段A","switchType":"sectionalizer"}`)
	swA := dataOf(p)["id"].(string)
	_, p = doJSON(t, h, http.MethodPost, "/api/feeders/"+feederID+"/switches", `{"name":"分段B","switchType":"sectionalizer"}`)
	swB := dataOf(p)["id"].(string)

	_, p = doJSON(t, h, http.MethodPost, "/api/feeders/"+feederID+"/sections", `{"name":"区段1","upstreamSwitchId":"`+outletID+`","downstreamSwitchId":"`+swA+`","lengthKm":1.2}`)
	sec1 := dataOf(p)["id"].(string)
	sectionID = sec1
	_, p = doJSON(t, h, http.MethodPost, "/api/feeders/"+feederID+"/sections", `{"name":"区段2","upstreamSwitchId":"`+swA+`","downstreamSwitchId":"`+swB+`","lengthKm":0.9}`)
	sec2 := dataOf(p)["id"].(string)

	_, p = doJSON(t, h, http.MethodPost, "/api/indicators", `{"name":"FI-101","sectionId":"`+sec1+`"}`)
	ind1 := dataOf(p)["id"].(string)
	indicatorID = ind1
	_, p = doJSON(t, h, http.MethodPost, "/api/indicators", `{"name":"FI-102","sectionId":"`+sec2+`"}`)
	ind2 := dataOf(p)["id"].(string)

	_, p = doJSON(t, h, http.MethodPost, "/api/indicators/"+ind1+"/signal", `{"status":"triggered"}`)
	if dataOf(p)["status"] != "triggered" {
		t.Fatalf("signal report failed: %v", p)
	}
	_, p = doJSON(t, h, http.MethodPost, "/api/indicators/"+ind2+"/signal", `{"status":"reset"}`)
	if dataOf(p)["status"] != "reset" {
		t.Fatalf("signal report failed: %v", p)
	}
	return feederID, sectionID, indicatorID
}

func TestFullClosedLoopViaHTTP(t *testing.T) {
	app, _ := newTestApp(t)
	h := app.Handler()
	feederID, sectionID, _ := seedFullChain(t, h)

	// 故障定位 → 建单
	resp, payload := doJSON(t, h, http.MethodPost, "/api/faults/locate", `{"feederId":"`+feederID+`"}`)
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("locate should be 201, got %d: %s", resp.StatusCode, body)
	}
	event := dataOf(payload)["event"].(map[string]any)
	locate := dataOf(payload)["locate"].(map[string]any)
	if event["status"] != "located" {
		t.Fatalf("event status should be located, got %v", event["status"])
	}
	if locate["primarySectionId"] != sectionID {
		t.Fatalf("primary should be %s, got %v", sectionID, locate["primarySectionId"])
	}
	faultID := event["id"].(string)

	// 未隔离直接复电 → 409
	resp, _ = doJSON(t, h, http.MethodPost, "/api/faults/"+faultID+"/restore", `{"operator":"调度员","note":"直接复电"}`)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("restore without isolation should 409, got %d", resp.StatusCode)
	}

	// 隔离确认 → 200
	resp, payload = doJSON(t, h, http.MethodPost, "/api/faults/"+faultID+"/isolate", `{"operator":"调度员","sectionId":"`+sectionID+`","note":"确认隔离"}`)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("isolate should be 200, got %d: %s", resp.StatusCode, body)
	}
	if dataOf(payload)["status"] != "repairing" {
		t.Fatalf("after isolate status should be repairing, got %v", dataOf(payload)["status"])
	}

	// 复电完成 → 200 + 停电记录生成
	resp, payload = doJSON(t, h, http.MethodPost, "/api/faults/"+faultID+"/restore", `{"operator":"调度员","note":"复电成功"}`)
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("restore should be 200, got %d: %s", resp.StatusCode, body)
	}
	if dataOf(payload)["status"] != "restored" {
		t.Fatalf("after restore status should be restored, got %v", dataOf(payload)["status"])
	}

	// 归档
	resp, payload = doJSON(t, h, http.MethodPost, "/api/faults/"+faultID+"/archive", `{"operator":"调度员"}`)
	if resp.StatusCode != http.StatusOK || dataOf(payload)["status"] != "archived" {
		t.Fatalf("archive failed: %d %v", resp.StatusCode, payload)
	}

	// 停电统计
	_, payload = doJSON(t, h, http.MethodGet, "/api/outages/summary", "")
	summary := dataOf(payload)
	if summary["totalRecords"].(float64) != 1 {
		t.Fatalf("expected 1 outage record, got %v", summary["totalRecords"])
	}

	// 总览
	_, payload = doJSON(t, h, http.MethodGet, "/api/overview", "")
	ov := dataOf(payload)
	if ov["feederCount"].(float64) != 1 {
		t.Fatalf("expected 1 feeder in overview, got %v", ov["feederCount"])
	}

	// 审计留痕
	_, payload = doJSON(t, h, http.MethodGet, "/api/audit?limit=100", "")
	audits := payload["data"].([]any)
	if len(audits) == 0 {
		t.Fatal("audit log should not be empty")
	}
	actions := map[string]bool{}
	for _, a := range audits {
		actions[a.(map[string]any)["action"].(string)] = true
	}
	for _, want := range []string{"fault.locate", "fault.isolate", "fault.restore", "fault.archive", "http.request"} {
		if !actions[want] {
			t.Fatalf("audit missing %s: %v", want, actions)
		}
	}
}

func TestLocateNoSignalRejected(t *testing.T) {
	app, _ := newTestApp(t)
	h := app.Handler()
	feederID, _, indicatorID := seedFullChain(t, h)
	// 复位全部信号
	_, payload := doJSON(t, h, http.MethodGet, "/api/indicators", "")
	for _, item := range payload["data"].([]any) {
		ind := item.(map[string]any)
		_, _ = doJSON(t, h, http.MethodPost, "/api/indicators/"+ind["id"].(string)+"/signal", `{"status":"reset"}`)
	}
	_ = indicatorID
	resp, payload := doJSON(t, h, http.MethodPost, "/api/faults/locate", `{"feederId":"`+feederID+`"}`)
	_ = payload
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("locate with no triggered signal should 400, got %d", resp.StatusCode)
	}
}

func TestTopologyCycleRejectedViaHTTP(t *testing.T) {
	app, _ := newTestApp(t)
	h := app.Handler()
	feederID, _, _ := seedFullChain(t, h)
	sws := e2eSwitches(t, h, feederID)
	// 用两端已连通的开关注册新区段 → 成环 400
	resp, _ := doJSON(t, h, http.MethodPost, "/api/feeders/"+feederID+"/sections",
		`{"name":"环","upstreamSwitchId":"`+sws[0]+`","downstreamSwitchId":"`+sws[1]+`","lengthKm":1}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("cycle section should 400, got %d", resp.StatusCode)
	}
}

func e2eSwitches(t *testing.T, h http.Handler, feederID string) []string {
	t.Helper()
	_, payload := doJSON(t, h, http.MethodGet, "/api/feeders/"+feederID+"/topology", "")
	tp := dataOf(payload)
	list := tp["switches"].([]any)
	out := make([]string, 0, len(list))
	for _, s := range list {
		out = append(out, s.(map[string]any)["id"].(string))
	}
	return out
}

func TestRequestIDAndRecoverMiddleware(t *testing.T) {
	app, _ := newTestApp(t)
	h := app.Handler()
	req := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("X-Request-Id") == "" {
		t.Fatal("X-Request-Id should be set")
	}
}
