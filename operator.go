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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
)

// Operator is a operator to manage test Box.
type Operator struct {
	cli  *kubernetes.Clientset
	boxs map[string]*Box
}

// New returns a Operator struct.
func New(c *rest.Config) *Operator {
	op := Operator{
		cli:  kubernetes.NewForConfig(c),
		boxs: make(map[string]*Box),
	}
	return op
}

// Start is to start operator.
// TODO: keep heartbeat with agent.
// func (o *Operator) Start() error {
// }

// CreateBox is to create new test Box.
func (o *Operator) CreateBox(b *Box) error {
	if _, ok := o.boxs[b.Name]; ok {
		return fmt.Errorf("[box:%s] is exist.", b.Name)
	}
	err := o.createNamespace(b.Name)
	if err != nil {
		return errors.Trace(err)
	}
	b.status = NEW
	o.boxs[b.Name] = b
	return nil
}

// GetBoxs is to list test Boxs.
func (o *Operator) ListBoxs() map[string]*Box {
	return o.boxs
}

// StartBox is to run test Box.
func (o *Operator) StartBox(b *Box) error {
	if _, ok := o.boxs[b.Name]; !ok {
		return fmt.Errorf("[box: %s] is not exist.", b.Name)
	}
	if b.status == RUNNING {
		return fmt.Errorf("[box: %s] is running.", b.Name)
	}
	for _, c := range b.cases {
		if c.pod != nil {
			continue
		}
		if err := o.createPod(b, c); err != nil {
			c.status = CREATEPODERROR
		}
	}
	err := b.start()
	if err != nil {
		return errors.Trace(err)
	}
	b.status = RUNNING
	return nil
}

// StopBox is to stop test Box.
func (o *Operator) StopBox(b *Box) error {
	if _, ok := o.boxs[b.Name]; !ok {
		return fmt.Errorf("[box: %s] is not exist.", b.Name)
	}
	err := b.stop()
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// DeleteBox is to delete test Box.
func (o *Operator) DeleteBox(b *Box) error {
	if _, ok := o.boxs[b.Name]; !ok {
		return fmt.Errorf("[box: %s] is not exist.", b.Name)
	}
	err := o.deleteNamespace(b.Name)
	if err != nil {
		return errors.Trace(err)
	}
	delete(o.boxs, b.Name)
	return nil
}

// GetBox is to get a test Box by name.
func (o *Operator) GetBox(name string) (*Box, error) {
	if box, ok := o.boxs[name]; ok {
		return box, nil
	} else {

		return nil, fmt.Errorf("[box: %s] is not exist.", b.Name)
	}
}

// AddCase is to add test Case in test Box.
func (o *Operator) AddCase(b *Box, c *Case) error {
	return nil
}

// DeleteCase is to delete test Case.
func (o *Operator) DeleteCase(b *Box, c *Case) error {
	return nil
}

// StartCase is to start test Case.
func (o *Operator) StartCase(b *Box, c *Case) error {
	return nil
}

// StopCase is to stop test Case.
func (o *Operator) StopCase(b *Box, c *Case) error {
	return nil
}

func (o *Operator) createNamespace(namespace string) error {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	_, err := o.cli.CoreV1().Namespaces().Create(ns)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (o *Operator) deleteNamespace(namespace string) error {
	_, err := o.cli.CoreV1().Namespaces().Delete(namespace)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (o *Operator) createPod(b *Box, c *Case) error {
	if !o.valid(b, c) {
		return fmt.Errorf("[box: %s] [case: %s] is invalid.", b.Name, c.Name)
	}
	cmds := []strings{}
	pod, err := o.cli.Pods(b.Name).Create(&v1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:   c.Name,
			Labels: c.Labels,
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:    c.Name,
					Image:   c.Image,
					Command: cmds,
					Ports: []v1.ContainerPort{
						{
							ContainerPort: c.Port,
						},
					},
				},
			},
		},
	})
	if err != nil {
		return errors.Trace(err)
	}
	c.pod = pod
	return nil
}

func (o *Operator) deletePod(b *Box, c *Case) error {
	if !o.valid(b, c) {
		return fmt.Errorf("[box: %s] [case: %s] is invalid.", b.Name, c.Name)
	}
	if c.pod == nil {
		return fmt.Errorf("[box: %s] [case: %s] pod has been deleted", b.Name, c.Name)
	}
	if err := o.cli.Pods(b.Name).Delete(c.Name, &v1.DeleteOptions{}); err != nil {
		return fmt.Errorf("[box: %s] [case: %s] delete faild: %v", b.Name, c.Name, err)
	}
	c.pod = nil
	return nil
}

func (o *Operator) valid(b *Box, c *Case) bool {
	if _, ok := o.boxs[b.Name]; !ok {
		return false
	}

	if _, ok := b.cases[c.Name]; !ok {
		return false
	}
	return true
}
