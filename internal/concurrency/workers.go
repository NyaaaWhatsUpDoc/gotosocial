/*
   GoToSocial
   Copyright (C) 2021-2022 GoToSocial Authors admin@gotosocial.org

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package concurrency

import (
	"context"
	"errors"
	"fmt"
	"path"
	"reflect"
	"runtime"

	"codeberg.org/gruf/go-runners"
	"github.com/superseriousbusiness/gotosocial/internal/log"
)

// WorkerPool represents a proccessor for MsgType objects, using a worker pool to allocate resources.
type WorkerPool[MsgType any] struct {
	workers runners.WorkerPool
	process func(context.Context, MsgType) error
	wc, qc  int    // worker count, queue count
	prefix  string // contains type prefix for logging
}

// New returns a new WorkerPool[MsgType] with given number of workers and queue ratio,
// where the queue ratio is multiplied by no. workers to get queue size. If args < 1
// then suitable defaults are determined from the runtime's GOMAXPROCS variable.
func NewWorkerPool[MsgType any](workers int, queueRatio int) *WorkerPool[MsgType] {
	var zero MsgType

	if workers < 1 {
		// ensure sensible workers
		workers = runtime.GOMAXPROCS(0) * 4
	}
	if queueRatio < 1 {
		// ensure sensible ratio
		queueRatio = 100
	}

	// Calculate the short type string for the msg type
	msgType := reflect.TypeOf(zero).String()
	_, msgType = path.Split(msgType)

	w := &WorkerPool[MsgType]{
		process: nil,
		prefix:  fmt.Sprintf("worker.Worker[%s]", msgType),
		wc:      workers,
		qc:      workers * queueRatio,
	}

	// Log new worker creation with type prefix
	log.Infof("%s created with workers=%d queue=%d",
		w.prefix,
		workers,
		workers*queueRatio,
	)

	return w
}

// Start will attempt to start the underlying worker pool, or return error.
func (w *WorkerPool[MsgType]) Start() error {
	log.Infof("%s starting", w.prefix)

	// Check processor was set
	if w.process == nil {
		return errors.New("nil Worker.process function")
	}

	// Attempt to start pool
	if !w.workers.Start(w.wc, w.qc) {
		return errors.New("failed to start Worker pool")
	}

	return nil
}

// Stop will attempt to stop the underlying worker pool, or return error.
func (w *WorkerPool[MsgType]) Stop() error {
	log.Infof("%s stopping", w.prefix)

	// Attempt to stop pool
	if !w.workers.Stop() {
		return errors.New("failed to stop Worker pool")
	}

	return nil
}

// SetProcessor will set the Worker's processor function, which is called for each queued message.
func (w *WorkerPool[MsgType]) SetProcessor(fn func(context.Context, MsgType) error) {
	if w.process != nil {
		log.Panicf("%s Worker.process is already set", w.prefix)
	}
	w.process = fn
}

// Queue will queue provided message to be processed with there's a free worker.
func (w *WorkerPool[MsgType]) Queue(msg MsgType) {
	log.Tracef("%s queueing message (queue=%d): %+v",
		w.prefix, w.workers.Queue(), msg,
	)
	w.workers.Enqueue(func(ctx context.Context) {
		if err := w.process(ctx, msg); err != nil {
			log.Errorf("%s %v", w.prefix, err)
		}
	})
}
