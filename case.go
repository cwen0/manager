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
	"k8s.io/client-go/pkg/api/v1"
)

// Case
type Case struct {
	// Name
	Name string `yaml:"name", json:"name"`

	// URL is the url for agent to down binary
	URL string `yaml:"url" json:"url"`

	// Image
	Image string `yaml:"image" json:"image"`

	// Port
	Port int `yaml:"port" json:"port"`

	// Label
	Labels map[string]string `yaml:"labels" json:"lables"`

	pod *v1.Pod

	status Status
}

func (c *Case) start() error {
	_, err := xpost(fmt.Sprintf("%s/%d/process/%s/start", c.pod.Status.PodIP, c.Port, c.Name), []byte())
	if err != nil {
		c.status = STARTERROR
		return errors.Trace(err)
	}
	return nil
}

func (c *Case) stop() error {
	_, err := xpost(fmt.Sprintf("%s/%d/process/%s/stop", c.pod.Status.PodIP, c.Port, c.Name), []byte())
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
