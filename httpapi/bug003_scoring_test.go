package httpapi

import (
	"net/http"
	"testing"
)

func TestBug003FaultNotFoundIs404(t *testing.T) {
	app, _ := newTestApp(t)
	h := app.Handler()
	resp, _ := doJSON(t, h, http.MethodGet, "/api/faults/FE-not-exist", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("不存在的故障应 404，实际 %d", resp.StatusCode)
	}
}

func TestBug003RestoreNoIsolateIsConflict(t *testing.T) {
	app, _ := newTestApp(t)
	h := app.Handler()
	feederID, sectionID, indID := seedFullChain(t, h)
	_ = indID
	_ = sectionID
	// 定位并进入 repairing，但不隔离，直接复电应 409
	_, p := doJSON(t, h, http.MethodPost, "/api/faults/locate", `{"feederId":"`+feederID+`"}`)
	ev := dataOf(p)["event"].(map[string]any)
	faultID := ev["id"].(string)
	_, _ = doJSON(t, h, http.MethodPost, "/api/faults/"+faultID+"/repair", `{"operator":"op"}`)
	resp, _ := doJSON(t, h, http.MethodPost, "/api/faults/"+faultID+"/restore", `{"operator":"op"}`)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("未隔离复电应 409，实际 %d", resp.StatusCode)
	}
}

func TestBug003TopologyCycleBadRequest(t *testing.T) {
	app, _ := newTestApp(t)
	h := app.Handler()
	// 建一条三节点链，再试图加一条成环区段
	_, p := doJSON(t, h, http.MethodPost, "/api/feeders", `{"name":"线","substation":"站","voltageLevel":"10kV"}`)
	feederID := dataOf(p)["id"].(string)
	_, p = doJSON(t, h, http.MethodPost, "/api/feeders/"+feederID+"/switches", `{"name":"出线","switchType":"feeder_outlet"}`)
	o := dataOf(p)["id"].(string)
	_, p = doJSON(t, h, http.MethodPost, "/api/feeders/"+feederID+"/switches", `{"name":"分段A","switchType":"sectionalizer"}`)
	a := dataOf(p)["id"].(string)
	_, p = doJSON(t, h, http.MethodPost, "/api/feeders/"+feederID+"/switches", `{"name":"分段B","switchType":"sectionalizer"}`)
	b := dataOf(p)["id"].(string)
	_, p = doJSON(t, h, http.MethodPost, "/api/feeders/"+feederID+"/sections", `{"name":"区段1","upstreamSwitchId":"`+o+`","downstreamSwitchId":"`+a+`","lengthKm":1.0}`)
	_ = p
	_, _ = doJSON(t, h, http.MethodPost, "/api/feeders/"+feederID+"/sections", `{"name":"区段2","upstreamSwitchId":"`+a+`","downstreamSwitchId":"`+b+`","lengthKm":1.0}`)
	// 加第三条区段 b->o 会成环
	resp, _ := doJSON(t, h, http.MethodPost, "/api/feeders/"+feederID+"/sections", `{"name":"区段3","upstreamSwitchId":"`+b+`","downstreamSwitchId":"`+o+`","lengthKm":1.0}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("成环区段应 400，实际 %d", resp.StatusCode)
	}
}

func TestBug003IsolateAlreadyIsolatedConflict(t *testing.T) {
	app, _ := newTestApp(t)
	h := app.Handler()
	feederID, sectionID, indID := seedFullChain(t, h)
	_ = indID
	_, p := doJSON(t, h, http.MethodPost, "/api/faults/locate", `{"feederId":"`+feederID+`"}`)
	ev := dataOf(p)["event"].(map[string]any)
	faultID := ev["id"].(string)
	_, _ = doJSON(t, h, http.MethodPost, "/api/faults/"+faultID+"/isolate", `{"operator":"op","sectionId":"`+sectionID+`"}`)
	resp, _ := doJSON(t, h, http.MethodPost, "/api/faults/"+faultID+"/isolate", `{"operator":"op","sectionId":"`+sectionID+`"}`)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("重复隔离应 409，实际 %d", resp.StatusCode)
	}
}
