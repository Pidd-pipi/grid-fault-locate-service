package httpapi

import (
	"net/url"
	"strconv"

	"example.com/grid-fault-locate-service/domain"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

// parseLimitOffset 解析 ?limit= 与 ?offset=，并应用默认值与上限。
func parseLimitOffset(q url.Values) (limit, offset int, err error) {
	limit = defaultPageSize
	offset = 0
	if v := q.Get("limit"); v != "" {
		n, perr := strconv.Atoi(v)
		if perr != nil || n < 0 {
			return 0, 0, domain.Invalidf("limit must be a non-negative integer, got %q", v)
		}
		if n > maxPageSize {
			n = maxPageSize
		}
		if n == 0 {
			n = defaultPageSize
		}
		limit = n
	}
	if v := q.Get("offset"); v != "" {
		n, perr := strconv.Atoi(v)
		if perr != nil || n < 0 {
			return 0, 0, domain.Invalidf("offset must be a non-negative integer, got %q", v)
		}
		offset = n
	}
	return limit, offset, nil
}

// parseOptionalBool 解析可选的布尔查询参数；缺省返回 false，非法值返回 400。
func parseOptionalBool(q url.Values, key string) (bool, error) {
	vs, ok := q[key]
	if !ok || len(vs) == 0 || vs[0] == "" {
		return false, nil
	}
	v := vs[0]
	switch v {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, domain.Invalidf("%s must be true or false, got %q", key, v)
	}
}

// paginate 对已排序的全量列表做 limit/offset 切片，返回当前页与总数。
func paginate[T any](items []T, limit, offset int) ([]T, int) {
	total := len(items)
	if offset >= total {
		return make([]T, 0), total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	out := make([]T, end-offset)
	copy(out, items[offset:end])
	return out, total
}
