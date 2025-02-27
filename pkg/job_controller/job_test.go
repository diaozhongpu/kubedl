package job_controller

import (
	"context"
	"github.com/alibaba/kubedl/pkg/metrics"
	"strconv"
	"testing"
	"time"

	"github.com/alibaba/kubedl/apis/model/v1alpha1"
	apiv1 "github.com/alibaba/kubedl/pkg/job_controller/api/v1"
	"github.com/alibaba/kubedl/pkg/test_job/v1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDeletePodsAndServices(T *testing.T) {
	type testCase struct {
		cleanPodPolicy               apiv1.CleanPodPolicy
		deleteRunningPodAndService   bool
		deleteSucceededPodAndService bool
	}

	var testcase = []testCase{
		{
			cleanPodPolicy:               apiv1.CleanPodPolicyRunning,
			deleteRunningPodAndService:   true,
			deleteSucceededPodAndService: false,
		},
		{
			cleanPodPolicy:               apiv1.CleanPodPolicyAll,
			deleteRunningPodAndService:   true,
			deleteSucceededPodAndService: true,
		},
		{
			cleanPodPolicy:               apiv1.CleanPodPolicyNone,
			deleteRunningPodAndService:   false,
			deleteSucceededPodAndService: false,
		},
	}

	for _, tc := range testcase {
		runningPod := newPod("runningPod", corev1.PodRunning)
		succeededPod := newPod("succeededPod", corev1.PodSucceeded)
		allPods := []*corev1.Pod{runningPod, succeededPod}
		runningPodService := newService("runningPod")
		succeededPodService := newService("succeededPod")
		worker := 2
		var podList = corev1.PodList{}
		var serviceList = corev1.ServiceList{}

		testJob := &v1.TestJob{
			TypeMeta: metav1.TypeMeta{
				Kind: v1.Kind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-job",
				Namespace: metav1.NamespaceDefault,
			},
			Spec: v1.TestJobSpec{
				TestReplicaSpecs: make(map[apiv1.ReplicaType]*apiv1.ReplicaSpec),
			},
		}

		if worker > 0 {
			worker := int32(worker)
			workerReplicaSpec := &apiv1.ReplicaSpec{
				Replicas: &worker,
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							corev1.Container{
								Name:  v1.DefaultContainerName,
								Image: "test-image-for-pkg:latest",
								Args:  []string{"Fake", "Fake"},
								Ports: []corev1.ContainerPort{
									corev1.ContainerPort{
										Name:          v1.DefaultPortName,
										ContainerPort: v1.DefaultPort,
									},
								},
							},
						},
					},
				},
			}
			testJob.Spec.TestReplicaSpecs[v1.TestReplicaTypeWorker] = workerReplicaSpec
		}

		testJobController := v1.TestJobController{
			Job: testJob,
		}

		scheme := runtime.NewScheme()
		_ = corev1.AddToScheme(scheme)
		_ = v1.AddToScheme(scheme)

		fakeClient := fake.NewFakeClientWithScheme(scheme, testJob)
		fakeClient.Create(context.Background(), runningPod)
		fakeClient.Create(context.Background(), succeededPod)
		fakeClient.Create(context.Background(), runningPodService)
		fakeClient.Create(context.Background(), succeededPodService)

		eventBroadcaster := record.NewBroadcaster()
		mainJobController := NewJobController(
			fakeClient,
			&testJobController,
			JobControllerConfiguration{},
			eventBroadcaster.NewRecorder(scheme, corev1.EventSource{Component: "broadcast-controller"}),
			&metrics.JobMetrics{},
		)

		runPolicy := apiv1.RunPolicy{
			CleanPodPolicy: &tc.cleanPodPolicy,
		}

		err := mainJobController.deletePodsAndServices(&runPolicy, testJob, allPods)

		if assert.NoError(T, err) {
			if tc.deleteRunningPodAndService {
				// should delete the running pod and its service
				mainJobController.Client.List(context.Background(), &podList)
				mainJobController.Client.List(context.Background(), &serviceList)
				assert.NotContains(T, podList.Items, *runningPod)
				assert.NotContains(T, serviceList.Items, *runningPodService)
			} else {
				// should NOT delete the running pod and its service
				mainJobController.Client.List(context.Background(), &podList)
				mainJobController.Client.List(context.Background(), &serviceList)
				assert.Contains(T, podList.Items, *runningPod)
				assert.Contains(T, serviceList.Items, *runningPodService)
			}

			if tc.deleteSucceededPodAndService {
				// should delete the SUCCEEDED pod and its service
				mainJobController.Client.List(context.Background(), &podList)
				mainJobController.Client.List(context.Background(), &serviceList)
				assert.NotContains(T, podList.Items, *succeededPod)
				assert.NotContains(T, serviceList.Items, *succeededPodService)
			} else {
				// should NOT delete the SUCCEEDED pod and its service
				mainJobController.Client.List(context.Background(), &podList)
				mainJobController.Client.List(context.Background(), &serviceList)
				assert.Contains(T, podList.Items, *succeededPod)
				assert.Contains(T, serviceList.Items, *succeededPodService)
			}
		}
	}
}

func TestPastBackoffLimit(T *testing.T) {
	type testCase struct {
		backOffLimit           int32
		shouldPassBackoffLimit bool
	}

	var testcase = []testCase{
		{
			backOffLimit:           int32(0),
			shouldPassBackoffLimit: false,
		},
	}

	for _, tc := range testcase {
		runningPod := newPod("runningPod", corev1.PodRunning)
		succeededPod := newPod("succeededPod", corev1.PodSucceeded)
		allPods := []*corev1.Pod{runningPod, succeededPod}

		testJobController := v1.TestJobController{
			Pods: allPods,
		}

		mainJobController := JobController{
			Controller: &testJobController,
		}
		runPolicy := apiv1.RunPolicy{
			BackoffLimit: &tc.backOffLimit,
		}

		result, err := mainJobController.pastBackoffLimit("fake-job", &runPolicy, nil, allPods)

		if assert.NoError(T, err) {
			assert.Equal(T, result, tc.shouldPassBackoffLimit)
		}
	}
}

func TestPastActiveDeadline(T *testing.T) {
	type testCase struct {
		activeDeadlineSeconds    int64
		shouldPassActiveDeadline bool
	}

	var testcase = []testCase{
		{
			activeDeadlineSeconds:    int64(0),
			shouldPassActiveDeadline: true,
		},
		{
			activeDeadlineSeconds:    int64(2),
			shouldPassActiveDeadline: false,
		},
	}

	for _, tc := range testcase {

		testJobController := v1.TestJobController{}

		mainJobController := JobController{
			Controller: &testJobController,
		}
		runPolicy := apiv1.RunPolicy{
			ActiveDeadlineSeconds: &tc.activeDeadlineSeconds,
		}
		jobStatus := apiv1.JobStatus{
			StartTime: &metav1.Time{
				Time: time.Now(),
			},
		}

		result := mainJobController.pastActiveDeadline(&runPolicy, jobStatus)
		assert.Equal(
			T, result, tc.shouldPassActiveDeadline,
			"Result is not expected for activeDeadlineSeconds == "+strconv.FormatInt(tc.activeDeadlineSeconds, 10))
	}
}

func TestCleanupJobIfTTL(T *testing.T) {
	ttl := int32(0)
	runPolicy := apiv1.RunPolicy{
		TTLSecondsAfterFinished: &ttl,
	}
	oneDayAgo := time.Now()
	// one day ago
	oneDayAgo.AddDate(0, 0, -1)
	jobStatus := apiv1.JobStatus{
		CompletionTime: &metav1.Time{
			Time: oneDayAgo,
		},
	}

	testJobController := &v1.TestJobController{
		Job: &v1.TestJob{},
	}
	mainJobController := JobController{
		Controller: testJobController,
	}

	var job interface{}
	_, err := mainJobController.cleanupJob(&runPolicy, jobStatus, job)
	if assert.NoError(T, err) {
		// job field is zeroed
		assert.Empty(T, testJobController.Job)
	}
}

func TestCleanupJob(T *testing.T) {
	ttl := int32(0)
	runPolicy := apiv1.RunPolicy{
		TTLSecondsAfterFinished: &ttl,
	}
	jobStatus := apiv1.JobStatus{
		CompletionTime: &metav1.Time{
			Time: time.Now(),
		},
	}

	testJobController := &v1.TestJobController{
		Job: &v1.TestJob{},
	}
	mainJobController := JobController{
		Controller: testJobController,
	}

	var job interface{}
	_, err := mainJobController.cleanupJob(&runPolicy, jobStatus, job)
	if assert.NoError(T, err) {
		assert.Empty(T, testJobController.Job)
	}
}

func newPod(name string, phase corev1.PodPhase) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: corev1.PodStatus{
			Phase: phase,
		},
	}
	return pod
}

func newService(name string) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return service
}

func TestPodTemplateAddModelPathEnv(T *testing.T) {
	replicas := map[apiv1.ReplicaType]*apiv1.ReplicaSpec{
		"Worker": {
			RestartPolicy: "Never",
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "tensorflow",
							Image: "kubedl/tf-mnist-with-summaries:1.0",
							Env: []corev1.EnvVar{
								{
									Name:  "test",
									Value: "value",
								},
							},
						},
					},
				},
			},
		},
	}
	modelVersion := &v1alpha1.ModelVersion{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   "default",
			Name:        "versionName",
			UID:         "9423255b-4600-11e7-af6a-28d2447dc82b",
			Labels:      make(map[string]string, 0),
			Annotations: make(map[string]string, 0),
		},
		Spec: v1alpha1.ModelVersionSpec{
			ModelName: "modelName",
			CreatedBy: "user1",
			Storage: &v1alpha1.Storage{
				LocalStorage: &v1alpha1.LocalStorage{
					Path:     "/tmp/model",
					NodeName: "localhost",
				},
			},
		},
		Status: v1alpha1.ModelVersionStatus{},
	}
	addModelPathEnv(replicas, &modelVersion.Spec)
	assert.Equal(T, v1alpha1.KubeDLModelPath, replicas["Worker"].Template.Spec.Containers[0].Env[1].Name)
}
