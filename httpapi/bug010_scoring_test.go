package httpapi

import (
	"net/http"
	"testing"
)

func TestBug010PaginateClampsRange(t *testing.T) {
	items := []int{1, 2, 3}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("offset+limit 超界不应 panic: %v", r)
		}
	}()
	page, total := paginate(items, 10, 2)
	if total != 3 || len(page) != 1 {
		t.Fatalf("超界分页应安全截断，len=%d total=%d", len(page), total)
	}
}

func TestBug010ListOutagesNoExtraEntry(t *testing.T) {
	app, _ := newTestApp(t)
	h := app.Handler()
	feederID, _, _ := seedFullChain(t, h)
	_, p := doJSON(t, h, http.MethodPost, "/api/faults/locate", `{"feederId":"`+feederID+`"}`)
	ev := dataOf(p)["event"].(map[string]any)
	faultID := ev["id"].(string)
	_, _ = doJSON(t, h, http.MethodPost, "/api/faults/"+faultID+"/repair", `{"operator":"op"}`)
	_, _ = doJSON(t, h, http.MethodPost, "/api/faults/"+faultID+"/isolate", `{"operator":"op","sectionId":"`+ev["primarySectionId"].(string)+`"}`)
	_, _ = doJSON(t, h, http.MethodPost, "/api/faults/"+faultID+"/restore", `{"operator":"op"}`)
	resp, payload := doJSON(t, h, http.MethodGet, "/api/outages?limit=10", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("停电列表应 200，实际 %d", resp.StatusCode)
	}
	arr, _ := payload["data"].([]any)
	if len(arr) != 1 {
		t.Fatalf("停电列表应 1 条，实际 %d", len(arr))
	}
}

func TestBug010OutageSummaryNonNil(t *testing.T) {
	app, _ := newTestApp(t)
	h := app.Handler()
	resp, payload := doJSON(t, h, http.MethodGet, "/api/outages/summary", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("停电汇总应 200，实际 %d", resp.StatusCode)
	}
	if _, ok := payload["data"]; !ok || payload["data"] == nil {
		t.Fatal("停电汇总 data 不应为 null")
	}
}

func TestBug010OverviewRecentFaultsNonEmpty(t *testing.T) {
	app, _ := newTestApp(t)
	h := app.Handler()
	feederID, _, _ := seedFullChain(t, h)
	_, p := doJSON(t, h, http.MethodPost, "/api/faults/locate", `{"feederId":"`+feederID+`"}`)
	if dataOf(p) == nil {
		t.Fatal("定位失败")
	}
	resp, payload := doJSON(t, h, http.MethodGet, "/api/overview", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("总览应 200，实际 %d", resp.StatusCode)
	}
	ov := dataOf(payload)
	recent, _ := ov["recentFaults"].([]any)
	if len(recent) == 0 {
		t.Fatal("总览近期故障不应被截空")
	}
}

func TestBug010OverviewActiveFaultsNonEmpty(t *testing.T) {
	app, _ := newTestApp(t)
	h := app.Handler()
	feederID, _, _ := seedFullChain(t, h)
	_, p := doJSON(t, h, http.MethodPost, "/api/faults/locate", `{"feederId":"`+feederID+`"}`)
	if dataOf(p) == nil {
		t.Fatal("定位失败")
	}
	resp, payload := doJSON(t, h, http.MethodGet, "/api/overview", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("总览应 200，实际 %d", resp.StatusCode)
	}
	ov := dataOf(payload)
	active, _ := ov["activeFaults"].([]any)
	if len(active) == 0 {
		t.Fatal("总览活跃故障不应被截空")
	}
}
