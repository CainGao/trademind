// Package middleware — 安全中间件。
//
// LoginLimiter: 防暴力破解登录限流器。
// 追踪每个用户名的失败尝试，超过阈值后临时锁定。
// 内存存储（私有化部署足够，重启即清零）。
package middleware

import (
	"log"
	"sync"
	"time"
)

// LoginLimiter 登录防暴力破解限流器。
// 策略：同一用户名在 window 内失败 maxAttempts 次后锁定 lockout 时长。
// 锁定期间拒绝所有登录尝试；锁定结束后自动重置计数。
type LoginLimiter struct {
	mu          sync.Mutex
	maxAttempts int           // 窗口内最大失败次数
	window      time.Duration // 计数窗口
	lockout     time.Duration // 锁定时长
	attempts    map[string]*loginState
}

// loginState 单个用户名的登录尝试状态。
type loginState struct {
	count       int       // 窗口内失败次数
	firstFail   time.Time // 窗口内首次失败时间
	lockedUntil time.Time // 锁定截止时间（零值表示未锁定）
}

// DefaultLoginLimiter 创建默认配置的登录限流器。
// 默认：5 次失败 / 5 分钟窗口 → 锁定 5 分钟。
func DefaultLoginLimiter() *LoginLimiter {
	return &LoginLimiter{
		maxAttempts: 5,
		window:      5 * time.Minute,
		lockout:     5 * time.Minute,
		attempts:    make(map[string]*loginState),
	}
}

// NewLoginLimiter 创建自定义配置的登录限流器。
func NewLoginLimiter(maxAttempts int, window, lockout time.Duration) *LoginLimiter {
	return &LoginLimiter{
		maxAttempts: maxAttempts,
		window:      window,
		lockout:     lockout,
		attempts:    make(map[string]*loginState),
	}
}

// Allow 检查指定用户名是否允许登录（未被锁定）。
// 如果窗口已过或锁定已解除，自动清理过期记录。
func (l *LoginLimiter) Allow(username string) bool {
	if username == "" {
		return true // 空用户名不限流（会被认证层拒绝）
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	s, ok := l.attempts[username]
	if !ok {
		return true
	}

	now := time.Now()

	// 锁定期内拒绝
	if now.Before(s.lockedUntil) {
		return false
	}

	// 锁定已解除（lockedUntil 非零且已过）→ 清理记录，重新开始
	if !s.lockedUntil.IsZero() && now.After(s.lockedUntil) {
		delete(l.attempts, username)
		return true
	}

	// 计数窗口已过 → 重置
	if now.Sub(s.firstFail) > l.window {
		delete(l.attempts, username)
		return true
	}

	return s.count < l.maxAttempts
}

// RecordFailure 记录一次失败登录尝试。
// 达到阈值时自动设置锁定。
func (l *LoginLimiter) RecordFailure(username string) {
	if username == "" {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	s, ok := l.attempts[username]

	if !ok || now.Sub(s.firstFail) > l.window {
		// 新窗口
		l.attempts[username] = &loginState{
			count:     1,
			firstFail: now,
		}
		return
	}

	s.count++
	if s.count >= l.maxAttempts {
		s.lockedUntil = now.Add(l.lockout)
	}
}

// RecordSuccess 登录成功时清除该用户名的计数。
func (l *LoginLimiter) RecordSuccess(username string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, username)
}

// LockoutRemaining 返回剩余锁定时长（0 表示未锁定）。
func (l *LoginLimiter) LockoutRemaining(username string) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()

	s, ok := l.attempts[username]
	if !ok {
		return 0
	}

	remaining := time.Until(s.lockedUntil)
	if remaining <= 0 {
		return 0
	}
	return remaining
}

// Cleanup 清理所有过期的记录（窗口已过且锁定已解除）。
// 可由定时器周期调用防止内存缓慢增长。
func (l *LoginLimiter) Cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for k, s := range l.attempts {
		if now.After(s.lockedUntil) && now.Sub(s.firstFail) > l.window {
			delete(l.attempts, k)
		}
	}
}

// StartCleanup 启动后台定期清理 goroutine，返回 stop 函数。
// 典型用法：stop := limiter.StartCleanup(10 * time.Minute); defer stop()
func (l *LoginLimiter) StartCleanup(interval time.Duration) func() {
	ticker := time.NewTicker(interval)
	done := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[login-limiter] 清理 goroutine panic（已恢复）: %v", r)
			}
		}()
		for {
			select {
			case <-ticker.C:
				l.Cleanup()
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()
	return func() { close(done) }
}
