package service

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// UpstreamSyncWorker 每隔 interval 对所有上游实例执行一次同步（刷新 cookie / token）。
// 默认间隔 1 分钟，防止上游登录态失效。
type UpstreamSyncWorker struct {
	upstreamRepo UpstreamRepository
	service      *UpstreamSessionService
	interval     time.Duration
	stopCh       chan struct{}
	stopOnce     sync.Once
	wg           sync.WaitGroup
}

// NewUpstreamSyncWorker 创建同步 Worker。
// interval <= 0 时默认为 1 分钟。
func NewUpstreamSyncWorker(upstreamRepo UpstreamRepository, service *UpstreamSessionService, interval time.Duration) *UpstreamSyncWorker {
	if interval <= 0 {
		interval = time.Minute
	}
	return &UpstreamSyncWorker{
		upstreamRepo: upstreamRepo,
		service:      service,
		interval:     interval,
		stopCh:       make(chan struct{}),
	}
}

// Start 启动后台同步循环（启动时立即执行一次）。
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

// Stop 停止同步循环，等待当前轮次完成。
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

	upstreams, err := w.upstreamRepo.List(ctx)
	if err != nil {
		slog.Error("[UpstreamSync] list upstreams failed", "err", err)
		return
	}
	if len(upstreams) == 0 {
		return
	}

	slog.Info("[UpstreamSync] syncing upstreams", "count", len(upstreams))

	var wg sync.WaitGroup
	for _, u := range upstreams {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			uCtx, uCancel := context.WithTimeout(ctx, 30*time.Second)
			defer uCancel()
			if err := w.service.SyncUpstream(uCtx, id); err != nil {
				slog.Warn("[UpstreamSync] sync failed", "upstream_id", id, "err", err)
			} else {
				slog.Debug("[UpstreamSync] sync ok", "upstream_id", id)
			}
		}(u.ID)
	}
	wg.Wait()
	slog.Info("[UpstreamSync] sync round complete", "count", len(upstreams))
}
