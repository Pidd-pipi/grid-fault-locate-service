package service

import (
	"fmt"
	"sort"
	"time"

	"example.com/grid-fault-locate-service/config"
	"example.com/grid-fault-locate-service/domain"
	"example.com/grid-fault-locate-service/store"
)

// TopologyService 线路/开关/区段拓扑维护与连通性校验。
type TopologyService struct {
	store *store.Store
	audit *AuditService
}

// NewTopologyService 构造拓扑服务。
func NewTopologyService(st *store.Store, audit *AuditService) *TopologyService {
	return &TopologyService{store: st, audit: audit}
}

// ---- 线路 Feeder ----

// FeederInput 线路创建/更新入参。
type FeederInput struct {
	Name         string              `json:"name"`
	Substation   string              `json:"substation"`
	VoltageLevel string              `json:"voltageLevel"`
	Status       domain.FeederStatus `json:"status"`
	Description  string              `json:"description"`
}

// CreateFeeder 新增线路。
func (s *TopologyService) CreateFeeder(in FeederInput, operator, requestID string) (*domain.Feeder, error) {
	if in.Name == "" {
		return nil, domain.Invalidf("feeder name is required")
	}
	if in.Substation == "" {
		return nil, domain.Invalidf("feeder substation is required")
	}
	if !config.IsValidVoltageLevel(in.VoltageLevel) {
		return nil, domain.Invalidf("feeder voltageLevel must be one of %v", config.VoltageLevels)
	}
	if in.Status == "" {
		in.Status = domain.FeederActive
	}
	now := time.Now()
	f := domain.NewFeeder(s.store.NextID("F"), in.Name, in.Substation, in.VoltageLevel, in.Status, now)
	f.Description = in.Description
	if err := f.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.CreateFeeder(f); err != nil {
		return nil, err
	}
	_ = s.audit.Record(requestID, domain.AuditFeederCreate, "feeder", f.ID, operator,
		"create feeder "+f.Name+" at "+f.Substation+" ("+f.VoltageLevel+")")
	return f, nil
}

// UpdateFeeder 更新线路基本信息。
func (s *TopologyService) UpdateFeeder(id string, in FeederInput, operator, requestID string) (*domain.Feeder, error) {
	f, err := s.store.GetFeeder(id)
	if err != nil {
		return nil, err
	}
	if in.Name == "" {
		return nil, domain.Invalidf("feeder name is required")
	}
	if in.Substation == "" {
		return nil, domain.Invalidf("feeder substation is required")
	}
	if !config.IsValidVoltageLevel(in.VoltageLevel) {
		return nil, domain.Invalidf("feeder voltageLevel must be one of %v", config.VoltageLevels)
	}
	if in.Status == "" {
		in.Status = f.Status
	}
	f.Name = in.Name
	f.Substation = in.Substation
	f.VoltageLevel = in.VoltageLevel
	f.Status = in.Status
	f.Description = in.Description
	f.Touch(time.Now())
	if err := f.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.UpdateFeeder(f); err != nil {
		return nil, err
	}
	_ = s.audit.Record(requestID, domain.AuditFeederUpdate, "feeder", f.ID, operator, "update feeder "+f.Name)
	return f, nil
}

// DeleteFeeder 删除线路，校验其下无开关/区段/故障事件。
func (s *TopologyService) DeleteFeeder(id, operator, requestID string) error {
	if _, err := s.store.GetFeeder(id); err != nil {
		return err
	}
	if n := s.store.CountSwitchesOfFeeder(id); n > 0 {
		return domain.Conflictf("feeder %s has %d switch nodes, cannot delete", id, n)
	}
	if n := s.store.CountSectionsOfFeeder(id); n > 0 {
		return domain.Conflictf("feeder %s has %d sections, cannot delete", id, n)
	}
	if n := len(s.store.ListIndicators(id, "")); n > 0 {
		return domain.Conflictf("feeder %s has %d indicators, cannot delete", id, n)
	}
	for _, f := range s.store.ListFaults("", id) {
		if f.Status.IsActive() {
			return domain.Conflictf("feeder %s has active fault %s, cannot delete", id, f.ID)
		}
	}
	if err := s.store.DeleteFeeder(id); err != nil {
		return err
	}
	_ = s.audit.Record(requestID, domain.AuditFeederDelete, "feeder", id, operator, "delete feeder "+id)
	return nil
}

// GetFeeder 查询线路。
func (s *TopologyService) GetFeeder(id string) (*domain.Feeder, error) {
	return s.store.GetFeeder(id)
}

// ListFeeders 列出线路，可按状态过滤。
func (s *TopologyService) ListFeeders(status domain.FeederStatus) []*domain.Feeder {
	list := s.store.ListFeeders()
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	if status == "" {
		return list
	}
	out := make([]*domain.Feeder, 0, len(list))
	for _, f := range list {
		if f.Status == status {
			out = append(out, f)
		}
	}
	return out
}

// ---- 开关 SwitchNode ----

// SwitchInput 开关创建/更新入参。
type SwitchInput struct {
	Name       string              `json:"name"`
	SwitchType domain.SwitchType   `json:"switchType"`
	Status     domain.SwitchStatus `json:"status"`
}

// outletOf 查找线路出线开关。
func (s *TopologyService) outletOf(feederID string) (*domain.SwitchNode, bool) {
	for _, sw := range s.store.ListSwitches(feederID) {
		if sw.IsFeederOutlet() {
			return sw, true
		}
	}
	return nil, false
}

// maxSwitchOrder 计算线路当前最大展示顺序。
func (s *TopologyService) maxSwitchOrder(feederID string) int {
	max := 0
	for _, sw := range s.store.ListSwitches(feederID) {
		if sw.Order > max {
			max = sw.Order
		}
	}
	return max
}

// AddSwitch 新增开关节点。出线开关必须唯一，且自动排到首端。
func (s *TopologyService) AddSwitch(feederID string, in SwitchInput, operator, requestID string) (*domain.SwitchNode, error) {
	if _, err := s.store.GetFeeder(feederID); err != nil {
		return nil, err
	}
	if in.Name == "" {
		return nil, domain.Invalidf("switch name is required")
	}
	if !in.SwitchType.Valid() {
		return nil, domain.Invalidf("switch type %q is invalid", in.SwitchType)
	}
	if in.Status != "" && !in.Status.Valid() {
		return nil, domain.Invalidf("switch status %q is invalid", in.Status)
	}
	now := time.Now()
	order := s.maxSwitchOrder(feederID) + 1
	if in.SwitchType == domain.SwitchTypeFeederOutlet {
		if _, ok := s.outletOf(feederID); ok {
			return nil, domain.Conflictf("feeder %s already has a feeder outlet switch", feederID)
		}
		// 出线开关排到首端，其余开关顺序 +1。
		for _, sw := range s.store.ListSwitches(feederID) {
			sw.Order++
			if err := s.store.UpdateSwitch(sw); err != nil {
				return nil, err
			}
		}
		order = 1
	}
	sw := domain.NewSwitchNode(s.store.NextID("SW"), feederID, in.Name, in.SwitchType, in.Status, order, now)
	if in.Status == "" {
		sw.Status = domain.SwitchClosed
	}
	if err := sw.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.CreateSwitch(sw); err != nil {
		return nil, err
	}
	_ = s.audit.Record(requestID, domain.AuditSwitchAdd, "switch", sw.ID, operator,
		"add switch "+sw.Name+" ("+string(sw.SwitchType)+") to feeder "+feederID)
	return sw, nil
}

// UpdateSwitch 更新开关（名称/分合状态）。类型与所属线路不可变。
func (s *TopologyService) UpdateSwitch(feederID, switchID string, in SwitchInput, operator, requestID string) (*domain.SwitchNode, error) {
	sw, err := s.store.GetSwitch(switchID)
	if err != nil {
		return nil, err
	}
	if sw.FeederID != feederID {
		return nil, domain.NotFoundf("switch %s not found in feeder %s", switchID, feederID)
	}
	if in.Name != "" {
		sw.Name = in.Name
	}
	if in.Status != "" {
		if !in.Status.Valid() {
			return nil, domain.Invalidf("switch status %q is invalid", in.Status)
		}
		sw.Status = in.Status
	}
	sw.Touch(time.Now())
	if err := sw.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.UpdateSwitch(sw); err != nil {
		return nil, err
	}
	_ = s.audit.Record(requestID, domain.AuditSwitchUpdate, "switch", sw.ID, operator,
		"update switch "+sw.Name+" to "+string(sw.Status))
	return sw, nil
}

// ToggleSwitch 分合闸操作：切换开关状态。
func (s *TopologyService) ToggleSwitch(feederID, switchID, operator, requestID string) (*domain.SwitchNode, error) {
	sw, err := s.store.GetSwitch(switchID)
	if err != nil {
		return nil, err
	}
	if sw.FeederID != feederID {
		return nil, domain.NotFoundf("switch %s not found in feeder %s", switchID, feederID)
	}
	sw.Toggle()
	sw.Touch(time.Now())
	if err := s.store.UpdateSwitch(sw); err != nil {
		return nil, err
	}
	_ = s.audit.Record(requestID, domain.AuditSwitchUpdate, "switch", sw.ID, operator,
		"toggle switch "+sw.Name+" to "+string(sw.Status))
	return sw, nil
}

// RemoveSwitch 删除开关节点，校验其非任何区段端点、且非出线开关。
func (s *TopologyService) RemoveSwitch(feederID, switchID, operator, requestID string) error {
	sw, err := s.store.GetSwitch(switchID)
	if err != nil {
		return err
	}
	if sw.FeederID != feederID {
		return domain.NotFoundf("switch %s not found in feeder %s", switchID, feederID)
	}
	if sw.IsFeederOutlet() {
		return domain.Topologyf("feeder outlet switch %s cannot be removed", switchID)
	}
	for _, sec := range s.store.ListSections(feederID) {
		eps := sec.Endpoints()
		if eps[0] == switchID || eps[1] == switchID {
			return domain.Topologyf("switch %s is an endpoint of section %s, cannot remove", switchID, sec.ID)
		}
	}
	if err := s.store.DeleteSwitch(switchID); err != nil {
		return err
	}
	// 重新编号剩余开关顺序。
	s.renumberSwitches(feederID)
	_ = s.audit.Record(requestID, domain.AuditSwitchRemove, "switch", switchID, operator, "remove switch "+sw.Name)
	return nil
}

// renumberSwitches 按当前 Order 升序重新编号。
func (s *TopologyService) renumberSwitches(feederID string) {
	list := s.store.ListSwitches(feederID)
	sort.Slice(list, func(i, j int) bool {
		if list[i].Order != list[j].Order {
			return list[i].Order < list[j].Order
		}
		return list[i].ID < list[j].ID
	})
	for i, sw := range list {
		if sw.Order != i+1 {
			sw.Order = i + 1
			_ = s.store.UpdateSwitch(sw)
		}
	}
}

// ---- 区段 FeederSection ----

// SectionInput 区段创建/更新入参。
type SectionInput struct {
	Name               string  `json:"name"`
	UpstreamSwitchID   string  `json:"upstreamSwitchId"`
	DownstreamSwitchID string  `json:"downstreamSwitchId"`
	LengthKm           float64 `json:"lengthKm"`
}

// AddSection 新增区段并执行拓扑校验：
// 端点必须为同线路开关、方向必须自出线开关向外、禁止成环、禁止悬空。
func (s *TopologyService) AddSection(feederID string, in SectionInput, operator, requestID string) (*domain.FeederSection, error) {
	if _, err := s.store.GetFeeder(feederID); err != nil {
		return nil, err
	}
	if in.Name == "" {
		return nil, domain.Invalidf("section name is required")
	}
	if in.LengthKm <= 0 {
		return nil, domain.Invalidf("section lengthKm must be positive")
	}
	up, err := s.store.GetSwitch(in.UpstreamSwitchID)
	if err != nil {
		return nil, domain.Invalidf("upstream switch: %v", err)
	}
	down, err := s.store.GetSwitch(in.DownstreamSwitchID)
	if err != nil {
		return nil, domain.Invalidf("downstream switch: %v", err)
	}
	if up.FeederID != feederID || down.FeederID != feederID {
		return nil, domain.Invalidf("section endpoints must belong to feeder %s", feederID)
	}
	if up.ID == down.ID {
		return nil, domain.Invalidf("section upstream and downstream switch must differ")
	}
	if err := s.validateSectionAddition(feederID, in.UpstreamSwitchID, in.DownstreamSwitchID); err != nil {
		return nil, err
	}
	now := time.Now()
	sec := domain.NewFeederSection(s.store.NextID("SEC"), feederID, in.Name, in.UpstreamSwitchID, in.DownstreamSwitchID, in.LengthKm, now)
	if err := sec.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.CreateSection(sec); err != nil {
		return nil, err
	}
	_ = s.audit.Record(requestID, domain.AuditSectionAdd, "section", sec.ID, operator,
		"add section "+sec.Name+" ("+in.UpstreamSwitchID+"->"+in.DownstreamSwitchID+", "+fmtLen(in.LengthKm)+")")
	return sec, nil
}

// validateSectionAddition 校验新增区段不破坏拓扑：
//  1. 端点重复/反向重复拒绝；
//  2. 两端已连通（成环）拒绝；
//  3. 两端均不可达出线开关（悬空）拒绝；
//  4. 上游端必须更靠近出线开关（方向自外）。
func (s *TopologyService) validateSectionAddition(feederID, up, down string) error {
	sections := s.store.ListSections(feederID)
	for _, sec := range sections {
		eps := sec.Endpoints()
		if (eps[0] == up && eps[1] == down) || (eps[0] == down && eps[1] == up) {
			return domain.Topologyf("section between %s and %s already exists", up, down)
		}
	}
	if domain.WouldCreateCycle(sections, up, down) {
		return domain.Topologyf("adding section %s->%s would create a cycle", up, down)
	}
	outlet, ok := s.outletOf(feederID)
	if !ok {
		return domain.Topologyf("feeder %s has no feeder outlet switch, cannot add section", feederID)
	}
	reachable := domain.ReachableFrom(sections, outlet.ID)
	upOK := reachable[up]
	downOK := reachable[down]
	if !upOK && !downOK {
		return domain.Topologyf("adding section %s->%s would be disconnected (dangling) from feeder outlet", up, down)
	}
	if !upOK && downOK {
		return domain.Topologyf("section direction must flow outward from feeder outlet: upstream %s is not reachable from outlet", up)
	}
	if upOK && downOK {
		// 两端均可达且未成环在树形拓扑下不会出现（已由成环校验拦截）。
		return domain.Topologyf("section %s->%s connects two already-reachable nodes", up, down)
	}
	return nil
}

// UpdateSection 更新区段：支持名称/长度修改与端点变更（端点变更走完整拓扑校验）。
func (s *TopologyService) UpdateSection(feederID, sectionID string, in SectionInput, operator, requestID string) (*domain.FeederSection, error) {
	sec, err := s.store.GetSection(sectionID)
	if err != nil {
		return nil, err
	}
	if sec.FeederID != feederID {
		return nil, domain.NotFoundf("section %s not found in feeder %s", sectionID, feederID)
	}
	if in.Name != "" {
		sec.Name = in.Name
	}
	if in.LengthKm > 0 {
		sec.LengthKm = in.LengthKm
	}
	if in.UpstreamSwitchID != "" && in.DownstreamSwitchID != "" &&
		(in.UpstreamSwitchID != sec.UpstreamSwitchID || in.DownstreamSwitchID != sec.DownstreamSwitchID) {
		// 端点变更：按 删除旧边 + 新增新边 校验。
		up, err := s.store.GetSwitch(in.UpstreamSwitchID)
		if err != nil {
			return nil, domain.Invalidf("upstream switch: %v", err)
		}
		down, err := s.store.GetSwitch(in.DownstreamSwitchID)
		if err != nil {
			return nil, domain.Invalidf("downstream switch: %v", err)
		}
		if up.FeederID != feederID || down.FeederID != feederID {
			return nil, domain.Invalidf("section endpoints must belong to feeder %s", feederID)
		}
		if err := s.validateSectionRepoint(feederID, sectionID, in.UpstreamSwitchID, in.DownstreamSwitchID); err != nil {
			return nil, err
		}
		sec.UpstreamSwitchID = in.UpstreamSwitchID
		sec.DownstreamSwitchID = in.DownstreamSwitchID
	}
	sec.Touch(time.Now())
	if err := sec.Validate(); err != nil {
		return nil, err
	}
	if err := s.store.UpdateSection(sec); err != nil {
		return nil, err
	}
	_ = s.audit.Record(requestID, domain.AuditSectionUpdate, "section", sec.ID, operator, "update section "+sec.Name)
	return sec, nil
}

// validateSectionRepoint 在移除 sectionID 后再新增 up->down 边做拓扑校验。
func (s *TopologyService) validateSectionRepoint(feederID, sectionID, up, down string) error {
	rest := make([]*domain.FeederSection, 0, len(s.store.ListSections(feederID)))
	for _, sec := range s.store.ListSections(feederID) {
		if sec.ID != sectionID {
			rest = append(rest, sec)
		}
	}
	if up == down {
		return domain.Invalidf("section upstream and downstream switch must differ")
	}
	for _, sec := range rest {
		eps := sec.Endpoints()
		if (eps[0] == up && eps[1] == down) || (eps[0] == down && eps[1] == up) {
			return domain.Topologyf("section between %s and %s already exists", up, down)
		}
	}
	if domain.WouldCreateCycle(rest, up, down) {
		return domain.Topologyf("repointing section %s->%s would create a cycle", up, down)
	}
	outlet, ok := s.outletOf(feederID)
	if !ok {
		return domain.Topologyf("feeder %s has no feeder outlet switch, cannot repoint section", feederID)
	}
	reachable := domain.ReachableFrom(rest, outlet.ID)
	upOK, downOK := reachable[up], reachable[down]
	if !upOK && !downOK {
		return domain.Topologyf("repointing section %s->%s would be disconnected (dangling) from feeder outlet", up, down)
	}
	if !upOK && downOK {
		return domain.Topologyf("section direction must flow outward from feeder outlet")
	}
	if upOK && downOK {
		return domain.Topologyf("repointing section %s->%s connects two already-reachable nodes", up, down)
	}
	return nil
}

// RemoveSection 删除区段：校验隔离状态与删除后连通性（不得悬空下游开关）。
func (s *TopologyService) RemoveSection(feederID, sectionID, operator, requestID string) error {
	sec, err := s.store.GetSection(sectionID)
	if err != nil {
		return err
	}
	if sec.FeederID != feederID {
		return domain.NotFoundf("section %s not found in feeder %s", sectionID, feederID)
	}
	if sec.Isolated {
		return domain.Topologyf("section %s is isolated, resolve isolation before removing", sectionID)
	}
	rest := make([]*domain.FeederSection, 0, len(s.store.ListSections(feederID)))
	for _, sc := range s.store.ListSections(feederID) {
		if sc.ID != sectionID {
			rest = append(rest, sc)
		}
	}
	if len(rest) > 0 {
		outlet, ok := s.outletOf(feederID)
		if !ok {
			return domain.Topologyf("feeder %s has no feeder outlet switch", feederID)
		}
		reachable := domain.ReachableFrom(rest, outlet.ID)
		// 删除后不允许存在悬空区段：剩余区段两端必须仍可达出线开关。
		// 只剩孤立节点（无任何区段挂接）的开关允许单独保留，可另行删除。
		for _, sc := range rest {
			eps := sc.Endpoints()
			if !reachable[eps[0]] || !reachable[eps[1]] {
				return domain.Topologyf("removing section %s would leave section %s disconnected (dangling) from feeder outlet", sectionID, sc.ID)
			}
		}
	}
	if err := s.store.DeleteSection(sectionID); err != nil {
		return err
	}
	_ = s.audit.Record(requestID, domain.AuditSectionRemove, "section", sectionID, operator, "remove section "+sec.Name)
	return nil
}

// ---- 拓扑视图 ----

// Topology 单个线路的完整拓扑视图。
type Topology struct {
	Feeder       *domain.Feeder           `json:"feeder"`
	Switches     []*domain.SwitchNode     `json:"switches"`
	Sections     []*domain.FeederSection  `json:"sections"`
	SectionOrder []string                 `json:"sectionOrder"`
	Indicators   []*domain.FaultIndicator `json:"indicators"`
}

// GetTopology 获取线路拓扑视图（开关按展示顺序、区段含下游顺序）。
func (s *TopologyService) GetTopology(feederID string) (*Topology, error) {
	f, err := s.store.GetFeeder(feederID)
	if err != nil {
		return nil, err
	}
	switches := s.store.ListSwitches(feederID)
	sort.Slice(switches, func(i, j int) bool {
		if switches[i].Order != switches[j].Order {
			return switches[i].Order < switches[j].Order
		}
		return switches[i].ID < switches[j].ID
	})
	sections := s.store.ListSections(feederID)
	order := make([]string, 0)
	if outlet, ok := s.outletOf(feederID); ok {
		order = domain.DownstreamOrder(sections, outlet.ID)
	}
	return &Topology{
		Feeder:       f,
		Switches:     switches,
		Sections:     sections,
		SectionOrder: order,
		Indicators:   s.store.ListIndicators(feederID, ""),
	}, nil
}

func fmtLen(km float64) string {
	return fmt.Sprintf("%.2fkm", km)
}
