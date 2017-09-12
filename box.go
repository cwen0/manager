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

import "github.com/ngaut/log"

const (
	RUNNING        = "running"
	ERROR          = "error"
	STOP           = "stop"
	BUILDING       = "building"
	NEW            = "new"
	CREATEPODERROR = "create pod failed"
	STARTERROR     = "start failed"
	STOPERROR      = "stop failed"
)

type Status string

// Box is a set of test case.
type Box struct {
	cases  map[string]*Case
	status Status
	Name   string
}

func (b *Box) start() error {
	for _, c := range b.cases {
		// TODO: retry
		err := c.start()
		if err != nil {
			log.Errorf("[box: %s][case: %s] start failed: %v", b.Name, c.Name, err)
		}
	}
	return nil
}

func (b *Box) stop() error {
	return nil
}

func (b *Box) listCase() []*Case {
	return nil
}

func (b *Box) getCase(name string) *Case {
	return nil
}

func (b *Box) startCase(c *Case) error {
	return nil
}

func (b *Box) stopCase(c *Case) error {
	return nil
}

func (b *Box) addCase(c *Case) error {
	return nil
}

func (b *Box) deleteCase(c *Case) error {
	return nil
}
