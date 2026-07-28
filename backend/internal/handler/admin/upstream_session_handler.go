package admin

import (
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// UpstreamSessionHandler 管理 Sub2API / NewAPI 上游账号的 HTTP 处理器
type UpstreamSessionHandler struct {
	upstreamService *service.UpstreamSessionService
}

// NewUpstreamSessionHandler 创建 handler 实例
func NewUpstreamSessionHandler(svc *service.UpstreamSessionService) *UpstreamSessionHandler {
	return &UpstreamSessionHandler{upstreamService: svc}
}

// ---------------------------------------------------------------------------
// DTO
// ---------------------------------------------------------------------------

// CreateUpstreamRequest 创建上游账号请求体
type CreateUpstreamRequest struct {
	Platform string `json:"platform" binding:"required,oneof=sub2api newapi"`
	Name     string `json:"name"     binding:"required"`
	BaseURL  string `json:"base_url"  binding:"required"`
	Email    string `json:"email"    binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UpstreamResponse 上游账号响应（脱敏，不返回明文密码/token）
type UpstreamResponse struct {
	ID           int64                     `json:"id"`
	Platform     string                    `json:"platform"`
	Name         string                    `json:"name"`
	BaseURL      string                    `json:"base_url"`
	Email        string                    `json:"email"`
	Health       string                    `json:"health"`
	HealthMsg    string                    `json:"health_msg,omitempty"`
	Balance      float64                   `json:"balance"`
	Groups       []service.UpstreamGroup   `json:"groups"`
	LastSyncedAt string                    `json:"last_synced_at,omitempty"`
	Status       string                    `json:"status"`
	CreatedAt    string                    `json:"created_at"`
	UpdatedAt    string                    `json:"updated_at"`
}

func upstreamToResponse(acc *service.Account) *UpstreamResponse {
	r := &UpstreamResponse{
		ID:       acc.ID,
		Platform: acc.Platform,
		Name:     acc.Name,
		Status:   acc.Status,
		CreatedAt: acc.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt: acc.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	// 从 Credentials 提取非敏感字段
	if acc.Credentials != nil {
		if v, ok := acc.Credentials["base_url"].(string); ok {
			r.BaseURL = v
		}
		if v, ok := acc.Credentials["email"].(string); ok {
			r.Email = v
		}
	}
	// 从 Extra 提取 session 数据
	if acc.Extra != nil {
		if v, ok := acc.Extra["health"].(string); ok {
			r.Health = v
		}
		if v, ok := acc.Extra["health_msg"].(string); ok {
			r.HealthMsg = v
		}
		if v, ok := acc.Extra["balance"].(float64); ok {
			r.Balance = v
		}
		if v, ok := acc.Extra["last_synced_at"].(string); ok {
			r.LastSyncedAt = v
		}
		// 解析 groups
		if raw, ok := acc.Extra["groups"]; ok && raw != nil {
			if groups, err := parseGroups(raw); err == nil {
				r.Groups = groups
			}
		}
	}
	if r.Health == "" {
		r.Health = "pending"
	}
	if r.Groups == nil {
		r.Groups = []service.UpstreamGroup{}
	}
	return r
}

// parseGroups 将 any（来自 json map）转换为 []UpstreamGroup
func parseGroups(raw any) ([]service.UpstreamGroup, error) {
	// raw 是 []interface{} 类型（来自 json.Unmarshal 到 map[string]any）
	items, ok := raw.([]interface{})
	if !ok {
		return nil, nil
	}
	groups := make([]service.UpstreamGroup, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		g := service.UpstreamGroup{}
		if v, ok := m["name"].(string); ok {
			g.Name = v
		}
		if v, ok := m["platform"].(string); ok {
			g.Platform = v
		}
		if v, ok := m["rate_multiplier"].(float64); ok {
			g.RateMultiplier = v
		}
		if v, ok := m["description"].(string); ok {
			g.Description = v
		}
		groups = append(groups, g)
	}
	return groups, nil
}

// ---------------------------------------------------------------------------
// 路由处理方法
// ---------------------------------------------------------------------------

// List 列出所有上游账号
// GET /api/v1/admin/upstreams
func (h *UpstreamSessionHandler) List(c *gin.Context) {
	ctx := c.Request.Context()
	accounts, err := h.upstreamService.ListUpstreamAccounts(ctx)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]*UpstreamResponse, 0, len(accounts))
	for i := range accounts {
		out = append(out, upstreamToResponse(&accounts[i]))
	}
	response.Success(c, out)
}

// Create 新增上游账号
// POST /api/v1/admin/upstreams
func (h *UpstreamSessionHandler) Create(c *gin.Context) {
	var req CreateUpstreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	ctx := c.Request.Context()
	account, err := h.upstreamService.CreateUpstreamAccount(ctx,
		req.Platform, req.Name, req.BaseURL, req.Email, req.Password)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "success",
		"data":    upstreamToResponse(account),
	})
}

// Get 查看单个上游账号
// GET /api/v1/admin/upstreams/:id
func (h *UpstreamSessionHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid upstream ID")
		return
	}
	ctx := c.Request.Context()
	account, err := h.upstreamService.GetUpstreamAccount(ctx, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, upstreamToResponse(account))
}

// Update 更新上游账号配置
// PUT /api/v1/admin/upstreams/:id
type UpdateUpstreamRequest struct {
	Name     string `json:"name"`
	BaseURL  string `json:"base_url"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Status   string `json:"status" binding:"omitempty,oneof=active inactive"`
}

func (h *UpstreamSessionHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid upstream ID")
		return
	}
	var req UpdateUpstreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	ctx := c.Request.Context()
	account, err := h.upstreamService.UpdateUpstreamAccount(ctx, id, req.Name, req.BaseURL, req.Email, req.Password, req.Status)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, upstreamToResponse(account))
}

// Delete 删除上游账号
// DELETE /api/v1/admin/upstreams/:id
func (h *UpstreamSessionHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid upstream ID")
		return
	}
	ctx := c.Request.Context()
	if err := h.upstreamService.DeleteUpstreamAccount(ctx, id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "upstream deleted successfully"})
}

// Sync 手动触发立即同步
// POST /api/v1/admin/upstreams/:id/sync
func (h *UpstreamSessionHandler) Sync(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid upstream ID")
		return
	}
	ctx := c.Request.Context()
	// 标记为 syncing
	_ = h.upstreamService.MarkSyncing(ctx, id)

	if err := h.upstreamService.SyncUpstream(ctx, id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	account, err := h.upstreamService.GetUpstreamAccount(ctx, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, upstreamToResponse(account))
}

// validateUpstreamPlatform 检查 account.Platform 是否为上游类型
func validateUpstreamPlatform(platform string) bool {
	return platform == domain.PlatformSub2api || platform == domain.PlatformNewapi
}
