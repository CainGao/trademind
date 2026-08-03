package middleware

import (
	"testing"
	"time"
)

func TestLoginLimiter_AllowOnFirstAttempt(t *testing.T) {
	limiter := DefaultLoginLimiter()
	if !limiter.Allow("admin") {
		t.Error("首次登录应被允许")
	}
}

func TestLoginLimiter_AllowEmptyUsername(t *testing.T) {
	limiter := DefaultLoginLimiter()
	if !limiter.Allow("") {
		t.Error("空用户名应被允许（由认证层拒绝）")
	}
	limiter.RecordFailure("") // 不应 panic
	limiter.RecordSuccess("")  // 不应 panic
}

func TestLoginLimiter_BlockAfterMaxAttempts(t *testing.T) {
	limiter := DefaultLoginLimiter() // 5 次

	for i := 0; i < 5; i++ {
		if !limiter.Allow("attacker") {
			t.Fatalf("第 %d 次尝试应被允许", i+1)
		}
		limiter.RecordFailure("attacker")
	}

	// 第 6 次应被锁定
	if limiter.Allow("attacker") {
		t.Error("达到阈值后应被锁定")
	}
}

func TestLimiter_ExactThreshold(t *testing.T) {
	limiter := DefaultLoginLimiter() // maxAttempts=5

	// 失败 4 次，第 5 次仍允许（Allow 检查时 count=4 < 5）
	for i := 0; i < 4; i++ {
		limiter.RecordFailure("user1")
	}
	if !limiter.Allow("user1") {
		t.Error("4 次失败后第 5 次应仍被允许")
	}

	// 第 5 次失败 → 触发锁定
	limiter.RecordFailure("user1")
	if limiter.Allow("user1") {
		t.Error("5 次失败后应被锁定")
	}
}

func TestLoginLimiter_SuccessClearsCounter(t *testing.T) {
	limiter := DefaultLoginLimiter()

	// 失败 3 次
	for i := 0; i < 3; i++ {
		limiter.RecordFailure("user2")
	}

	// 登录成功 → 清除
	limiter.RecordSuccess("user2")

	// 应该重新允许 5 次尝试
	for i := 0; i < 5; i++ {
		if !limiter.Allow("user2") {
			t.Fatalf("成功后第 %d 次应被允许", i+1)
		}
		limiter.RecordFailure("user2")
	}
}

func TestLoginLimiter_DifferentUsersIndependent(t *testing.T) {
	limiter := DefaultLoginLimiter()

	// user_a 被锁定
	for i := 0; i < 5; i++ {
		limiter.RecordFailure("user_a")
	}

	// user_b 不受影响
	if !limiter.Allow("user_b") {
		t.Error("不同用户不应受其他用户锁定影响")
	}
}

func TestLoginLimiter_LockoutRemaining(t *testing.T) {
	limiter := NewLoginLimiter(3, 5*time.Minute, 10*time.Minute)

	// 未锁定
	if r := limiter.LockoutRemaining("user3"); r != 0 {
		t.Errorf("未锁定用户剩余时间应为 0，得到 %v", r)
	}

	// 触发锁定
	for i := 0; i < 3; i++ {
		limiter.RecordFailure("user3")
	}

	r := limiter.LockoutRemaining("user3")
	if r <= 0 || r > 10*time.Minute {
		t.Errorf("锁定剩余时间应在 (0, 10min] 范围内，得到 %v", r)
	}
}

func TestLoginLimiter_WindowExpiryResets(t *testing.T) {
	limiter := NewLoginLimiter(3, 50*time.Millisecond, 100*time.Millisecond)

	// 失败 2 次（未锁定）
	limiter.RecordFailure("user4")
	limiter.RecordFailure("user4")

	// 等待窗口过期
	time.Sleep(60 * time.Millisecond)

	// 窗口过期后应重新允许，且计数重置
	if !limiter.Allow("user4") {
		t.Error("窗口过期后应允许登录")
	}

	// 再失败 2 次不应锁定（因为窗口已重置）
	limiter.RecordFailure("user4")
	limiter.RecordFailure("user4")
	if !limiter.Allow("user4") {
		t.Error("窗口重置后 2 次失败不应锁定")
	}
}

func TestLoginLimiter_LockoutExpiryResets(t *testing.T) {
	limiter := NewLoginLimiter(2, 1*time.Second, 50*time.Millisecond)

	// 触发锁定
	limiter.RecordFailure("user5")
	limiter.RecordFailure("user5")

	if limiter.Allow("user5") {
		t.Error("应被锁定")
	}

	// 等待锁定期过
	time.Sleep(60 * time.Millisecond)

	// 锁定结束后应允许
	if !limiter.Allow("user5") {
		t.Error("锁定期结束后应允许登录")
	}
}

func TestLoginLimiter_CleanupRemovesExpired(t *testing.T) {
	limiter := NewLoginLimiter(2, 50*time.Millisecond, 50*time.Millisecond)

	limiter.RecordFailure("user6")
	limiter.RecordFailure("user6") // 触发锁定

	time.Sleep(60 * time.Millisecond)

	limiter.Cleanup()

	// 清理后记录应不存在，Allow 返回 true
	if !limiter.Allow("user6") {
		t.Error("Cleanup 后过期记录应被清理")
	}
}

func TestLoginLimiter_CleanupKeepsActive(t *testing.T) {
	limiter := DefaultLoginLimiter()

	limiter.RecordFailure("user7")

	limiter.Cleanup() // 未过期，不应被清理

	// user7 有 1 次失败，仍应允许
	if !limiter.Allow("user7") {
		t.Error("未过期记录不应被 Cleanup 清理")
	}
}

func TestLoginLimiter_ProgressiveLockoutDuringLock(t *testing.T) {
	limiter := NewLoginLimiter(2, 1*time.Hour, 100*time.Millisecond)

	// 触发锁定
	limiter.RecordFailure("user8")
	limiter.RecordFailure("user8")

	r1 := limiter.LockoutRemaining("user8")
	if r1 <= 0 {
		t.Fatal("应被锁定")
	}

	// 锁定期间再记录失败 → 延长锁定
	time.Sleep(20 * time.Millisecond)
	limiter.RecordFailure("user8")

	r2 := limiter.LockoutRemaining("user8")
	if r2 <= r1-20*time.Millisecond {
		t.Errorf("锁定期间失败应延长锁定时间，r1=%v r2=%v", r1, r2)
	}
}
