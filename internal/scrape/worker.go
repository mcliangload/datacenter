package scrape

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"

	"datacenter/internal/config"
	"datacenter/internal/logger"
	"datacenter/internal/model"
	"datacenter/internal/service"
	"datacenter/internal/store"
)

// Worker 刮削 Worker：从任务队列领取任务并执行集合注册的刮削脚本
type Worker struct {
	tasks *store.TaskStore
	items *store.ItemStore
	cols  *store.CollectionStore
	cfg   config.ScrapeConfig
	log   *zap.Logger
}

// NewWorker 构造刮削 Worker
func NewWorker(db *mongo.Database, cfg config.ScrapeConfig) *Worker {
	return &Worker{
		tasks: store.NewTaskStore(db),
		items: store.NewItemStore(db),
		cols:  store.NewCollectionStore(db),
		cfg:   cfg,
		log:   logger.L(),
	}
}

// Run 启动 Worker 池并阻塞，直到收到退出信号
func Run(db *mongo.Database, cfg config.ScrapeConfig) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	w := NewWorker(db, cfg)
	for i := 0; i < cfg.WorkerCount; i++ {
		go w.loop(ctx, i)
	}
	w.log.Info("刮削 Worker 池已启动",
		zap.Int("worker_count", cfg.WorkerCount),
		zap.Int("timeout_seconds", cfg.TimeoutSeconds),
		zap.Int("reclaim_seconds", cfg.ReclaimSeconds))

	<-ctx.Done()
	w.log.Info("收到退出信号，刮削 Worker 池退出")
	return nil
}

func (w *Worker) loop(ctx context.Context, id int) {
	log := w.log.With(zap.Int("worker", id))
	poll := time.Duration(w.cfg.PollIntervalMs) * time.Millisecond
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		task, err := w.tasks.ClaimNext(context.Background(), w.cfg.ReclaimSeconds)
		if err != nil {
			log.Error("领取任务失败", zap.Error(err))
			time.Sleep(poll)
			continue
		}
		if task == nil {
			time.Sleep(poll)
			continue
		}
		w.process(task)
	}
}

// process 执行单个刮削任务：
//   - 成功判据：脚本产出合法 JSON 且通过集合标签定义校验（Q3/Q8）；
//   - 退出码仅记录用于诊断，不作为成功判据（Q3）。
func (w *Worker) process(task *model.ScrapeTask) {
	log := w.log.With(
		zap.String("task_id", task.ID.Hex()),
		zap.String("item_id", task.ItemID.Hex()),
		zap.String("script", task.ScriptPath),
		zap.String("data", task.DataPath))

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(w.cfg.TimeoutSeconds)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, task.ScriptPath, task.DataPath)
	var stdout, stderr limitedBuffer
	stdout.limit = w.cfg.OutputLimitBytes
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := -1
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			w.fail(task, "执行超时（超过 "+(time.Duration(w.cfg.TimeoutSeconds)*time.Second).String()+"）", exitCode, log)
			return
		}
		msg := "脚本执行失败: " + err.Error()
		if stderr.String() != "" {
			msg += " | stderr: " + stderr.String()
		}
		w.fail(task, msg, exitCode, log)
		return
	}

	// 解析 stdout JSON（标签对象）
	dec := json.NewDecoder(&stdout)
	dec.UseNumber()
	var raw map[string]interface{}
	if err := dec.Decode(&raw); err != nil {
		w.fail(task, "脚本输出不是合法 JSON: "+err.Error(), exitCode, log)
		return
	}

	// 按集合标签定义校验（刮削场景忽略未知标签）
	col, err := w.cols.FindByID(context.Background(), task.CollectionID)
	if err != nil {
		w.fail(task, "读取集合失败: "+err.Error(), exitCode, log)
		return
	}
	if col == nil {
		w.fail(task, "集合不存在（可能已被删除）", exitCode, log)
		return
	}
	validated, verr := service.ValidateAndNormalizeTags(col.TagSchema, raw, true)
	if verr != nil {
		w.fail(task, verr.Message, exitCode, log)
		return
	}

	item, err := w.items.FindByID(context.Background(), task.ItemID)
	if err != nil {
		w.fail(task, "读取数据项失败: "+err.Error(), exitCode, log)
		return
	}
	if item == nil {
		w.fail(task, "数据项不存在（可能已被删除）", exitCode, log)
		return
	}

	// 合并：手动标签始终优先（manual_tags），刮削结果仅补充手动未产出的标签。
	// 无论当前 tag_source 是 manual/mixed，重刮都不会覆盖手动标签。
	newTags := validated
	source := model.TagSourceScrape
	if len(item.ManualTags) > 0 {
		merged := make(map[string]interface{}, len(item.ManualTags)+len(validated))
		for k, v := range validated {
			merged[k] = v
		}
		for k, v := range item.ManualTags {
			merged[k] = v // 手动优先：覆盖刮削的同名标签
		}
		newTags = merged
		source = model.TagSourceMixed
	}

	now := time.Now()
	if err := w.items.UpdateFields(context.Background(), task.ItemID, map[string]interface{}{
		"tags":            newTags,
		"tag_source":      source,
		"scrape_status":   model.ItemScrapeSuccess,
		"last_scraped_at": now,
	}); err != nil {
		w.fail(task, "更新数据项失败: "+err.Error(), exitCode, log)
		return
	}
	if err := w.tasks.Complete(context.Background(), task.ID, model.TaskStatusSuccess, &exitCode, "", validated); err != nil {
		log.Error("完成任务失败", zap.Error(err))
	}
	log.Info("刮削成功",
		zap.Int("exit_code", exitCode),
		zap.Int("tag_count", len(validated)))
}

func (w *Worker) fail(task *model.ScrapeTask, msg string, exitCode int, log *zap.Logger) {
	log.Warn("刮削失败", zap.String("error", msg))
	if err := w.tasks.Complete(context.Background(), task.ID, model.TaskStatusFailed, &exitCode, msg, nil); err != nil {
		log.Error("记录任务失败状态失败", zap.Error(err))
	}
	if err := w.items.UpdateFields(context.Background(), task.ItemID, map[string]interface{}{
		"scrape_status": model.ItemScrapeFailed,
	}); err != nil {
		log.Error("更新数据项刮削状态失败", zap.Error(err))
	}
}

// limitedBuffer 限制输出大小，超限截断并追加标记
type limitedBuffer struct {
	buf       bytes.Buffer
	limit     int
	truncated bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.truncated {
		return len(p), nil
	}
	if b.limit > 0 && b.buf.Len()+len(p) > b.limit {
		room := b.limit - b.buf.Len()
		if room > 0 {
			_, _ = b.buf.Write(p[:room])
		}
		_, _ = b.buf.WriteString("... (输出超限被截断)")
		b.truncated = true
		return len(p), nil
	}
	return b.buf.Write(p)
}

func (b *limitedBuffer) Read(p []byte) (int, error) { return b.buf.Read(p) }

func (b *limitedBuffer) String() string { return b.buf.String() }
