package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// ErrUpstreamNotFound is returned when an upstream record cannot be found.
var ErrUpstreamNotFound = infraerrors.NotFound("UPSTREAM_NOT_FOUND", "upstream not found")

// Upstream 代表一条上游实例记录（独立表，区别于账号）。
// 上游登录凭据（BaseURL/Email/Password）在此持久化，
// Session 令牌（AccessToken/RefreshToken/SessionCookie）由 UpstreamSessionService 更新。
type Upstream struct {
	ID             int64
	Name           string
	Platform       string // "sub2api" | "newapi"
	BaseURL        string
	Email          string
	Password       string // 明文存储，敏感字段
	AccessToken    *string
	RefreshToken   *string
	ExpiresAt      *time.Time
	SessionCookie  *string
	UpstreamUserID *int64
	Groups         []UpstreamGroup // 同步后的可用分组列表
	Balance        float64
	Health         string  // "pending" | "syncing" | "ok" | "error"
	HealthMsg      *string // 最近一次错误信息（Health="error" 时非空）
	LastSyncedAt   *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// UpstreamRepository 定义对 upstreams 表的数据访问操作。
type UpstreamRepository interface {
	// Create 持久化一条新上游记录，回写 ID/CreatedAt/UpdatedAt。
	Create(ctx context.Context, u *Upstream) error

	// GetByID 按主键查询（自动过滤软删除行）。
	GetByID(ctx context.Context, id int64) (*Upstream, error)

	// Update 全量更新一条上游记录（仅更新业务字段，不覆盖 session 令牌）。
	Update(ctx context.Context, u *Upstream) error

	// Delete 软删除（设置 deleted_at）。
	Delete(ctx context.Context, id int64) error

	// List 返回所有未删除的上游记录，按 ID 升序。
	List(ctx context.Context) ([]Upstream, error)

	// SaveSession 在成功登录/刷新后原子写入 session 令牌字段。
	// 根据 platform 的不同，只有对应的字段会被写入非 nil 值，
	// 其余字段传 nil 时会被清除。
	SaveSession(ctx context.Context, id int64,
		accessToken, refreshToken *string,
		expiresAt *time.Time,
		sessionCookie *string,
		upstreamUserID *int64,
	) error

	// SetHealth 更新 health/health_msg/last_synced_at 三个字段。
	SetHealth(ctx context.Context, id int64, health string, healthMsg *string, lastSyncedAt *time.Time) error

	// UpdateGroupsAndBalance 同步成功后更新分组列表和余额。
	UpdateGroupsAndBalance(ctx context.Context, id int64, groups []UpstreamGroup, balance float64) error
}
