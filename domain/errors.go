// Package domain 定义配电网故障定位与复电服务的领域模型、
// 业务枚举与核心业务规则（状态机、拓扑约束等）。
package domain

import (
	"errors"
	"fmt"
)

// 领域层统一错误哨兵。服务层通过 fmt.Errorf("%w: ...", ErrXxx) 包装后返回，
// httpapi 的错误处理中间件据此映射 HTTP 状态码与业务错误码。
var (
	// ErrNotFound 目标实体不存在。
	ErrNotFound = errors.New("not found")
	// ErrConflict 数据冲突（重复创建、状态冲突等）。
	ErrConflict = errors.New("conflict")
	// ErrInvalid 参数或字段非法。
	ErrInvalid = errors.New("invalid argument")
	// ErrStateTransition 状态机不允许的迁移。
	ErrStateTransition = errors.New("invalid state transition")
	// ErrTopologyInvalid 拓扑约束不满足（成环、悬空、连通性破坏等）。
	ErrTopologyInvalid = errors.New("invalid topology")
	// ErrNoFaultSignal 没有可用的故障翻牌信号，无法定位。
	ErrNoFaultSignal = errors.New("no fault signal")
	// ErrNotIsolated 复电前未完成故障区段隔离确认。
	ErrNotIsolated = errors.New("fault not isolated")
	// ErrSuspiciousOnly 定位依据中仅剩可疑信号，需人工核验。
	ErrSuspiciousOnly = errors.New("only suspicious signals")
)

// Invalidf 构造一个携带具体原因的非法参数错误。
func Invalidf(format string, args ...any) error {
	return fmt.Errorf("%v: %s", ErrInvalid, fmt.Sprintf(format, args...))
}

// Conflictf 构造一个携带具体原因的冲突错误。
func Conflictf(format string, args ...any) error {
	return fmt.Errorf("%v: %s", ErrConflict, fmt.Sprintf(format, args...))
}

// Topologyf 构造一个携带具体原因的拓扑校验错误。
func Topologyf(format string, args ...any) error {
	return fmt.Errorf("%v: %s", ErrTopologyInvalid, fmt.Sprintf(format, args...))
}

// Statef 构造一个携带具体原因的状态迁移错误。
func Statef(format string, args ...any) error {
	return fmt.Errorf("%v: %s", ErrStateTransition, fmt.Sprintf(format, args...))
}

// NotFoundf 构造一个携带具体原因的未找到错误。
func NotFoundf(format string, args ...any) error {
	return fmt.Errorf("%v: %s", ErrNotFound, fmt.Sprintf(format, args...))
}
