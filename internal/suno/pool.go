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
	pool sync.Map

	dir      string
	interval time.Duration
	logger   *slog.Logger
}

func NewWorkerPool(logger *slog.Logger, interval time.Duration, dir string) *WorkerPool {
	return &WorkerPool{dir: dir, interval: interval, logger: logger}
}

func (p *WorkerPool) Contains(idOrAlias string) bool {
	found := false
	p.pool.Range(func(key, value any) bool {
		if value.(*Worker).ID() == idOrAlias || value.(*Worker).Alias() == idOrAlias {
			found = true
			return false
		}
		return true
	})
	return found
}

func (p *WorkerPool) Remove(idOrAlias string) error {
	w := p.Get(idOrAlias)
	if w != nil {
		p.pool.Delete(w.ID())
		return w.Close()
	}
	return nil
}

func (p *WorkerPool) Get(idOrAlias string) *Worker {
	var w *Worker
	p.pool.Range(func(key, value any) bool {
		if value.(*Worker).ID() == idOrAlias || value.(*Worker).Alias() == idOrAlias {
			w = value.(*Worker)
			return false
		}
		return true
	})
	return w
}

func (p *WorkerPool) Infos() PlaylistInfos {
	var infos = make(PlaylistInfos)
	p.pool.Range(func(key, value any) bool {
		infos[value.(*Worker).Alias()] = value.(*Worker).Info()
		return true
	})
	return infos
}

func (p *WorkerPool) Add(ctx context.Context, id, alias string) error {
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

	worker, err := NewWorker(ctx, p.logger.With("id", id).With("alias", alias), id, alias, p.interval, dir)
	if err != nil {
		return err
	}

	p.pool.Store(id, worker)

	worker.Start(ctx)

	return nil
}

func (p *WorkerPool) Close() error {
	p.pool.Range(func(_, value any) bool {
		value.(*Worker).Close()
		return true
	})
	return nil
}
