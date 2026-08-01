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
)

// upstreamUserAgent 是发往上游实例的 HTTP 请求统一使用的 User-Agent。
// 上游站点常经 Cloudflare/WAF/Bot 防护，Go 默认的 "Go-http-client/1.1" 会被拦截并
// 返回 403 HTML 页面（body 以 '<' 开头），导致 JSON 解码报 "invalid character '<'"。
// 使用浏览器式 UA 可绕过该拦截。
const upstreamUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"

// UpstreamGroup 代表一个可用分组及其倍率
type UpstreamGroup struct {
	Name           string  `json:"name"`
	Platform       string  `json:"platform,omitempty"`
	RateMultiplier float64 `json:"rate_multiplier"`
	Description    string  `json:"description,omitempty"`
}

// UpstreamSessionService 处理 Sub2API / NewAPI 上游的登录、token 刷新、数据同步及 key 创建。
type UpstreamSessionService struct {
	upstreamRepo UpstreamRepository
	httpClient   *http.Client
}

// NewUpstreamSessionService 创建服务实例。
func NewUpstreamSessionService(upstreamRepo UpstreamRepository) *UpstreamSessionService {
	return &UpstreamSessionService{
		upstreamRepo: upstreamRepo,
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

	// 上游经 WAF/反代拦截时会返回 HTML（而非 JSON）。
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

// LoginSub2api 使用 email/password 登录 Sub2API，返回 access_token, refresh_token, expires_at。
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

// RefreshSub2api 使用 refresh_token 续签 access_token。
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

// GetGroupsSub2api 拉取 Sub2API 可用分组列表。
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

// GetBalanceSub2api 获取 Sub2API 账号余额。
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

type sub2apiCreateKeyResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Key string `json:"key"`
	} `json:"data"`
}

// CreateAPIKeySub2api 在 Sub2API 上创建 API key，返回 key 字符串。
// groupID 传 nil 时不指定分组。
func (s *UpstreamSessionService) CreateAPIKeySub2api(ctx context.Context, baseURL, accessToken, keyName string, groupID *int64) (string, error) {
	var bodyStr string
	if groupID != nil {
		bodyStr = fmt.Sprintf(`{"name":%q,"group_id":%d}`, keyName, *groupID)
	} else {
		bodyStr = fmt.Sprintf(`{"name":%q}`, keyName)
	}
	headers := map[string]string{"Authorization": "Bearer " + accessToken}
	var resp sub2apiCreateKeyResp
	statusCode, err := s.doJSON(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/v1/keys",
		strings.NewReader(bodyStr), headers, &resp)
	if err != nil {
		return "", fmt.Errorf("sub2api create key: %w", err)
	}
	if statusCode != http.StatusOK || resp.Code != 0 {
		return "", fmt.Errorf("sub2api create key failed (http %d): %s", statusCode, resp.Message)
	}
	if resp.Data.Key == "" {
		return "", fmt.Errorf("sub2api create key: empty key in response")
	}
	return resp.Data.Key, nil
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

// LoginNewapi 使用 username/password 登录 NewAPI，返回 session cookie 值 + user_id。
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

// GetGroupsNewapi 拉取 NewAPI 可用分组列表。
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
		Quota float64 `json:"quota"`
	} `json:"data"`
}

// GetBalanceNewapi 获取 NewAPI 账号 quota（除以 500000 换算为 USD）。
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
	return resp.Data.Quota / 500000, nil
}

type newapiCreateTokenResp struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Key string `json:"key"`
	} `json:"data"`
}

// CreateAPIKeyNewapi 在 NewAPI 上创建 token（API key），返回 key 字符串。
// groupName 传空字符串时不指定分组。
func (s *UpstreamSessionService) CreateAPIKeyNewapi(ctx context.Context, baseURL, sessionCookie string, userID int64, keyName, groupName string) (string, error) {
	var bodyStr string
	if groupName != "" {
		bodyStr = fmt.Sprintf(`{"name":%q,"group":%q}`, keyName, groupName)
	} else {
		bodyStr = fmt.Sprintf(`{"name":%q}`, keyName)
	}
	headers := map[string]string{
		"Cookie":       "session=" + sessionCookie,
		"New-Api-User": fmt.Sprintf("%d", userID),
	}
	var resp newapiCreateTokenResp
	statusCode, err := s.doJSON(ctx, http.MethodPost, strings.TrimRight(baseURL, "/")+"/api/user/token",
		strings.NewReader(bodyStr), headers, &resp)
	if err != nil {
		return "", fmt.Errorf("newapi create token: %w", err)
	}
	if statusCode != http.StatusOK || !resp.Success {
		return "", fmt.Errorf("newapi create token failed (http %d): %s", statusCode, resp.Message)
	}
	if resp.Data.Key == "" {
		return "", fmt.Errorf("newapi create token: empty key in response")
	}
	return resp.Data.Key, nil
}

// ---------------------------------------------------------------------------
// CreateKey — 统一入口（按 platform 路由）
// ---------------------------------------------------------------------------

// CreateKey 向上游平台创建一个与账号同名的 API key 并返回。
// groupID 用于 Sub2API（传 nil 表示不限定分组），groupName 用于 NewAPI。
// 调用前上游必须已成功同步（即持有有效 access_token / session_cookie）。
func (s *UpstreamSessionService) CreateKey(ctx context.Context, upstream *Upstream, keyName string, groupID *int64, groupName string) (string, error) {
	switch upstream.Platform {
	case "sub2api":
		if upstream.AccessToken == nil || *upstream.AccessToken == "" {
			return "", fmt.Errorf("sub2api upstream %d: no access token, sync first", upstream.ID)
		}
		return s.CreateAPIKeySub2api(ctx, upstream.BaseURL, *upstream.AccessToken, keyName, groupID)
	case "newapi":
		if upstream.SessionCookie == nil || *upstream.SessionCookie == "" {
			return "", fmt.Errorf("newapi upstream %d: no session cookie, sync first", upstream.ID)
		}
		var uid int64
		if upstream.UpstreamUserID != nil {
			uid = *upstream.UpstreamUserID
		}
		return s.CreateAPIKeyNewapi(ctx, upstream.BaseURL, *upstream.SessionCookie, uid, keyName, groupName)
	default:
		return "", fmt.Errorf("unsupported platform for CreateKey: %s", upstream.Platform)
	}
}

// ---------------------------------------------------------------------------
// 统一同步接口
// ---------------------------------------------------------------------------

// SyncUpstream 同步指定上游的 session、分组倍率、余额，自动处理 token 续签逻辑。
func (s *UpstreamSessionService) SyncUpstream(ctx context.Context, upstreamID int64) error {
	u, err := s.upstreamRepo.GetByID(ctx, upstreamID)
	if err != nil {
		return fmt.Errorf("get upstream %d: %w", upstreamID, err)
	}

	switch u.Platform {
	case "sub2api":
		return s.syncSub2api(ctx, u)
	case "newapi":
		return s.syncNewapi(ctx, u)
	default:
		return fmt.Errorf("unsupported platform: %s", u.Platform)
	}
}

func (s *UpstreamSessionService) syncSub2api(ctx context.Context, u *Upstream) error {
	now := time.Now()

	// 判断是否需要刷新 token（剩余不足 5 分钟则刷新）
	needRefresh := u.AccessToken == nil || *u.AccessToken == ""
	if !needRefresh && u.ExpiresAt != nil {
		needRefresh = u.ExpiresAt.Before(now.Add(5 * time.Minute))
	}

	var accessToken, refreshToken string
	var expiresAt time.Time

	if u.AccessToken != nil {
		accessToken = *u.AccessToken
	}
	if u.RefreshToken != nil {
		refreshToken = *u.RefreshToken
	}

	if needRefresh {
		if refreshToken != "" {
			newAccess, newExp, err := s.RefreshSub2api(ctx, u.BaseURL, refreshToken)
			if err != nil {
				slog.Warn("[UpstreamSync] sub2api refresh failed, re-login", "upstream_id", u.ID, "err", err)
				a, r, exp, lerr := s.LoginSub2api(ctx, u.BaseURL, u.Email, u.Password)
				if lerr != nil {
					return s.markError(ctx, u.ID, lerr)
				}
				accessToken, refreshToken, expiresAt = a, r, exp
			} else {
				accessToken = newAccess
				expiresAt = newExp
			}
		} else {
			a, r, exp, err := s.LoginSub2api(ctx, u.BaseURL, u.Email, u.Password)
			if err != nil {
				return s.markError(ctx, u.ID, err)
			}
			accessToken, refreshToken, expiresAt = a, r, exp
		}

		// 持久化新 token
		rt := refreshToken
		exp := expiresAt
		if err := s.upstreamRepo.SaveSession(ctx, u.ID, &accessToken, &rt, &exp, u.SessionCookie, u.UpstreamUserID); err != nil {
			slog.Warn("[UpstreamSync] sub2api save session failed", "upstream_id", u.ID, "err", err)
		}
	}

	// 拉取分组
	groups, err := s.GetGroupsSub2api(ctx, u.BaseURL, accessToken)
	if err != nil {
		slog.Warn("[UpstreamSync] sub2api get groups failed", "upstream_id", u.ID, "err", err)
	}

	// 拉取余额
	balance, err := s.GetBalanceSub2api(ctx, u.BaseURL, accessToken)
	if err != nil {
		slog.Warn("[UpstreamSync] sub2api get balance failed", "upstream_id", u.ID, "err", err)
	}

	if len(groups) > 0 || balance > 0 {
		_ = s.upstreamRepo.UpdateGroupsAndBalance(ctx, u.ID, groups, balance)
	}

	lastSync := now
	msg := (*string)(nil)
	return s.upstreamRepo.SetHealth(ctx, u.ID, "ok", msg, &lastSync)
}

func (s *UpstreamSessionService) syncNewapi(ctx context.Context, u *Upstream) error {
	now := time.Now()

	sessionCookie := ""
	if u.SessionCookie != nil {
		sessionCookie = *u.SessionCookie
	}
	var userID int64
	if u.UpstreamUserID != nil {
		userID = *u.UpstreamUserID
	}

	// 若 session 为空或验证失败则重新登录
	needLogin := sessionCookie == ""
	if !needLogin {
		if _, err := s.GetGroupsNewapi(ctx, u.BaseURL, sessionCookie, userID); err != nil {
			needLogin = true
		}
	}

	if needLogin {
		cookie, uid, err := s.LoginNewapi(ctx, u.BaseURL, u.Email, u.Password)
		if err != nil {
			return s.markError(ctx, u.ID, err)
		}
		sessionCookie, userID = cookie, uid
		if err := s.upstreamRepo.SaveSession(ctx, u.ID, nil, nil, nil, &sessionCookie, &userID); err != nil {
			slog.Warn("[UpstreamSync] newapi save session failed", "upstream_id", u.ID, "err", err)
		}
	}

	// 拉取分组
	groups, err := s.GetGroupsNewapi(ctx, u.BaseURL, sessionCookie, userID)
	if err != nil {
		slog.Warn("[UpstreamSync] newapi get groups failed", "upstream_id", u.ID, "err", err)
	}

	// 拉取余额
	balance, err := s.GetBalanceNewapi(ctx, u.BaseURL, sessionCookie, userID)
	if err != nil {
		slog.Warn("[UpstreamSync] newapi get balance failed", "upstream_id", u.ID, "err", err)
	}

	if len(groups) > 0 || balance > 0 {
		_ = s.upstreamRepo.UpdateGroupsAndBalance(ctx, u.ID, groups, balance)
	}

	lastSync := now
	msg := (*string)(nil)
	return s.upstreamRepo.SetHealth(ctx, u.ID, "ok", msg, &lastSync)
}

// markError 将同步错误持久化到 upstream.health 字段并返回原始错误。
func (s *UpstreamSessionService) markError(ctx context.Context, upstreamID int64, syncErr error) error {
	msg := syncErr.Error()
	now := time.Now()
	_ = s.upstreamRepo.SetHealth(ctx, upstreamID, "error", &msg, &now)
	return syncErr
}
