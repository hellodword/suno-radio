package suno

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"sync"
	"time"
)

type WorkerPool struct {
	ctx context.Context

	lock sync.Mutex
	pool []*Worker

	dir      string
	interval time.Duration
	logger   *slog.Logger
}

func NewWorkerPool(ctx context.Context, logger *slog.Logger, interval time.Duration, dir string) *WorkerPool {
	return &WorkerPool{ctx: ctx, dir: dir, interval: interval, logger: logger}
}

func (p *WorkerPool) Contains(id string) bool {
	for i := range p.pool {
		if id == p.pool[i].ID() {
			return true
		}
	}
	return false
}

func (p *WorkerPool) Get(id string) *Worker {
	for i := range p.pool {
		if id == p.pool[i].ID() {
			return p.pool[i]
		}
	}
	return nil
}

func (p *WorkerPool) IDs() []string {
	var ids = make([]string, 0)
	for i := range p.pool {
		ids = append(ids, p.pool[i].ID())
	}
	return ids
}

func (p *WorkerPool) Add(id string) error {
	if p.Contains(id) {
		return nil
	}

	dir := path.Join(p.dir, id)

	stat, err := os.Stat(dir)
	if errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	} else if stat != nil && !stat.IsDir() {
		err = fmt.Errorf("not dir %s", dir)
		return err
	}

	worker, err := NewWorker(p.ctx, p.logger.With("id", id), id, p.interval, dir)
	if err != nil {
		return err
	}

	p.lock.Lock()
	p.pool = append(p.pool, worker)
	p.lock.Unlock()

	worker.Start()

	return nil
}
