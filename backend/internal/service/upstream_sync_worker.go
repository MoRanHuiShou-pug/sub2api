package service

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// UpstreamSyncWorker 每隔 interval 对所有 upstream_session 账号执行一次同步
// 参照 AccountExpiryService 的 Start/Stop 模式
type UpstreamSyncWorker struct {
	service  *UpstreamSessionService
	interval time.Duration
	stopCh   chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
}

// NewUpstreamSyncWorker 创建同步 Worker（默认 5 分钟间隔）
func NewUpstreamSyncWorker(service *UpstreamSessionService, interval time.Duration) *UpstreamSyncWorker {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	return &UpstreamSyncWorker{
		service:  service,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Start 启动后台同步循环（启动时立即执行一次）
func (w *UpstreamSyncWorker) Start() {
	if w == nil || w.service == nil {
		return
	}
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		w.runOnce()
		for {
			select {
			case <-ticker.C:
				w.runOnce()
			case <-w.stopCh:
				return
			}
		}
	}()
}

// Stop 停止同步循环，等待当前轮次完成
func (w *UpstreamSyncWorker) Stop() {
	if w == nil {
		return
	}
	w.stopOnce.Do(func() {
		close(w.stopCh)
	})
	w.wg.Wait()
}

func (w *UpstreamSyncWorker) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	accounts, err := w.service.ListUpstreamAccounts(ctx)
	if err != nil {
		slog.Error("[UpstreamSync] list accounts failed", "err", err)
		return
	}
	if len(accounts) == 0 {
		return
	}

	slog.Info("[UpstreamSync] syncing upstream accounts", "count", len(accounts))

	// 并发同步，每个账号独立超时
	var wg sync.WaitGroup
	for _, acc := range accounts {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			accCtx, accCancel := context.WithTimeout(ctx, 30*time.Second)
			defer accCancel()
			if err := w.service.SyncUpstream(accCtx, id); err != nil {
				slog.Warn("[UpstreamSync] sync failed", "account_id", id, "err", err)
			} else {
				slog.Debug("[UpstreamSync] sync ok", "account_id", id)
			}
		}(acc.ID)
	}
	wg.Wait()
	slog.Info("[UpstreamSync] sync round complete", "count", len(accounts))
}
