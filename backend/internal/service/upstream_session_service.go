package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
)

// upstreamUserAgent 是发往上游实例的 HTTP 请求统一使用的 User-Agent。
// 上游站点常经 Cloudflare/WAF/Bot 防护，Go 默认的 "Go-http-client/1.1" 会被拦截并
// 返回 403 HTML 页面（body 以 '<' 开头），导致 JSON 解码报 "invalid character '<'"。
// 使用浏览器式 UA 可绕过该拦截。
const upstreamUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"

// UpstreamCredentials 存储在 Account.Credentials JSONB 中
type UpstreamCredentials struct {
	BaseURL  string `json:"base_url"`
	Email    string `json:"email"`
	Password string `json:"password"` // 明文；如需加密可后续替换为 password_enc
}

// UpstreamSessionData 存储在 Account.Extra JSONB 中（自动同步结果）
type UpstreamSessionData struct {
	// Sub2API
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresAt    string `json:"expires_at,omitempty"`
	// NewAPI
	SessionCookie string `json:"session_cookie,omitempty"`
	UserID        int64  `json:"user_id,omitempty"`
	// 公共
	Balance      float64         `json:"balance"`
	Groups       []UpstreamGroup `json:"groups"`
	Health       string          `json:"health"`        // "ok" | "error" | "pending"
	HealthMsg    string          `json:"health_msg,omitempty"`
	LastSyncedAt string          `json:"last_synced_at,omitempty"`
}

// UpstreamGroup 代表一个可用分组及其倍率
type UpstreamGroup struct {
	Name           string  `json:"name"`
	Platform       string  `json:"platform,omitempty"`
	RateMultiplier float64 `json:"rate_multiplier"`
	Description    string  `json:"description,omitempty"`
}

// UpstreamSessionService 处理 Sub2API / NewAPI 上游的登录、token 刷新、数据同步
type UpstreamSessionService struct {
	accountRepo AccountRepository
	httpClient  *http.Client
}

// NewUpstreamSessionService 创建服务实例
func NewUpstreamSessionService(accountRepo AccountRepository) *UpstreamSessionService {
	return &UpstreamSessionService{
		accountRepo: accountRepo,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// ---------------------------------------------------------------------------
// 公共辅助函数
// ---------------------------------------------------------------------------

func (s *UpstreamSessionService) doJSON(ctx context.Context, method, url string, body io.Reader, headers map[string]string, out any) (int, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", upstreamUserAgent)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	// 上游经 WAF/反代拦截时会返回 HTML（而非 JSON）。若响应体不是 JSON，
	// 直接 decode 会得到 "invalid character '<'" 这类无意义报错。
	// 这里先按 Content-Type 判定：非 JSON 响应读取前缀，报出清晰的错误。
	if ct := resp.Header.Get("Content-Type"); ct != "" && !strings.Contains(ct, "json") {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
		return resp.StatusCode, fmt.Errorf("upstream returned non-JSON (http %d, content-type %q): %s",
			resp.StatusCode, ct, strings.TrimSpace(string(snippet)))
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return resp.StatusCode, fmt.Errorf("decode: %w", err)
		}
	}
	return resp.StatusCode, nil
}

// ---------------------------------------------------------------------------
// Sub2API
// ---------------------------------------------------------------------------

type sub2apiLoginResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"` // seconds
	} `json:"data"`
}

// LoginSub2api 使用 email/password 登录 Sub2API，返回 access_token, refresh_token, expires_at
func (s *UpstreamSessionService) LoginSub2api(ctx context.Context, baseURL, email, password string) (accessToken, refreshToken string, expiresAt time.Time, err error) {
	payload := fmt.Sprintf(`{"email":%q,"password":%q}`, email, password)
	var resp sub2apiLoginResp
	statusCode, err := s.doJSON(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/v1/auth/login",
		strings.NewReader(payload), nil, &resp)
	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("sub2api login: %w", err)
	}
	if statusCode != http.StatusOK || resp.Code != 0 {
		return "", "", time.Time{}, fmt.Errorf("sub2api login failed (http %d): %s", statusCode, resp.Message)
	}
	exp := time.Now().Add(time.Duration(resp.Data.ExpiresIn) * time.Second)
	return resp.Data.AccessToken, resp.Data.RefreshToken, exp, nil
}

type sub2apiRefreshResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	} `json:"data"`
}

// RefreshSub2api 使用 refresh_token 续签 access_token
func (s *UpstreamSessionService) RefreshSub2api(ctx context.Context, baseURL, refreshToken string) (accessToken string, expiresAt time.Time, err error) {
	payload := fmt.Sprintf(`{"refresh_token":%q}`, refreshToken)
	var resp sub2apiRefreshResp
	statusCode, err := s.doJSON(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/v1/auth/refresh",
		strings.NewReader(payload), nil, &resp)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sub2api refresh: %w", err)
	}
	if statusCode != http.StatusOK || resp.Code != 0 {
		return "", time.Time{}, fmt.Errorf("sub2api refresh failed (http %d): %s", statusCode, resp.Message)
	}
	exp := time.Now().Add(time.Duration(resp.Data.ExpiresIn) * time.Second)
	return resp.Data.AccessToken, exp, nil
}

type sub2apiGroupsResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    []struct {
		ID             int64   `json:"id"`
		Name           string  `json:"name"`
		Platform       string  `json:"platform"`
		RateMultiplier float64 `json:"rate_multiplier"`
		Description    string  `json:"description"`
	} `json:"data"`
}

// GetGroupsSub2api 拉取 Sub2API 可用分组列表
func (s *UpstreamSessionService) GetGroupsSub2api(ctx context.Context, baseURL, accessToken string) ([]UpstreamGroup, error) {
	var resp sub2apiGroupsResp
	headers := map[string]string{"Authorization": "Bearer " + accessToken}
	statusCode, err := s.doJSON(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/v1/groups/available",
		nil, headers, &resp)
	if err != nil {
		return nil, fmt.Errorf("sub2api groups: %w", err)
	}
	if statusCode != http.StatusOK || resp.Code != 0 {
		return nil, fmt.Errorf("sub2api groups failed (http %d): %s", statusCode, resp.Message)
	}
	groups := make([]UpstreamGroup, 0, len(resp.Data))
	for _, g := range resp.Data {
		groups = append(groups, UpstreamGroup{
			Name:           g.Name,
			Platform:       g.Platform,
			RateMultiplier: g.RateMultiplier,
			Description:    g.Description,
		})
	}
	return groups, nil
}

type sub2apiMeResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Balance float64 `json:"balance"`
	} `json:"data"`
}

// GetBalanceSub2api 获取 Sub2API 账号余额
func (s *UpstreamSessionService) GetBalanceSub2api(ctx context.Context, baseURL, accessToken string) (float64, error) {
	var resp sub2apiMeResp
	headers := map[string]string{"Authorization": "Bearer " + accessToken}
	statusCode, err := s.doJSON(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/v1/auth/me",
		nil, headers, &resp)
	if err != nil {
		return 0, fmt.Errorf("sub2api balance: %w", err)
	}
	if statusCode != http.StatusOK || resp.Code != 0 {
		return 0, fmt.Errorf("sub2api balance failed (http %d): %s", statusCode, resp.Message)
	}
	return resp.Data.Balance, nil
}

// ---------------------------------------------------------------------------
// NewAPI
// ---------------------------------------------------------------------------

type newapiLoginResp struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		ID          int64  `json:"id"`
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
	} `json:"data"`
}

// LoginNewapi 使用 username/password 登录 NewAPI，返回 session cookie 值 + user_id
func (s *UpstreamSessionService) LoginNewapi(ctx context.Context, baseURL, username, password string) (sessionCookie string, userID int64, err error) {
	payload := fmt.Sprintf(`{"username":%q,"password":%q}`, username, password)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(baseURL, "/")+"/api/user/login",
		strings.NewReader(payload))
	if err != nil {
		return "", 0, fmt.Errorf("newapi login build: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", upstreamUserAgent)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("newapi login: %w", err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); ct != "" && !strings.Contains(ct, "json") {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
		return "", 0, fmt.Errorf("newapi login returned non-JSON (http %d): %s",
			resp.StatusCode, strings.TrimSpace(string(snippet)))
	}

	var loginResp newapiLoginResp
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return "", 0, fmt.Errorf("newapi login decode: %w", err)
	}
	if !loginResp.Success {
		return "", 0, fmt.Errorf("newapi login failed: %s", loginResp.Message)
	}

	// 从 Set-Cookie 头提取 session 值
	for _, c := range resp.Cookies() {
		if c.Name == "session" {
			return c.Value, loginResp.Data.ID, nil
		}
	}
	return "", 0, fmt.Errorf("newapi login: session cookie not found in response")
}

type newapiGroupsResp struct {
	Success bool                       `json:"success"`
	Data    map[string]newapiGroupInfo `json:"data"`
}
type newapiGroupInfo struct {
	Ratio float64 `json:"ratio"`
	Desc  string  `json:"desc"`
}

// GetGroupsNewapi 拉取 NewAPI 可用分组列表（需要 session cookie + user_id header）
func (s *UpstreamSessionService) GetGroupsNewapi(ctx context.Context, baseURL, sessionCookie string, userID int64) ([]UpstreamGroup, error) {
	headers := map[string]string{
		"Cookie":       "session=" + sessionCookie,
		"New-Api-User": fmt.Sprintf("%d", userID),
	}
	var resp newapiGroupsResp
	statusCode, err := s.doJSON(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/user/groups",
		nil, headers, &resp)
	if err != nil {
		return nil, fmt.Errorf("newapi groups: %w", err)
	}
	if statusCode != http.StatusOK || !resp.Success {
		return nil, fmt.Errorf("newapi groups failed (http %d)", statusCode)
	}
	groups := make([]UpstreamGroup, 0, len(resp.Data))
	for name, info := range resp.Data {
		groups = append(groups, UpstreamGroup{
			Name:           name,
			RateMultiplier: info.Ratio,
			Description:    info.Desc,
		})
	}
	return groups, nil
}

type newapiSelfResp struct {
	Success bool `json:"success"`
	Data    struct {
		Quota float64 `json:"quota"` // NewAPI internal quota units
	} `json:"data"`
}

// GetBalanceNewapi 获取 NewAPI 账号 quota（除以 500000 换算为 USD）
func (s *UpstreamSessionService) GetBalanceNewapi(ctx context.Context, baseURL, sessionCookie string, userID int64) (float64, error) {
	headers := map[string]string{
		"Cookie":       "session=" + sessionCookie,
		"New-Api-User": fmt.Sprintf("%d", userID),
	}
	var resp newapiSelfResp
	statusCode, err := s.doJSON(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/user/self",
		nil, headers, &resp)
	if err != nil {
		return 0, fmt.Errorf("newapi balance: %w", err)
	}
	if statusCode != http.StatusOK || !resp.Success {
		return 0, fmt.Errorf("newapi balance failed (http %d)", statusCode)
	}
	// 换算：quota / 500000 ≈ USD
	return resp.Data.Quota / 500000, nil
}

// ---------------------------------------------------------------------------
// 统一同步接口
// ---------------------------------------------------------------------------

// SyncUpstream 同步指定账号的 session、分组倍率、余额
// 自动处理 token 续签逻辑
func (s *UpstreamSessionService) SyncUpstream(ctx context.Context, accountID int64) error {
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return fmt.Errorf("get account %d: %w", accountID, err)
	}
	if account.Type != domain.AccountTypeUpstreamSession {
		return fmt.Errorf("account %d is not upstream_session type", accountID)
	}

	// 解析 credentials
	credsJSON, err := json.Marshal(account.Credentials)
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}
	var creds UpstreamCredentials
	if err := json.Unmarshal(credsJSON, &creds); err != nil {
		return fmt.Errorf("parse credentials: %w", err)
	}

	// 解析现有 extra（session data）
	var sessData UpstreamSessionData
	if account.Extra != nil {
		extraJSON, _ := json.Marshal(account.Extra)
		_ = json.Unmarshal(extraJSON, &sessData)
	}

	switch account.Platform {
	case domain.PlatformSub2api:
		return s.syncSub2api(ctx, account, &creds, &sessData)
	case domain.PlatformNewapi:
		return s.syncNewapi(ctx, account, &creds, &sessData)
	default:
		return fmt.Errorf("unsupported platform: %s", account.Platform)
	}
}

func (s *UpstreamSessionService) syncSub2api(ctx context.Context, account *Account, creds *UpstreamCredentials, sess *UpstreamSessionData) error {
	now := time.Now()

	// 判断是否需要刷新 token（剩余不足 5 分钟则刷新）
	needRefresh := true
	if sess.ExpiresAt != "" {
		if exp, err := time.Parse(time.RFC3339, sess.ExpiresAt); err == nil {
			needRefresh = exp.Before(now.Add(5 * time.Minute))
		}
	}
	if sess.AccessToken == "" {
		needRefresh = true
	}

	if needRefresh {
		if sess.RefreshToken != "" {
			// 尝试 refresh
			newAccess, newExp, err := s.RefreshSub2api(ctx, creds.BaseURL, sess.RefreshToken)
			if err != nil {
				slog.Warn("[UpstreamSync] sub2api refresh failed, re-login", "account_id", account.ID, "err", err)
				// fallback: re-login
				a, r, exp, lerr := s.LoginSub2api(ctx, creds.BaseURL, creds.Email, creds.Password)
				if lerr != nil {
					return s.saveError(ctx, account, lerr)
				}
				sess.AccessToken, sess.RefreshToken, newExp = a, r, exp
			} else {
				sess.AccessToken = newAccess
				sess.ExpiresAt = newExp.UTC().Format(time.RFC3339)
			}
		} else {
			a, r, exp, err := s.LoginSub2api(ctx, creds.BaseURL, creds.Email, creds.Password)
			if err != nil {
				return s.saveError(ctx, account, err)
			}
			sess.AccessToken, sess.RefreshToken = a, r
			sess.ExpiresAt = exp.UTC().Format(time.RFC3339)
		}
	}

	// 拉取分组
	groups, err := s.GetGroupsSub2api(ctx, creds.BaseURL, sess.AccessToken)
	if err != nil {
		slog.Warn("[UpstreamSync] sub2api get groups failed", "account_id", account.ID, "err", err)
	} else {
		sess.Groups = groups
	}

	// 拉取余额
	balance, err := s.GetBalanceSub2api(ctx, creds.BaseURL, sess.AccessToken)
	if err != nil {
		slog.Warn("[UpstreamSync] sub2api get balance failed", "account_id", account.ID, "err", err)
	} else {
		sess.Balance = balance
	}

	sess.Health = "ok"
	sess.HealthMsg = ""
	sess.LastSyncedAt = now.UTC().Format(time.RFC3339)
	return s.saveSession(ctx, account, sess)
}

func (s *UpstreamSessionService) syncNewapi(ctx context.Context, account *Account, creds *UpstreamCredentials, sess *UpstreamSessionData) error {
	now := time.Now()

	// NewAPI 无独立 refresh；session 有效期约 30 天，但我们每次同步都重新验证
	// 若 session cookie 为空或验证失败则重新登录
	needLogin := sess.SessionCookie == ""
	if !needLogin {
		// 验证 session 是否仍有效（发一次 /api/user/groups）
		_, err := s.GetGroupsNewapi(ctx, creds.BaseURL, sess.SessionCookie, sess.UserID)
		if err != nil {
			needLogin = true
		}
	}

	if needLogin {
		cookie, uid, err := s.LoginNewapi(ctx, creds.BaseURL, creds.Email, creds.Password)
		if err != nil {
			return s.saveError(ctx, account, err)
		}
		sess.SessionCookie = cookie
		sess.UserID = uid
	}

	// 拉取分组
	groups, err := s.GetGroupsNewapi(ctx, creds.BaseURL, sess.SessionCookie, sess.UserID)
	if err != nil {
		slog.Warn("[UpstreamSync] newapi get groups failed", "account_id", account.ID, "err", err)
	} else {
		sess.Groups = groups
	}

	// 拉取余额
	balance, err := s.GetBalanceNewapi(ctx, creds.BaseURL, sess.SessionCookie, sess.UserID)
	if err != nil {
		slog.Warn("[UpstreamSync] newapi get balance failed", "account_id", account.ID, "err", err)
	} else {
		sess.Balance = balance
	}

	sess.Health = "ok"
	sess.HealthMsg = ""
	sess.LastSyncedAt = now.UTC().Format(time.RFC3339)
	return s.saveSession(ctx, account, sess)
}

func (s *UpstreamSessionService) saveSession(ctx context.Context, account *Account, sess *UpstreamSessionData) error {
	b, _ := json.Marshal(sess)
	var extra map[string]any
	_ = json.Unmarshal(b, &extra)
	account.Extra = extra
	if err := s.accountRepo.Update(ctx, account); err != nil {
		return fmt.Errorf("save session for account %d: %w", account.ID, err)
	}
	return nil
}

func (s *UpstreamSessionService) saveError(ctx context.Context, account *Account, syncErr error) error {
	if account.Extra == nil {
		account.Extra = map[string]any{}
	}
	account.Extra["health"] = "error"
	account.Extra["health_msg"] = syncErr.Error()
	account.Extra["last_synced_at"] = time.Now().UTC().Format(time.RFC3339)
	_ = s.accountRepo.Update(ctx, account)
	return syncErr
}

// GetUpstreamAccount 获取单个上游账号
func (s *UpstreamSessionService) GetUpstreamAccount(ctx context.Context, id int64) (*Account, error) {
	return s.accountRepo.GetByID(ctx, id)
}

// UpdateUpstreamAccount 更新上游账号配置（仅更新非空字段）
func (s *UpstreamSessionService) UpdateUpstreamAccount(ctx context.Context, id int64, name, baseURL, email, password, status string) (*Account, error) {
	account, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if name != "" {
		account.Name = name
	}
	if status != "" {
		account.Status = status
	}
	if account.Credentials == nil {
		account.Credentials = map[string]any{}
	}
	if baseURL != "" {
		account.Credentials["base_url"] = baseURL
	}
	if email != "" {
		account.Credentials["email"] = email
	}
	if password != "" {
		account.Credentials["password"] = password
		// 密码更新后清空 session，下次同步会重新登录
		account.Extra = map[string]any{"health": "pending"}
	}
	if err := s.accountRepo.Update(ctx, account); err != nil {
		return nil, fmt.Errorf("update upstream account: %w", err)
	}
	return account, nil
}

// DeleteUpstreamAccount 删除上游账号
func (s *UpstreamSessionService) DeleteUpstreamAccount(ctx context.Context, id int64) error {
	return s.accountRepo.Delete(ctx, id)
}

// MarkSyncing 将账号 health 标记为 syncing（手动触发同步前调用）
func (s *UpstreamSessionService) MarkSyncing(ctx context.Context, id int64) error {
	account, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if account.Extra == nil {
		account.Extra = map[string]any{}
	}
	account.Extra["health"] = "syncing"
	return s.accountRepo.Update(ctx, account)
}
func (s *UpstreamSessionService) ListUpstreamAccounts(ctx context.Context) ([]Account, error) {
	return s.accountRepo.ListAllWithFilters(ctx, "", domain.AccountTypeUpstreamSession, "", "", AccountListGroupUngrouped, "")
}

// CreateUpstreamAccount 创建上游账号并立即同步一次
func (s *UpstreamSessionService) CreateUpstreamAccount(ctx context.Context, platform, name, baseURL, email, password string) (*Account, error) {
	creds := map[string]any{
		"base_url": baseURL,
		"email":    email,
		"password": password,
	}
	account := &Account{
		Name:     name,
		Platform: platform,
		Type:     domain.AccountTypeUpstreamSession,
		Credentials: creds,
		Extra: map[string]any{
			"health": "pending",
		},
		Priority: 0,
		Status:   domain.StatusActive,
	}
	if err := s.accountRepo.Create(ctx, account); err != nil {
		return nil, fmt.Errorf("create upstream account: %w", err)
	}
	// 立即触发同步（不阻塞返回；若失败只记录 extra.health=error）
	go func() {
		syncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.SyncUpstream(syncCtx, account.ID); err != nil {
			slog.Warn("[UpstreamSync] initial sync failed", "account_id", account.ID, "err", err)
		}
	}()
	return account, nil
}
