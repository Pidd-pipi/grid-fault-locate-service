package domain

import "time"

// FeederSection 线路区段实体：区段两端必须是开关节点（上游/下游）。
type FeederSection struct {
	ID                 string    `json:"id"`
	FeederID           string    `json:"feederId"`
	Name               string    `json:"name"`
	UpstreamSwitchID   string    `json:"upstreamSwitchId"`
	DownstreamSwitchID string    `json:"downstreamSwitchId"`
	LengthKm           float64   `json:"lengthKm"`
	IsCandidate        bool      `json:"isCandidate"`
	Isolated           bool      `json:"isolated"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

// Validate 校验区段字段合法性（端点必须非空、长度必须为正）。
func (s *FeederSection) Validate() error {
	if s.Name == "" {
		return Invalidf("section name is required")
	}
	if s.UpstreamSwitchID == "" || s.DownstreamSwitchID == "" {
		return Invalidf("section requires both upstream and downstream switch nodes")
	}
	if s.UpstreamSwitchID == s.DownstreamSwitchID {
		return Invalidf("section upstream and downstream switch must differ")
	}
	if s.LengthKm <= 0 {
		return Invalidf("section lengthKm must be positive")
	}
	return nil
}

// Endpoints 返回区段两端开关 ID。
func (s *FeederSection) Endpoints() [2]string {
	return [2]string{s.UpstreamSwitchID, s.DownstreamSwitchID}
}

// OtherEndpoint 返回区段相对某端点的另一端。
func (s *FeederSection) OtherEndpoint(switchID string) (string, bool) {
	switch switchID {
	case s.UpstreamSwitchID:
		return s.DownstreamSwitchID, true
	case s.DownstreamSwitchID:
		return s.UpstreamSwitchID, true
	}
	return "", false
}

// Touch 更新区段 UpdatedAt 时间戳。
func (s *FeederSection) Touch(now time.Time) {
	s.UpdatedAt = now
}

// NewFeederSection 构造区段实体。
func NewFeederSection(id, feederID, name, up, down string, lengthKm float64, now time.Time) *FeederSection {
	return &FeederSection{
		ID:                 id,
		FeederID:           feederID,
		Name:               name,
		UpstreamSwitchID:   up,
		DownstreamSwitchID: down,
		LengthKm:           lengthKm,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

// ---- 纯拓扑图工具函数（无存储依赖，便于单测） ----

var orderScratch = make([]string, 0, 64)
var neighborScratch = make([]string, 0, 16)
var reachSeenScratch map[string]bool

// Edge 表示一条无向边（两端开关 ID）。
type Edge [2]string

// BuildAdjacency 根据区段集合构建无向邻接表。
func BuildAdjacency(sections []*FeederSection) map[string][]string {
	adj := make(map[string][]string)
	add := func(u, v string) {
		adj[u] = append(adj[u], v)
		adj[v] = append(adj[v], u)
	}
	for _, sec := range sections {
		add(sec.UpstreamSwitchID, sec.DownstreamSwitchID)
	}
	return adj
}

// ReachableFrom 返回从起点出发，沿区段无向边可达的开关节点集合（含起点）。
func ReachableFrom(sections []*FeederSection, start string) map[string]bool {
	adj := BuildAdjacency(sections)
	if reachSeenScratch == nil {
		reachSeenScratch = map[string]bool{start: true}
	}
	seen := reachSeenScratch
	seen[start] = true
	queue := []string{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, next := range adj[cur] {
			if !seen[next] {
				seen[next] = true
				queue = append(queue, next)
			}
		}
	}
	return seen
}

// IsConnectedBetween 判断两个开关节点之间是否存在区段路径。
func IsConnectedBetween(sections []*FeederSection, a, b string) bool {
	if a == b {
		return true
	}
	return ReachableFrom(sections, a)[b]
}

// WouldCreateCycle 判断在现有区段图上新增边 a-b 是否构成环：
// 若 a、b 已通过既有路径连通，则新增边必成环。
func WouldCreateCycle(sections []*FeederSection, a, b string) bool {
	if a == b {
		return true
	}
	return IsConnectedBetween(sections, a, b)
}

// DownstreamOrder 从出线开关出发，沿区段有向边（上游→下游）做 DFS，
// 返回按下游顺序排列的区段 ID 列表。无法从出线开关到达的区段排在末尾。
func DownstreamOrder(sections []*FeederSection, outletSwitchID string) []string {
	byUp := make(map[string][]*FeederSection)
	for _, sec := range sections {
		byUp[sec.UpstreamSwitchID] = append(byUp[sec.UpstreamSwitchID], sec)
	}
	seen := make(map[string]bool)
	orderScratch = orderScratch[:0]
	order := orderScratch
	var dfs func(switchID string)
	dfs = func(switchID string) {
		for _, sec := range byUp[switchID] {
			if seen[sec.ID] {
				continue
			}
			seen[sec.ID] = true
			order = append(order, sec.ID)
			dfs(sec.DownstreamSwitchID)
		}
	}
	dfs(outletSwitchID)
	// 追加所有未被出线开关可达的区段（理论上校验后不应存在）。
	for _, sec := range sections {
		if !seen[sec.ID] {
			seen[sec.ID] = true
			order = append(order, sec.ID)
		}
	}
	return order
}

// NeighborSectionIDs 返回与指定区段共享端点（上/下游开关）的相邻区段 ID 集合。
func NeighborSectionIDs(sections []*FeederSection, sectionID string) []string {
	var target *FeederSection
	for _, sec := range sections {
		if sec.ID == sectionID {
			target = sec
			break
		}
	}
	if target == nil {
		return nil
	}
	eps := target.Endpoints()
	neighborScratch = neighborScratch[:0]
	neighbors := neighborScratch
	for _, sec := range sections {
		if sec.ID == sectionID {
			continue
		}
		secEps := sec.Endpoints()
		if secEps[0] == eps[0] || secEps[1] == eps[0] || secEps[0] == eps[1] || secEps[1] == eps[1] {
			neighbors = append(neighbors, sec.ID)
		}
	}
	return neighbors
}
