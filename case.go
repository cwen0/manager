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

	"github.com/GregoryIan/agent/daemon"
	"github.com/juju/errors"
	"github.com/ngaut/log"
	"golang.org/x/net/context"
	"k8s.io/client-go/pkg/api/v1"
)

var defaultSyncInterval = 10 * time.Second

// Case defines the test case.
type Case struct {
	// Name is the name of the test case.
	Name string `yaml:"name", json:"name"`

	// URL is the url for agent to down binary.
	URL string `yaml:"url" json:"url"`

	// Image is the docker image for running container.
	Image string `yaml:"image" json:"image"`

	// Port is export for container.
	Port int `yaml:"port" json:"port"`

	// Labels is for k8s to schedule.
	Labels map[string]string `yaml:"labels" json:"lables"`

	state CaseState
	pod   *v1.Pod
	sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

func (c *Case) start() error {
	_, err := xpost(fmt.Sprintf("%s/%d/process/%s/start", c.pod.Status.PodIP, c.Port, c.Name), []byte(""))
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (c *Case) stop() error {
	_, err := xpost(fmt.Sprintf("%s/%d/process/%s/stop", c.pod.Status.PodIP, c.Port, c.Name), []byte(""))
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// TODO: sync status
func (c *Case) monitor(ctx context.Context) error {
	ticker := time.NewTicker(defaultSyncInterval)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-c.ctx.Done():
			return nil
		case <-ticker.C:
			url := fmt.Sprintf("%s/%d/process/%s/state", c.pod.Status.PodIP, c.Port, c.Name)
			resp, err := xpost(url, []byte(""))
			if err != nil {
				log.Errorf("request: %s, failed: %v", url, err)
				c.changeToState(CaseTimeout)
			}
			switch resp.CState {
			case daemon.ProcStatStopped:
				c.changeToState(CaseStopped)
			case daemon.ProcStatStarting:
				c.changeToState(CaseStarting)
			case daemon.ProcStatRunning:
				c.changeToState(CaseRunning)
			case daemon.ProcStatRestarting:
				c.changeToState(CaseRestarting)
			case daemon.ProcStatStopping:
				c.changeToState(CaseStopping)
			case daemon.ProcStatKilling:
				c.changeToState(CaseKilling)
			case daemon.ProcStatTerminating:
				c.changeToState(CaseTerminating)
			case daemon.ProcStatExited:
				c.changeToState(CaseTerminating)
			case daemon.ProcStatKilled:
				c.changeToState(CaseKilled)
			case daemon.ProcStatFatal:
				c.changeToState(CaseFatal)
			case daemon.ProcStatUnknown:
				c.changeToState(CaseUnknown)
			default:
				c.changeToState(CaseUnknown)
			}
		}
	}
	return nil
}
