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
	"context"
	"sync"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/juju/errors"
	"github.com/ngaut/log"
)

type Box struct {
	name  string
	cases map[string]*Case
	sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

// NewK8sBox return a k8sBox
func NewBox(name string, cases []*Case, timeout time.Duration) Box {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return &k8sBox{
		name:   name,
		cases:  nil, // todo
		ctx:    ctx, // transfer to case
		cancel: cancel,
	}
}

func (b *k8sBox) Name() string {
	return b.name
}

func (b *k8sBox) Start() error {
	if atomic.LoadInt32(&b)
	var wg sync.WaitGroup
	for _, c := range b.cases {
		wg.Add(1)
		go func(c *Case) {
			defer wg.Done()
			if c.state == CaseRunning || c.state == CaseStarting || c.state == CaseRestarting {
				continue
			}
			err := runWithRetry(100, 3*time.Second, c.start)
			if err != nil {
				log.Errorf("[box: %s][case: %s]start failed: %v", b.Name(), c.Name(), err)
				c.changeToState(CaseStartError)
				return
			}
		}(c)
	}
	wg.Wait()
	return nil
}

func (b *k8sBox) Stop() error {
	var wg sync.WaitGroup
	for _, c := range b.cases {
		wg.Add(1)
		go func(c *Case) {
			defer wg.Done()
			err := runWithRetry(100, 3*time.Second, c.stop)
			if err != nil {
				log.Errorf("[box: %s][case: %s]stop failed: %v", b.Name(), c.Name(), err)
				return
			}
		}(c)
	}
	wg.Wait()
	return nil
}

func (b *k8sBox) State() {}

func (b *k8sBox) Cases() map[string] *Case {
	cs := make(map[string] *Case)
	b.RLock()
	for n, c := range b.cases {
		cs[n] = c
	}
	b.RUnlock()
	return cs
}

func (b *k8sBox) AddCase(c *Case) error {
	b.RLock()
	_, ok := b.cases[name]
	b.RUnlock()
	if !ok {
		return errors.AlreadyExistsf("[box: %s][case: %s]", b.Name, c.Name)
	}

	// todo: is fine? c.initial()?
	c.ctx, c.cancel = context.WithCancel(b.ctx)
	b.Lock()
	b.cases[c.Name] = c
	b.Unlock()
	return nil
}

func (b *k8sBox) removeCase(name string) error {
	b.RLock()
	c, ok := b.cases[name]
	b.RUnlock()
	if !ok {
		return errors.NotFoundf("case %s", name)
	}

	// TODO: close sync goroutine
	err := c.Stop()
	if err != nil {
		return errors.Trace(err)
	}
	b.Lock()
	delete(b.cases, c.Name)
	b.Unlock()
	return nil
}
