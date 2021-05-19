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

package abstractjob

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/alibaba/kubedl/pkg/job_controller"
)

const (
	controllerName = "DefaultController"
)

var log = logf.Log.WithName("default-controller")

// var _ reconcile.Reconciler = &JobReconciler{}
// var _ v1.ControllerInterface = &JobReconciler{}

type JobReconciler struct {
	client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
	ctrl     job_controller.JobController
}

func (r *JobReconciler) GetAPIGroupVersionKind() schema.GroupVersionKind {
	panic("GetAPIGroupVersionKind method is not implemented!")
}
