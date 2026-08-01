package repository

import (
	"context"
	"encoding/json"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbupstream "github.com/Wei-Shaw/sub2api/ent/upstream"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// upstreamRepository 实现 service.UpstreamRepository。
type upstreamRepository struct {
	client *dbent.Client
}

// NewUpstreamRepository 创建 UpstreamRepository 实例。
func NewUpstreamRepository(client *dbent.Client) service.UpstreamRepository {
	return &upstreamRepository{client: client}
}

// ---------------------------------------------------------------------------
// CRUD
// ---------------------------------------------------------------------------

func (r *upstreamRepository) Create(ctx context.Context, u *service.Upstream) error {
	groups := upstreamGroupsToRaw(u.Groups)
	created, err := r.client.Upstream.Create().
		SetName(u.Name).
		SetPlatform(u.Platform).
		SetBaseURL(u.BaseURL).
		SetEmail(u.Email).
		SetPassword(u.Password).
		SetNillableAccessToken(u.AccessToken).
		SetNillableRefreshToken(u.RefreshToken).
		SetNillableExpiresAt(u.ExpiresAt).
		SetNillableSessionCookie(u.SessionCookie).
		SetNillableUpstreamUserID(u.UpstreamUserID).
		SetGroups(groups).
		SetBalance(u.Balance).
		SetHealth(healthOrDefault(u.Health)).
		SetNillableHealthMsg(u.HealthMsg).
		SetNillableLastSyncedAt(u.LastSyncedAt).
		Save(ctx)
	if err != nil {
		return translatePersistenceError(err, nil, nil)
	}
	u.ID = created.ID
	u.CreatedAt = created.CreatedAt
	u.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *upstreamRepository) GetByID(ctx context.Context, id int64) (*service.Upstream, error) {
	m, err := r.client.Upstream.Query().
		Where(dbupstream.IDEQ(id)).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrUpstreamNotFound, nil)
	}
	return upstreamEntityToService(m), nil
}

func (r *upstreamRepository) Update(ctx context.Context, u *service.Upstream) error {
	groups := upstreamGroupsToRaw(u.Groups)
	updated, err := r.client.Upstream.UpdateOneID(u.ID).
		SetName(u.Name).
		SetPlatform(u.Platform).
		SetBaseURL(u.BaseURL).
		SetEmail(u.Email).
		SetPassword(u.Password).
		SetNillableAccessToken(u.AccessToken).
		SetNillableRefreshToken(u.RefreshToken).
		SetNillableExpiresAt(u.ExpiresAt).
		SetNillableSessionCookie(u.SessionCookie).
		SetNillableUpstreamUserID(u.UpstreamUserID).
		SetGroups(groups).
		SetBalance(u.Balance).
		SetHealth(healthOrDefault(u.Health)).
		SetNillableHealthMsg(u.HealthMsg).
		SetNillableLastSyncedAt(u.LastSyncedAt).
		Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrUpstreamNotFound, nil)
	}
	u.UpdatedAt = updated.UpdatedAt
	return nil
}

func (r *upstreamRepository) Delete(ctx context.Context, id int64) error {
	now := time.Now()
	_, err := r.client.Upstream.UpdateOneID(id).
		SetDeletedAt(now).
		Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrUpstreamNotFound, nil)
	}
	return nil
}

func (r *upstreamRepository) List(ctx context.Context) ([]service.Upstream, error) {
	rows, err := r.client.Upstream.Query().
		Order(dbent.Asc(dbupstream.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]service.Upstream, 0, len(rows))
	for _, m := range rows {
		out = append(out, *upstreamEntityToService(m))
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// 定向更新（避免全量覆写带来的竞态）
// ---------------------------------------------------------------------------

func (r *upstreamRepository) SaveSession(
	ctx context.Context,
	id int64,
	accessToken, refreshToken *string,
	expiresAt *time.Time,
	sessionCookie *string,
	upstreamUserID *int64,
) error {
	b := r.client.Upstream.UpdateOneID(id)

	if accessToken != nil {
		b.SetAccessToken(*accessToken)
	} else {
		b.ClearAccessToken()
	}
	if refreshToken != nil {
		b.SetRefreshToken(*refreshToken)
	} else {
		b.ClearRefreshToken()
	}
	if expiresAt != nil {
		b.SetExpiresAt(*expiresAt)
	} else {
		b.ClearExpiresAt()
	}
	if sessionCookie != nil {
		b.SetSessionCookie(*sessionCookie)
	} else {
		b.ClearSessionCookie()
	}
	if upstreamUserID != nil {
		b.SetUpstreamUserID(*upstreamUserID)
	} else {
		b.ClearUpstreamUserID()
	}

	_, err := b.Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrUpstreamNotFound, nil)
	}
	return nil
}

func (r *upstreamRepository) SetHealth(ctx context.Context, id int64, health string, healthMsg *string, lastSyncedAt *time.Time) error {
	b := r.client.Upstream.UpdateOneID(id).
		SetHealth(health).
		SetNillableHealthMsg(healthMsg).
		SetNillableLastSyncedAt(lastSyncedAt)
	if healthMsg == nil {
		b.ClearHealthMsg()
	}
	_, err := b.Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrUpstreamNotFound, nil)
	}
	return nil
}

func (r *upstreamRepository) UpdateGroupsAndBalance(ctx context.Context, id int64, groups []service.UpstreamGroup, balance float64) error {
	raw := upstreamGroupsToRaw(groups)
	_, err := r.client.Upstream.UpdateOneID(id).
		SetGroups(raw).
		SetBalance(balance).
		Save(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrUpstreamNotFound, nil)
	}
	return nil
}

// ---------------------------------------------------------------------------
// 辅助函数
// ---------------------------------------------------------------------------

// upstreamEntityToService 将 ent 生成的 Upstream 实体转换为领域对象。
func upstreamEntityToService(m *dbent.Upstream) *service.Upstream {
	if m == nil {
		return nil
	}
	groups := upstreamGroupsFromRaw(m.Groups)
	return &service.Upstream{
		ID:             m.ID,
		Name:           m.Name,
		Platform:       m.Platform,
		BaseURL:        m.BaseURL,
		Email:          m.Email,
		Password:       m.Password,
		AccessToken:    m.AccessToken,
		RefreshToken:   m.RefreshToken,
		ExpiresAt:      m.ExpiresAt,
		SessionCookie:  m.SessionCookie,
		UpstreamUserID: m.UpstreamUserID,
		Groups:         groups,
		Balance:        m.Balance,
		Health:         m.Health,
		HealthMsg:      m.HealthMsg,
		LastSyncedAt:   m.LastSyncedAt,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

// upstreamGroupsToRaw 将领域 UpstreamGroup slice 转换为 ent 要求的 []map[string]interface{}。
func upstreamGroupsToRaw(groups []service.UpstreamGroup) []map[string]interface{} {
	if len(groups) == 0 {
		return []map[string]interface{}{}
	}
	raw := make([]map[string]interface{}, 0, len(groups))
	for _, g := range groups {
		m := map[string]interface{}{
			"name":            g.Name,
			"rate_multiplier": g.RateMultiplier,
		}
		if g.ID != 0 {
			m["id"] = g.ID
		}
		if g.Platform != "" {
			m["platform"] = g.Platform
		}
		if g.Description != "" {
			m["description"] = g.Description
		}
		raw = append(raw, m)
	}
	return raw
}

// upstreamGroupsFromRaw 将 ent JSONB 的 []map[string]interface{} 还原为领域 UpstreamGroup slice。
// 使用 JSON round-trip 以避免手写字段断言。
func upstreamGroupsFromRaw(raw []map[string]interface{}) []service.UpstreamGroup {
	if len(raw) == 0 {
		return nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var groups []service.UpstreamGroup
	if err := json.Unmarshal(b, &groups); err != nil {
		return nil
	}
	return groups
}

// healthOrDefault 确保 Health 字段有默认值。
func healthOrDefault(h string) string {
	if h == "" {
		return "pending"
	}
	return h
}
