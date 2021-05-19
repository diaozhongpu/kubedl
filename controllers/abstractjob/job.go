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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetJobFromAPIClient returns the Job from API server
func (r *JobReconciler) GetJobFromAPIClient(namespace, name string) (metav1.Object, error) {
	panic("GetJobFromAPIClient method is not implemented!")
}
