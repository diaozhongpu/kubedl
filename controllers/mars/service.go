/*
Copyright 2020 The Alibaba Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	"github.com/alibaba/kubedl/apis/training/v1alpha1"
	"github.com/alibaba/kubedl/pkg/job_controller"
	"github.com/alibaba/kubedl/pkg/util"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *MarsJobReconciler) GetServicesForJob(obj interface{}) ([]*corev1.Service, error) {
	job, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}
	selector, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels: r.ctrl.GenLabels(job.GetName()),
	})
	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	serviceList := &corev1.ServiceList{}
	err = r.List(context.Background(), serviceList, client.MatchingLabelsSelector{Selector: selector})
	if err != nil {
		return nil, err
	}
	services := util.ToServicePointerList(serviceList.Items)
	// If any adoptions are attempted, we should first recheck for deletion
	// with an uncached quorum read sometime after listing services (see #42639).
	canAdoptFunc := job_controller.RecheckDeletionTimestamp(func() (metav1.Object, error) {
		fresh, err := r.GetJobFromInformerCache(job.GetNamespace(), job.GetName())
		if err != nil {
			return nil, err
		}
		if fresh.GetUID() != job.GetUID() {
			return nil, fmt.Errorf("original Job %v/%v is gone: got uid %v, wanted %v", job.GetNamespace(), job.GetName(), fresh.GetUID(), job.GetUID())
		}
		return fresh, nil
	})
	cm := job_controller.NewServiceControllerRefManager(job_controller.NewServiceControl(r.Client, r.recorder), job, selector, r.GetAPIGroupVersionKind(), canAdoptFunc)
	return cm.ClaimServices(services)
}

func (r *MarsJobReconciler) DeleteService(job interface{}, name string, namespace string) error {
	marsJob, ok := job.(*v1alpha1.MarsJob)
	if !ok {
		return fmt.Errorf("%+v is not a type of MarsJob", job)
	}

	r.ctrl.DeleteService(marsJob, name, namespace)
	return nil
}
