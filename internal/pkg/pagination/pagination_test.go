package pagination

import "testing"

func TestNormalize_DefaultValues(t *testing.T) {
	page, pageSize := Normalize(1, 20)
	if page != 1 || pageSize != 20 {
		t.Errorf("Normalize(1,20) = (%d,%d), want (1,20)", page, pageSize)
	}
}

func TestNormalize_NegativePage(t *testing.T) {
	page, pageSize := Normalize(-5, 20)
	if page != 1 {
		t.Errorf("page = %d, want 1", page)
	}
	if pageSize != 20 {
		t.Errorf("pageSize = %d, want 20", pageSize)
	}
}

func TestNormalize_ZeroPage(t *testing.T) {
	page, _ := Normalize(0, 20)
	if page != 1 {
		t.Errorf("page = %d, want 1", page)
	}
}

func TestNormalize_NegativePageSize(t *testing.T) {
	_, pageSize := Normalize(1, -10)
	if pageSize != DefaultPageSize {
		t.Errorf("pageSize = %d, want %d", pageSize, DefaultPageSize)
	}
}

func TestNormalize_ZeroPageSize(t *testing.T) {
	_, pageSize := Normalize(1, 0)
	if pageSize != DefaultPageSize {
		t.Errorf("pageSize = %d, want %d", pageSize, DefaultPageSize)
	}
}

func TestNormalize_OversizedPageSize(t *testing.T) {
	_, pageSize := Normalize(1, 999999)
	if pageSize != MaxPageSize {
		t.Errorf("pageSize = %d, want %d (capped)", pageSize, MaxPageSize)
	}
}

func TestNormalize_ExactlyMaxPageSize(t *testing.T) {
	_, pageSize := Normalize(1, MaxPageSize)
	if pageSize != MaxPageSize {
		t.Errorf("pageSize = %d, want %d", pageSize, MaxPageSize)
	}
}

func TestNormalize_LargePage(t *testing.T) {
	// Large page values are valid (just means deep pagination).
	// Normalize should NOT cap page — only pageSize.
	page, _ := Normalize(99999, 20)
	if page != 99999 {
		t.Errorf("page = %d, want 99999 (page should not be capped)", page)
	}
}

func TestNormalize_BothInvalid(t *testing.T) {
	page, pageSize := Normalize(-1, -1)
	if page != 1 {
		t.Errorf("page = %d, want 1", page)
	}
	if pageSize != DefaultPageSize {
		t.Errorf("pageSize = %d, want %d", pageSize, DefaultPageSize)
	}
}
