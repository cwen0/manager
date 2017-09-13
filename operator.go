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
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/coreos/etcd/clientv3"
	"github.com/juju/errors"
	"github.com/ngaut/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"
)

const (
	RootPath = "stability-test"
)

// Operator is a operator to manage test Box.
type Operator struct {
	sync.RWMutex
	cli     *kubernetes.Clientset
	boxs    map[string]*Box
	etcdCli *clientv3.Client
}

// New returns a Operator struct.
func New(c *rest.Config, etcdCli *clientv3.Client) (*Operator, error) {
	cli, err := kubernetes.NewForConfig(c)
	if err != nil {
		return nil, errors.Trace(err)
	}
	op := &Operator{
		cli:     cli,
		boxs:    make(map[string]*Box),
		etcdCli: etcdCli,
	}
	return op, nil
}

// Start starts sync state from agent and recover from etcd.
func (o *Operator) Start(ctx context.Context) error {
	// TODO: recover from etcd
	if err := o.recover(ctx); err != nil {
		log.Errorf("revover failed: %v", err)
		return errors.Trace(err)
	}
	for _, b := range o.boxs {
		go b.monitor(ctx, o.etcdCli)
	}
	return nil
}

// CreateBox is to create new test Box.
func (o *Operator) CreateBox(ctx context.Context, b *Box) error {
	o.RLock()
	_, ok := o.boxs[b.Name]
	o.RUnlock()
	if ok {
		return errors.AlreadyExistsf("[box: %s]", b.Name)
	}
	err := o.createNamespace(b.Name)
	if err != nil {
		return errors.Trace(err)
	}
	b.ctx, b.cancel = context.WithCancel(context.Background())
	o.Lock()
	o.boxs[b.Name] = b
	o.Unlock()
	for _, c := range b.cases {
		go func(c *Case) {
			if err := o.createPod(b, c); err != nil {
				c.changeToState(CaseCreatePodError)
			}
		}(c)
	}
	go b.monitor(ctx, o.etcdCli)
	return nil
}

// GetBoxs is to list test Boxs.
func (o *Operator) ListBoxs() map[string]*Box {
	return o.boxs
}

// StartBox is to run test Box.
func (o *Operator) StartBox(b *Box) error {
	o.RLock()
	_, ok := o.boxs[b.Name]
	o.RUnlock()
	if !ok {
		return errors.NotFoundf("[box: %s]", b.Name)
	}
	err := b.start()
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// StopBox is to stop test Box.
func (o *Operator) StopBox(b *Box) error {
	o.RLock()
	_, ok := o.boxs[b.Name]
	o.RUnlock()
	if !ok {
		return errors.NotFoundf("[box: %s]", b.Name)
	}
	err := b.stop()
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// DeleteBox is to delete test Box.
func (o *Operator) DeleteBox(b *Box) error {
	o.RLock()
	_, ok := o.boxs[b.Name]
	o.RUnlock()
	if !ok {
		return errors.NotFoundf("[box: %s]", b.Name)
	}
	// TODO: close sync state
	// wait all sync goroutine is closed
	b.cancel()
	delete(o.boxs, b.Name)
	err := o.deleteNamespace(b.Name)
	if err != nil {
		return errors.Trace(err)
	}
	if _, err := o.etcdCli.Delete(context.TODO(), fmt.Sprintf("%s/%s", RootPath, b.Name), clientv3.WithPrefix()); err != nil {
		log.Errorf("delete [box: %s] from etcd, failed: %v", b.Name, err)
		return errors.Trace(err)
	}
	return nil
}

// GetBox is to get a test Box by name.
func (o *Operator) GetBox(name string) (*Box, error) {
	o.RLock()
	box, ok := o.boxs[name]
	o.Unlock()
	if ok {
		return box, nil
	} else {
		return nil, errors.NotFoundf("[box: %s]", name)
	}
}

// AddCase is to add test Case in test Box.
func (o *Operator) AddCase(ctx context.Context, b *Box, c *Case) error {
	if o.valid(b, c) {
		return errors.AlreadyExistsf("[box: %s][case: %s]", b.Name, c.Name)
	}
	if err := b.addCase(c); err != nil {
		return errors.Trace(err)
	}
	if err := o.createPod(b, c); err != nil {
		c.changeToState(CaseCreatePodError)
		return errors.Trace(err)
	}
	go c.monitor(ctx, b.Name, o.etcdCli)
	return nil
}

// DeleteCase is to delete test Case.
func (o *Operator) DeleteCase(b *Box, c *Case) error {
	if !o.valid(b, c) {
		return errors.NotFoundf("[box: %s][case: %s]", b.Name, c.Name)
	}
	if err := b.deleteCase(c); err != nil {
		return errors.Trace(err)
	}
	if err := o.deletePod(b, c); err != nil {
		return errors.Trace(err)
	}

	if err := o.etcdCli.Delete(context.TODO(), fmt.Sprintf("%s/%s/%s", RootPath, b.Name, c.Name)); err != nil {
		log.Errorf("delete [box: %s][case: %s] from etcd, failed: %v", b.Name, c.Name, err)
		return errors.Trace(err)
	}
	return nil
}

// StartCase is to start test Case.
func (o *Operator) StartCase(b *Box, c *Case) error {
	if !o.valid(b, c) {
		return errors.NotFoundf("[box: %s][case: %s]", b.Name, c.Name)
	}
	if err := b.startCase(c); err != nil {
		c.changeToState(CaseStartError)
		return errors.Trace(err)
	}
	return nil
}

// StopCase is to stop test Case.
func (o *Operator) StopCase(b *Box, c *Case) error {
	if !o.valid(b, c) {
		return errors.NotFoundf("[box: %s][case: %s]", b.Name, c.Name)
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
	o.RLock()
	_, ok := o.boxs[b.Name]
	defer o.RUnlock()
	if !ok {
		return false
	}

	if !b.valid(c) {
		return false
	}
	return true
}

func (o *Operator) recover(ctx context.Context) error {
	resp, err := o.etcdCli.Get(context.TODO(), RootPath, clientv3.WithPrefix())
	if err != nil {
		return errors.Trace(err)
	}
	for _, cs := range resp.Kvs {
		keys := strings.Split(cs.Key, "/")
		if lens(keys) != 3 {
			return errors.New("get key from etcd is invalid")
		}
		if keys[2] != "" && keys[1] != "" {
			var caseTmp Case
			if err := json.Unmarshal(cs.Value, &caseTmp); err != nil {
				log.Errorf("Unmarshal [%s] failed: %v", string(cs.Value), err)
				return errors.Trace(err)
			}
			caseTmp.changeToState(caseTmp.Status)
			o.RLock()
			box, ok := o.boxs[keys[1]]
			o.RUnlock()
			if !ok {
				box = &Box{
					Name: keys[1],
				}

				if err := o.CreateBox(ctx, box); err != nil {
					return errors.Trace(err)
				}
			}
			err := box.addCase(caseTmp)
			if err != nil {
				return errors.Trace(err)
			}
			go caseTmp.monitor(ctx, boxName, o.etcdCli)
		}
	}
}
