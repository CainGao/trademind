// Package pagination provides centralized pagination parameter normalization.
//
// All List endpoints accept ?page=N&page_size=M from the URL. Without bounds
// checking, a client can request page_size=999999 (memory exhaustion / DoS)
// or page=-1 (negative OFFSET → SQLite error or undefined behavior).
//
// Normalize() clamps both values to safe ranges in one place, eliminating
// the need for scattered inline guards across handlers and repos.
package pagination

const (
	// DefaultPageSize is used when page_size is missing or invalid.
	DefaultPageSize = 20
	// MaxPageSize caps page_size to prevent unbounded result sets.
	MaxPageSize = 100
)

// Normalize clamps page and pageSize to safe ranges:
//   - page < 1 → 1
//   - pageSize < 1 → DefaultPageSize (20)
//   - pageSize > MaxPageSize (100) → MaxPageSize
//
// Call this at the handler layer (after parsing query params) OR at the repo
// layer as defense-in-depth. Both layers guarding is the recommended pattern
// (b2c_repo and agent_repo already do this).
func Normalize(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return page, pageSize
}
