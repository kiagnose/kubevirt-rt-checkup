/*
 * This file is part of the kiagnose project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2023 Red Hat, Inc.
 *
 */

package client

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	kvcorev1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

type Client struct {
	kubecli.KubevirtClient
}

type resultWrapper struct {
	vmi *kvcorev1.VirtualMachineInstance
	err error
}

func New() (*Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	client, err := kubecli.GetKubevirtClientFromRESTConfig(config)
	if err != nil {
		return nil, err
	}

	return &Client{client}, nil
}

func (c *Client) CreateVirtualMachineInstance(ctx context.Context,
	namespace string,
	vmi *kvcorev1.VirtualMachineInstance) (*kvcorev1.VirtualMachineInstance, error) {
	resultCh := make(chan resultWrapper, 1)

	go func() {
		createdVMI, err := c.KubevirtClient.VirtualMachineInstance(namespace).Create(vmi)
		resultCh <- resultWrapper{createdVMI, err}
	}()

	select {
	case result := <-resultCh:
		return result.vmi, result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Client) GetVirtualMachineInstance(ctx context.Context, namespace, name string) (*kvcorev1.VirtualMachineInstance, error) {
	resultCh := make(chan resultWrapper, 1)

	go func() {
		vmi, err := c.KubevirtClient.VirtualMachineInstance(namespace).Get(name, &metav1.GetOptions{})
		resultCh <- resultWrapper{vmi, err}
	}()

	select {
	case result := <-resultCh:
		return result.vmi, result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Client) DeleteVirtualMachineInstance(ctx context.Context, namespace, name string) error {
	resultCh := make(chan error, 1)

	go func() {
		err := c.KubevirtClient.VirtualMachineInstance(namespace).Delete(name, &metav1.DeleteOptions{})
		resultCh <- err
	}()

	select {
	case err := <-resultCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
