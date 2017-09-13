// Copyright 2017 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package operator

import (
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/ngaut/log"
	"golang.org/x/net/context"
)

// Box is a set of test case.
type Box struct {
	// Name is the name of the test box.
	Name  string
	cases map[string]*Case
	sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

func (b *Box) start() error {
	var wg sync.WaitGroup
	for _, c := range b.cases {
		wg.Add(1)
		go func(c *Case) {
			defer wg.Done()
			if c.state == CaseRunning || c.state == CaseStarting || c.state == CaseRestarting {
				return
			}
			err := runWithRetry(100, 3*time.Second, c.start)
			if err != nil {
				log.Errorf("[box: %s][case: %s]start failed: %v", b.Name, c.Name, err)
				c.changeToState(CaseStartError)
				return
			}
		}(c)
	}
	wg.Wait()
	return nil
}

func (b *Box) stop() error {
	var wg sync.WaitGroup
	for _, c := range b.cases {
		wg.Add(1)
		go func(c *Case) {
			defer wg.Done()
			err := runWithRetry(100, 3*time.Second, c.stop)
			if err != nil {
				log.Errorf("[box: %s][case: %s]stop failed: %v", b.Name, c.Name, err)
				return
			}
		}(c)
	}
	wg.Wait()
	return nil
}

func (b *Box) listCase() map[string]*Case {
	return b.cases
}

func (b *Box) getCase(name string) (*Case, error) {
	b.RLock()
	c, ok := b.cases[name]
	b.Unlock()
	if !ok {
		return nil, errors.NotFoundf("[box: %s][case: %s]", b.Name, name)
	} else {
		return c, nil
	}
}

func (b *Box) startCase(c *Case) error {
	if !b.valid(c) {
		return errors.NotFoundf("[box: %s][case: %s]", b.Name, c.Name)
	}
	err := runWithRetry(100, 3*time.Second, c.start)
	if err != nil {
		return errors.Errorf("[box: %s][case: %s] start failed: %v", b.Name, c.Name, err)
	}
	return nil
}

func (b *Box) stopCase(c *Case) error {
	if !b.valid(c) {
		return errors.NotFoundf("[box: %s][case: %s]", b.Name, c.Name)
	}
	err := runWithRetry(100, 3*time.Second, c.stop)
	if err != nil {
		return errors.Errorf("[box: %s][case: %s] stop failed: %v", b.Name, c.Name, err)
	}
	return nil
}

func (b *Box) addCase(c *Case) error {
	if b.valid(c) {
		return errors.AlreadyExistsf("[box: %s][case: %s]", b.Name, c.Name)
	}
	c.ctx, c.cancel = context.WithCancel(context.Background())
	b.Lock()
	b.cases[c.Name] = c
	b.Unlock()
	c.changeToState(CaseNew)
	return nil
}

func (b *Box) deleteCase(c *Case) error {
	if !b.valid(c) {
		return errors.NotFoundf("[box: %s][case: %s]", b.Name, c.Name)
	}
	// TODO: close sync goroutine
	c.cancel()
	delete(b.cases, c.Name)
	return nil
}

func (b *Box) valid(c *Case) bool {
	b.RLock()
	_, ok := b.cases[c.Name]
	b.Unlock()
	if !ok {
		return false
	}
	return true
}

func (b *Box) monitor(ctx context.Context) error {
	var wg sync.WaitGroup
	for _, c := range b.cases {
		wg.Add(1)
		go func(c *Case) {
			defer wg.Done()
			c.monitor(ctx)
		}(c)
	}
	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			break
		case <-b.ctx.Done():
			for _, c := range b.cases {
				c.cancel()
			}
			wg.Wait()
			break
		default:
		}
	}
	return nil
}
