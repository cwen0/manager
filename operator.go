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
func New(c *rest.Config) (*Operator, error) {
	cli, err := kubernetes.NewForConfig(c)
	if err != nil {
		return nil, errors.Trace(err)
	}
	op := &Operator{
		cli:  cli,
		boxs: make(map[string]*Box),
	}
	return op, nil
}

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
	if b.status == STARTING {
		return fmt.Errorf("[box: %s] is starting.", b.Name)
	}
	for _, c := range b.cases {
		if c.pod != nil {
			continue
		}
		if err := o.createPod(b, c); err != nil {
			c.status = CREATEPODERROR
		}
	}
	b.status = STARTING
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
	if b.status == STOP {
		return fmt.Errorf("[box: %s] had been stoped.", b.Name)
	}
	err := b.stop()
	if err != nil {
		return errors.Trace(err)
	}
	b.status = STOP
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
		return nil, fmt.Errorf("[box: %s] is not exist.", name)
	}
}

// AddCase is to add test Case in test Box.
func (o *Operator) AddCase(b *Box, c *Case) error {
	if o.valid(b, c) {
		return fmt.Errorf("[box: %s][case: %s] is exist.", b.Name, c.Name)
	}
	if err := b.addCase(c); err != nil {
		return errors.Trace(err)
	}
	return nil
}

// DeleteCase is to delete test Case.
func (o *Operator) DeleteCase(b *Box, c *Case) error {
	if !o.valid(b, c) {
		return fmt.Errorf("[box: %s][case: %s] is not exist.", b.Name, c.Name)
	}
	if err := b.deleteCase(c); err != nil {
		return errors.Trace(err)
	}
	if err := o.deletePod(b, c); err != nil {
		return errors.Trace(err)
	}
	return nil
}

// StartCase is to start test Case.
func (o *Operator) StartCase(b *Box, c *Case) error {
	if !o.valid(b, c) {
		return fmt.Errorf("[box: %s][case: %s] is not exist.", b.Name, c.Name)
	}
	if c.pod == nil {
		if err := o.createPod(b, c); err != nil {
			c.status = CREATEPODERROR
			return errors.Trace(err)
		}
	}
	if err := b.startCase(c); err != nil {
		return errors.Trace(err)
	}
	return nil
}

// StopCase is to stop test Case.
func (o *Operator) StopCase(b *Box, c *Case) error {
	if !o.valid(b, c) {
		return fmt.Errorf("[box: %s][case: %s] is not exist.", b.Name, c.Name)
	}
	if err := b.stopCase(c); err != nil {
		return errors.Trace(err)
	}
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
	err := o.cli.CoreV1().Namespaces().Delete(namespace, &metav1.DeleteOptions{})
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

func (o *Operator) createPod(b *Box, c *Case) error {
	if !o.valid(b, c) {
		return fmt.Errorf("[box: %s] [case: %s] is invalid.", b.Name, c.Name)
	}
	cmds := []string{}
	pod, err := o.cli.Pods(b.Name).Create(&v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
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
							ContainerPort: int32(c.Port),
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
	if err := o.cli.Pods(b.Name).Delete(c.Name, &metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("[box: %s] [case: %s] delete faild: %v", b.Name, c.Name, err)
	}
	c.pod = nil
	return nil
}

func (o *Operator) valid(b *Box, c *Case) bool {
	if _, ok := o.boxs[b.Name]; !ok {
		return false
	}

	if !b.valid(c) {
		return false
	}
	return true
}
