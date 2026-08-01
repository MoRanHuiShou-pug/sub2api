package admin

import (
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// UpstreamSessionHandler 管理 Sub2API / NewAPI 上游账号的 HTTP 处理器
type UpstreamSessionHandler struct {
	upstreamRepo service.UpstreamRepository
	upstreamSvc  *service.UpstreamSessionService
}

// NewUpstreamSessionHandler 创建 handler 实例
func NewUpstreamSessionHandler(repo service.UpstreamRepository, svc *service.UpstreamSessionService) *UpstreamSessionHandler {
	return &UpstreamSessionHandler{upstreamRepo: repo, upstreamSvc: svc}
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
	ID           int64                   `json:"id"`
	Platform     string                  `json:"platform"`
	Name         string                  `json:"name"`
	BaseURL      string                  `json:"base_url"`
	Email        string                  `json:"email"`
	Health       string                  `json:"health"`
	HealthMsg    string                  `json:"health_msg,omitempty"`
	Balance      float64                 `json:"balance"`
	Groups       []service.UpstreamGroup `json:"groups"`
	LastSyncedAt string                  `json:"last_synced_at,omitempty"`
	CreatedAt    string                  `json:"created_at"`
	UpdatedAt    string                  `json:"updated_at"`
}

func upstreamToResponse(u *service.Upstream) *UpstreamResponse {
	r := &UpstreamResponse{
		ID:        u.ID,
		Platform:  u.Platform,
		Name:      u.Name,
		BaseURL:   u.BaseURL,
		Email:     u.Email,
		Health:    u.Health,
		Balance:   u.Balance,
		Groups:    u.Groups,
		CreatedAt: u.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt: u.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if u.HealthMsg != nil {
		r.HealthMsg = *u.HealthMsg
	}
	if u.LastSyncedAt != nil {
		r.LastSyncedAt = u.LastSyncedAt.UTC().Format("2006-01-02T15:04:05Z")
	}
	if r.Health == "" {
		r.Health = "pending"
	}
	if r.Groups == nil {
		r.Groups = []service.UpstreamGroup{}
	}
	return r
}

// ---------------------------------------------------------------------------
// 路由处理方法
// ---------------------------------------------------------------------------

// List 列出所有上游账号
// GET /api/v1/admin/upstreams
func (h *UpstreamSessionHandler) List(c *gin.Context) {
	ctx := c.Request.Context()
	upstreams, err := h.upstreamRepo.List(ctx)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	out := make([]*UpstreamResponse, 0, len(upstreams))
	for i := range upstreams {
		out = append(out, upstreamToResponse(&upstreams[i]))
	}
	response.Success(c, out)
}

// Create 新增上游账号并立即触发一次同步
// POST /api/v1/admin/upstreams
func (h *UpstreamSessionHandler) Create(c *gin.Context) {
	var req CreateUpstreamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	ctx := c.Request.Context()
	u := &service.Upstream{
		Platform: req.Platform,
		Name:     req.Name,
		BaseURL:  req.BaseURL,
		Email:    req.Email,
		Password: req.Password,
		Health:   "pending",
	}
	if err := h.upstreamRepo.Create(ctx, u); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	// 创建后立即异步同步一次（防止登录态延迟）
	go func() {
		_ = h.upstreamSvc.SyncUpstream(c.Request.Context(), u.ID)
	}()
	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "success",
		"data":    upstreamToResponse(u),
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
	u, err := h.upstreamRepo.GetByID(ctx, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, upstreamToResponse(u))
}

// UpdateUpstreamRequest 更新上游账号配置请求体
type UpdateUpstreamRequest struct {
	Name     string `json:"name"`
	BaseURL  string `json:"base_url"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Update 更新上游账号配置
// PUT /api/v1/admin/upstreams/:id
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
	u, err := h.upstreamRepo.GetByID(ctx, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if req.Name != "" {
		u.Name = req.Name
	}
	if req.BaseURL != "" {
		u.BaseURL = req.BaseURL
	}
	if req.Email != "" {
		u.Email = req.Email
	}
	if req.Password != "" {
		u.Password = req.Password
	}
	if err := h.upstreamRepo.Update(ctx, u); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, upstreamToResponse(u))
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
	if err := h.upstreamRepo.Delete(ctx, id); err != nil {
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
	_ = h.upstreamRepo.SetHealth(ctx, id, "syncing", nil, nil)

	if err := h.upstreamSvc.SyncUpstream(ctx, id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	u, err := h.upstreamRepo.GetByID(ctx, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, upstreamToResponse(u))
}
