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

	"github.com/juju/errors"
	"golang.org/x/net/context"
	"k8s.io/client-go/pkg/api/v1"
)

// Case
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

	pod    *v1.Pod
	status Status
	ctx    *context.Context
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
		c.status = STOPERROR
		return errors.Trace(err)
	}
	return nil
}

// TODO: watch status
//func (c *Case) watch() error {
//	return nil
//}
