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
	"fmt"
	"sync"
	"time"

	"github.com/ngaut/log"
	"golang.org/x/net/context"
)

const (
	RUNNING        = "running"
	ERROR          = "error"
	STOP           = "stop"
	BUILDING       = "building"
	NEW            = "new"
	CREATEPODERROR = "create pod failed"
	STARTERROR     = "start failed"
	STARTING       = "starting"
	STOPERROR      = "stop failed"
	TIMEOUT        = "timeout"
)

type Status string

// Box is a set of test case.
type Box struct {
	// Name is the name of the test box.
	Name   string
	cases  map[string]*Case
	status Status
	ctx    *context.Context
}

func (b *Box) start() error {
	wg := sync.WaitGroup
	for _, c := range b.cases {
		wg.Add(1)
		go func(c *Case) {
			defer wg.Done()
			if c.status == RUNNING {
				return
			}
			err := runWithRetry(100, 3*time.Second, c.start)
			if err != nil {
				log.Errorf("[box: %s][case: %s]start failed: %v", b.Name, c.Name, err)
				c.status = STARTERROR
				return
			}
			c.status = RUNNING

		}(c)
	}
	wg.Wait()
	return nil
}

func (b *Box) stop() error {
	wg := sync.WaitGroup
	for _, c := range b.cases {
		wg.Add(1)
		go func(c *Case) {
			defer wg.Done()
			if c.status == STOP {
				return
			}
			err := runWithRetry(100, 3*time.Second, c.stop)
			if err != nil {
				log.Errorf("[box: %s][case: %s]stop failed: %v", b.Name, c.Name, err)
				return
			}
			c.status = STOP
		}(c)
	}
	wg.Wait()
	return nil
}

func (b *Box) listCase() map[string]*Case {
	return b.cases
}

func (b *Box) getCase(name string) (*Case, error) {
	if c, ok := b.cases[name]; !ok {
		return nil, fmt.Errorf("[box: %s][case: %s] is not exist.", b.Name, name)
	} else {
		return c, nil
	}
}

func (b *Box) startCase(c *Case) error {
	if !b.valid(c) {
		return fmt.Errorf("[box: %s][case: %s] is not exist.", b.Name, c.Name)
	}
	if c.status == RUNNING {
		return fmt.Errorf("[box: %s][case: %s] is running.", b.Name, c.Name)
	}
	c.status = STARTING
	err := runWithRetry(100, 3*time.Second, c.start)
	if err != nil {
		c.status = STARTERROR
		return fmt.Errorf("[box: %s][case: %s] start failed: %v", b.Name, c.Name, err)
	}
	c.status = RUNNING
	return nil
}

func (b *Box) stopCase(c *Case) error {
	if !b.valid(c) {
		return fmt.Errorf("[box: %s][case: %s] is not exist.", b.Name, c.Name)
	}
	if c.status != RUNNING {
		return fmt.Errorf("[box: %s][case: %s] is not running.", b.Name, c.Name)
	}
	err := runWithRetry(100, 3*time.Second, c.stop)
	if err != nil {
		return fmt.Errorf("[box: %s][case: %s] start failed: %v", b.Name, c.Name, err)
	}
	c.status = STOP
	return nil
}

func (b *Box) addCase(c *Case) error {
	if b.valid(c) {
		return fmt.Errorf("[box: %s][case: %s] is exist.", b.Name, c.Name)
	}
	b.cases[c.Name] = c
	c.status = NEW
	return nil
}

func (b *Box) deleteCase(c *Case) error {
	if !b.valid(c) {
		return fmt.Errorf("[box: %s][case: %s] is not exist.", b.Name, c.Name)
	}
	delete(b.cases, c.Name)
	return nil
}

func (b *Box) valid(c *Case) bool {
	if _, ok := b.cases[c.Name]; !ok {
		return false
	}
	return true
}
