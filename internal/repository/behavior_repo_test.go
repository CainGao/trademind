package repository

import "testing"

// ===== clampDays（统计查询天数边界校验，gotcha #59）=====

func TestClampDays_DefaultOnZero(t *testing.T) {
	if d := clampDays(0); d != 14 {
		t.Errorf("clampDays(0) = %d, want 14（默认回退）", d)
	}
}

func TestClampDays_DefaultOnNegative(t *testing.T) {
	if d := clampDays(-1); d != 14 {
		t.Errorf("clampDays(-1) = %d, want 14（负数回退）", d)
	}
	if d := clampDays(-99999); d != 14 {
		t.Errorf("clampDays(-99999) = %d, want 14", d)
	}
}

func TestClampDays_DefaultOnTooLarge(t *testing.T) {
	// ?days=99999999 会导致全表扫描，必须被拦截
	if d := clampDays(maxStatsDays + 1); d != 14 {
		t.Errorf("clampDays(%d) = %d, want 14（越界上界回退）", maxStatsDays+1, d)
	}
	if d := clampDays(99999999); d != 14 {
		t.Errorf("clampDays(99999999) = %d, want 14（注入式超大值回退）", d)
	}
}

func TestClampDays_PassesValidRange(t *testing.T) {
	valid := []int{1, 7, 14, 30, 60, maxStatsDays}
	for _, d := range valid {
		if got := clampDays(d); got != d {
			t.Errorf("clampDays(%d) = %d, want %d（合法值不应被修改）", d, got, d)
		}
	}
}

func TestClampDays_BoundaryValues(t *testing.T) {
	// 边界值：恰好 1 和 maxStatsDays(90) 是合法的
	if d := clampDays(1); d != 1 {
		t.Errorf("clampDays(1) = %d, want 1（下界含）", d)
	}
	if d := clampDays(maxStatsDays); d != maxStatsDays {
		t.Errorf("clampDays(%d) = %d, want %d（上界含）", maxStatsDays, d, maxStatsDays)
	}
}
