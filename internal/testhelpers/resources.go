// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testhelpers

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/GoogleCloudPlatform/cloud-sql-proxy-operator/internal/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func buildPodTemplateSpec(mainPodSleep int, appLabel string) corev1.PodTemplateSpec {
	podCmd := fmt.Sprintf("echo Container 1 is Running ; sleep %d", mainPodSleep)
	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{"app": appLabel},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:            "busybox",
				Image:           "busybox",
				ImagePullPolicy: "IfNotPresent",
				Command:         []string{"sh", "-c", podCmd},
			}},
		},
	}
}

const (
	userKey     = "DB_USER"
	passwordKey = "DB_PASS"
	dbNameKey   = "DB_NAME"
)

// BuildSecret creates a Secret object containing database information to be used
// by the pod to connect to the database.
func BuildSecret(secretName, user, password, dbName string) corev1.Secret {
	return corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: secretName,
		},
		Type: "Opaque",
		Data: map[string][]byte{
			userKey:     []byte(user),
			passwordKey: []byte(password),
			dbNameKey:   []byte(dbName),
		},
	}
}

// BuildPgPodSpec creates a podspec specific to Postgres databases that will connect
// and run a trivial query. It also configures the pod's Liveness probe so that
// the pod's `Ready` condition is `Ready` when the database can connect.
func BuildPgPodSpec(mainPodSleep int, appLabel, secretName string) corev1.PodTemplateSpec {
	const (
		livenessCmd    = "psql --host=$DB_HOST --port=$DB_PORT --username=$DB_USER '--command=select 1' --echo-queries --dbname=$DB_NAME"
		imageName      = "postgres"
		passEnvVarName = "PGPASSWORD"
	)

	return buildConnectPodSpec(mainPodSleep, appLabel, secretName, livenessCmd, passEnvVarName, imageName)
}

// BuildMysqlPodSpec creates a podspec specific to Mysql databases that will connect
// and run a trivial query. It also configures the pod's Liveness probe so that
// the pod's `Ready` condition is `Ready` when the database can connect.
func BuildMysqlPodSpec(mainPodSleep int, appLabel, secretName string) corev1.PodTemplateSpec {
	const (
		livenessCmd    = "mysql --host=$DB_HOST --port=$DB_PORT --user=$DB_USER --password=$DB_PASS --database=$DB_NAME '--execute=select now()' "
		imageName      = "mysql"
		passEnvVarName = "DB_PASS"
	)

	return buildConnectPodSpec(mainPodSleep, appLabel, secretName, livenessCmd, passEnvVarName, imageName)
}

// BuildMSSQLPodSpec creates a podspec specific to Mysql databases that will connect
// and run a trivial query. It also configures the pod's Liveness probe so that
// the pod's `Ready` condition is `Ready` when the database can connect.
func BuildMSSQLPodSpec(mainPodSleep int, appLabel, secretName string) corev1.PodTemplateSpec {
	const (
		livenessCmd    = "/opt/mssql-tools/bin/sqlcmd -S \"tcp:$DB_HOST,$DB_PORT\" -U $DB_USER -P $DB_PASS -Q \"use $DB_NAME ; select 1 ;\""
		imageName      = "mcr.microsoft.com/mssql-tools"
		passEnvVarName = "DB_PASS"
	)

	return buildConnectPodSpec(mainPodSleep, appLabel, secretName, livenessCmd, passEnvVarName, imageName)
}

func buildConnectPodSpec(mainPodSleep int, appLabel, secretName, livenessCmd, passEnvVarName, imageName string) corev1.PodTemplateSpec {
	podCmd := fmt.Sprintf(`echo Container 1 is Running
	sleep 30
	%s
	sleep %d`, livenessCmd, mainPodSleep)

	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{"app": appLabel},
		},

		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:            "db-client-app",
				Image:           imageName,
				ImagePullPolicy: "IfNotPresent",
				Command:         []string{"/bin/sh", "-e", "-x", "-c", podCmd},
				Resources: corev1.ResourceRequirements{
					Requests: map[corev1.ResourceName]resource.Quantity{
						corev1.ResourceCPU: *resource.NewMilliQuantity(500, resource.DecimalExponent),
					},
				},
				LivenessProbe: &corev1.Probe{InitialDelaySeconds: 10, PeriodSeconds: 30, FailureThreshold: 3,
					ProbeHandler: corev1.ProbeHandler{
						Exec: &corev1.ExecAction{
							Command: []string{"/bin/sh", "-c", "-e", livenessCmd},
						},
					},
				},
				ReadinessProbe: &corev1.Probe{InitialDelaySeconds: 10, PeriodSeconds: 30, FailureThreshold: 3,
					ProbeHandler: corev1.ProbeHandler{
						Exec: &corev1.ExecAction{
							Command: []string{"/bin/sh", "-c", "-e", livenessCmd},
						},
					},
				},
				Env: []corev1.EnvVar{
					{
						Name: "DB_USER",
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
								Key:                  userKey,
							},
						},
					},
					{
						Name: passEnvVarName,
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
								Key:                  passwordKey,
							},
						},
					},
					{
						Name: "DB_NAME",
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
								Key:                  dbNameKey,
							},
						},
					},
				},
			}},
		},
	}
}

// BuildDeployment creates a StatefulSet object with a default pod template
// that will sleep for 1 hour.
func BuildDeployment(name types.NamespacedName, appLabel string) *appsv1.Deployment {
	var two int32 = 2
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name,
			Namespace: name.Namespace,
			Labels:    map[string]string{"app": appLabel},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &two,
			Strategy: appsv1.DeploymentStrategy{Type: "RollingUpdate"},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": appLabel},
			},
			Template: buildPodTemplateSpec(3600, appLabel),
		},
	}
}

// BuildStatefulSet creates a StatefulSet object with a default pod template
// that will sleep for 1 hour.
func BuildStatefulSet(name types.NamespacedName, appLabel string) *appsv1.StatefulSet {
	var two int32 = 2
	return &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{Kind: "StatefulSet", APIVersion: "apps/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name,
			Namespace: name.Namespace,
			Labels:    map[string]string{"app": appLabel},
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:       &two,
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{Type: "RollingUpdate"},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": appLabel},
			},
			Template: buildPodTemplateSpec(3600, appLabel),
		},
	}
}

// BuildDaemonSet creates a DaemonSet object with a default pod template
// that will sleep for 1 hour.
func BuildDaemonSet(name types.NamespacedName, appLabel string) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{Kind: "DaemonSet", APIVersion: "apps/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name,
			Namespace: name.Namespace,
			Labels:    map[string]string{"app": appLabel},
		},
		Spec: appsv1.DaemonSetSpec{
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{Type: "RollingUpdate"},
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": appLabel},
			},
			Template: buildPodTemplateSpec(3600, appLabel),
		},
	}
}

// BuildJob creates a Job object with a default pod template
// that will sleep for 30 seconds.
func BuildJob(name types.NamespacedName, appLabel string) *batchv1.Job {
	job := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{Kind: "Job", APIVersion: "batch/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name,
			Namespace: name.Namespace,
			Labels:    map[string]string{"app": appLabel},
		},
		Spec: batchv1.JobSpec{
			Template:    buildPodTemplateSpec(30, appLabel),
			Parallelism: ptr(int32(1)), // run the pod 20 times, 1 at a time
			Completions: ptr(int32(20)),
		},
	}
	job.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyNever
	return job
}

func ptr[T any](i T) *T {
	return &i
}

// BuildCronJob creates CronJob object with a default pod template
// that will sleep for 60 seconds.
func BuildCronJob(name types.NamespacedName, appLabel string) *batchv1.CronJob {
	job := &batchv1.CronJob{
		TypeMeta: metav1.TypeMeta{Kind: "CronJob", APIVersion: "batch/v1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name.Name,
			Namespace: name.Namespace,
			Labels:    map[string]string{"app": appLabel},
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "* * * * *",
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: buildPodTemplateSpec(60, appLabel),
				},
			},
		},
	}
	job.Spec.JobTemplate.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyNever
	return job

}

// CreateWorkload Creates the workload in Kubernetes, waiting to confirm that
// the workload exists.
func (cc *TestCaseClient) CreateWorkload(ctx context.Context, o client.Object) error {
	err := cc.Client.Create(ctx, o)
	if err != nil {
		return err
	}

	err = RetryUntilSuccess(5, time.Second, func() error {
		return cc.Client.Get(ctx, client.ObjectKeyFromObject(o), o)
	})
	if err != nil {
		return err
	}
	return nil
}

// GetAuthProxyWorkloadAfterReconcile finds an AuthProxyWorkload resource named key, waits for its
// "UpToDate" condition to be "True", and the returns it. Fails after 30 seconds
// if the containers does not match.
func (cc *TestCaseClient) GetAuthProxyWorkloadAfterReconcile(ctx context.Context, key types.NamespacedName) (*v1alpha1.AuthProxyWorkload, error) {
	createdPodmod := &v1alpha1.AuthProxyWorkload{}
	// We'll need to retry getting this newly created resource, given that creation may not immediately happen.
	err := RetryUntilSuccess(6, DefaultRetryInterval, func() error {
		err := cc.Client.Get(ctx, key, createdPodmod)
		if err != nil {
			return err
		}
		if GetConditionStatus(createdPodmod.Status.Conditions, v1alpha1.ConditionUpToDate) != metav1.ConditionTrue {
			return errors.New("AuthProxyWorkload found, but reconcile not complete yet")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return createdPodmod, nil
}

// CreateBusyboxDeployment creates a simple busybox deployment, using the
// key as its namespace and name. It also sets the label "app"= appLabel.
func (cc *TestCaseClient) CreateBusyboxDeployment(ctx context.Context, name types.NamespacedName, appLabel string) (*appsv1.Deployment, error) {

	d := BuildDeployment(name, appLabel)

	err := cc.Client.Create(ctx, d)
	if err != nil {
		return nil, err
	}

	cd := &appsv1.Deployment{}
	err = RetryUntilSuccess(5, time.Second, func() error {
		return cc.Client.Get(ctx, types.NamespacedName{
			Namespace: name.Namespace,
			Name:      name.Name,
		}, cd)
	})
	if err != nil {
		return nil, err
	}
	return cd, nil
}

// ListPods lists all the pods in a particular deployment.
func ListPods(ctx context.Context, c client.Client, ns string, selector *metav1.LabelSelector) (*corev1.PodList, error) {

	podList := &corev1.PodList{}
	sel, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return nil, fmt.Errorf("unable to make pod selector for deployment %v", err)
	}

	err = c.List(ctx, podList, client.InNamespace(ns), client.MatchingLabelsSelector{Selector: sel})
	if err != nil {
		return nil, fmt.Errorf("unable to list pods for deployment %v", err)
	}

	return podList, nil
}

// ExpectPodContainerCount finds a deployment and keeps checking until the number of
// containers on the deployment's PodSpec.Containers == count. Returns error after 30 seconds
// if the containers do not match.
func (cc *TestCaseClient) ExpectPodContainerCount(ctx context.Context, podSelector *metav1.LabelSelector, count int, allOrAny string) error {

	var (
		countBadPods int
		countPods    int
	)

	err := RetryUntilSuccess(24, DefaultRetryInterval, func() error {
		countBadPods = 0
		pods, err := ListPods(ctx, cc.Client, cc.Namespace, podSelector)
		if err != nil {
			return err
		}
		countPods = len(pods.Items)
		if len(pods.Items) == 0 {
			return fmt.Errorf("got 0 pods, want at least 1 pod")
		}
		for _, pod := range pods.Items {
			got := len(pod.Spec.Containers)
			if got != count {
				countBadPods++
			}
		}
		switch {
		case allOrAny == "all" && countBadPods > 0:
			return fmt.Errorf("got the wrong number of containers on %d of %d pods", countBadPods, len(pods.Items))
		case allOrAny == "any" && countBadPods == countPods:
			return fmt.Errorf("got the wrong number of containers on %d of %d pods", countBadPods, len(pods.Items))
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("want %v containers, got the wrong number of containers on %d of %d pods", count, countBadPods, countPods)
	}

	return nil
}

// ExpectPodReady finds a deployment and keeps checking until the number of
// containers on the deployment's PodSpec.Containers == count. Returns error after 30 seconds
// if the containers do not match.
func (cc *TestCaseClient) ExpectPodReady(ctx context.Context, podSelector *metav1.LabelSelector, allOrAny string) error {

	var (
		countBadPods int
		countPods    int
	)

	return RetryUntilSuccess(24, DefaultRetryInterval, func() error {
		countBadPods = 0
		pods, err := ListPods(ctx, cc.Client, cc.Namespace, podSelector)
		if err != nil {
			return err
		}
		countPods = len(pods.Items)
		if len(pods.Items) == 0 {
			return fmt.Errorf("got 0 pods, want at least 1 pod")
		}
		for _, pod := range pods.Items {
			if !isPodReady(pod) {
				countBadPods++
			}
		}
		switch {
		case allOrAny == "all" && countBadPods > 0:
			return fmt.Errorf("got %d pods not ready of %d pods, want 0 pods not ready", countBadPods, len(pods.Items))
		case allOrAny == "any" && countBadPods == countPods:
			return fmt.Errorf("got  %d pods not ready of %d pods, want at least 1 pod ready ", countBadPods, len(pods.Items))
		default:
			return nil
		}
	})

}

func isPodReady(pod corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady &&
			condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// ExpectContainerCount finds a deployment and keeps checking until the number of
// containers on the deployment's PodSpec.Containers == count. Returns error after 30 seconds
// if the containers do not match.
func (cc *TestCaseClient) ExpectContainerCount(ctx context.Context, key types.NamespacedName, count int) error {

	var (
		got        int
		deployment = &appsv1.Deployment{}
	)
	err := RetryUntilSuccess(6, DefaultRetryInterval, func() error {
		err := cc.Client.Get(ctx, key, deployment)
		if err != nil {
			return err
		}
		got = len(deployment.Spec.Template.Spec.Containers)
		if got != count {
			return fmt.Errorf("deployment found, got %v, want %v containers", got, count)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("want %v containers, got %v number of containers did not resolve after waiting for reconcile", count, got)
	}

	return nil
}

// CreateDeploymentReplicaSetAndPods mimics the behavior of the deployment controller
// built into kubernetes. It creates one ReplicaSet and DeploymentSpec.Replicas pods
// with the correct labels and ownership annotations as if it were in a live cluster.
// This will make it easier to test and debug the behavior of our pod injection webhooks.
func (cc *TestCaseClient) CreateDeploymentReplicaSetAndPods(ctx context.Context, d *appsv1.Deployment) (*appsv1.ReplicaSet, []*corev1.Pod, error) {
	podTemplateHash := strconv.FormatUint(rand.Uint64(), 16)
	rs := &appsv1.ReplicaSet{
		TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet", APIVersion: "apps/metav1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", d.Name, podTemplateHash),
			Namespace: d.Namespace,
			Annotations: map[string]string{
				"deployment.kubernetes.io/desired-replicas": "2",
				"deployment.kubernetes.io/max-replicas":     "3",
				"deployment.kubernetes.io/revision":         "1",
			},
			Generation: 1,
			Labels: map[string]string{
				"app":               d.Spec.Template.Labels["app"],
				"enablewait":        "yes",
				"pod-template-hash": podTemplateHash,
			},
		},
		Spec: appsv1.ReplicaSetSpec{
			Replicas: d.Spec.Replicas,
			Selector: d.Spec.Selector,
			Template: d.Spec.Template,
		},
	}

	err := controllerutil.SetOwnerReference(d, rs, cc.Client.Scheme())
	if err != nil {
		return nil, nil, err
	}

	err = cc.Client.Create(ctx, rs)
	if err != nil {
		return nil, nil, err
	}

	var replicas int32
	if d.Spec.Replicas != nil {
		replicas = *d.Spec.Replicas
	} else {
		replicas = 1
	}
	var pods []*corev1.Pod
	for i := int32(0); i < replicas; i++ {
		podID := strconv.FormatUint(uint64(rand.Uint32()), 16)
		p := &corev1.Pod{
			TypeMeta: metav1.TypeMeta{Kind: "Pod", APIVersion: "v1"},
			ObjectMeta: metav1.ObjectMeta{
				Name:       fmt.Sprintf("%s-%s-%s", d.Name, podTemplateHash, podID),
				Namespace:  d.Namespace,
				Generation: 1,
				Labels: map[string]string{
					"app":               d.Spec.Template.Labels["app"],
					"enablewait":        "yes",
					"pod-template-hash": podTemplateHash,
				},
			},
			Spec: d.Spec.Template.Spec,
		}
		err = controllerutil.SetOwnerReference(rs, p, cc.Client.Scheme())
		if err != nil {
			return rs, nil, err
		}

		err = cc.Client.Create(ctx, p)
		if err != nil {
			return rs, nil, err
		}
		pods = append(pods, p)
	}
	return rs, pods, nil
}

// BuildAuthProxyWorkload creates an AuthProxyWorkload object with a
// single connection instance.
func BuildAuthProxyWorkload(key types.NamespacedName, connectionString string) *v1alpha1.AuthProxyWorkload {
	return &v1alpha1.AuthProxyWorkload{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.String(),
			Kind:       "AuthProxyWorkload",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
		Spec: v1alpha1.AuthProxyWorkloadSpec{

			Instances: []v1alpha1.InstanceSpec{{
				ConnectionString: connectionString,
				HostEnvName:      "DB_HOST",
				PortEnvName:      "DB_PORT",
			}},
		},
	}
}

// CreateAuthProxyWorkload creates an AuthProxyWorkload in the kubernetes cluster.
func (cc *TestCaseClient) CreateAuthProxyWorkload(ctx context.Context, key types.NamespacedName, appLabel string, connectionString string, kind string) error {
	proxy := BuildAuthProxyWorkload(key, connectionString)
	proxy.Spec.Workload = v1alpha1.WorkloadSelectorSpec{
		Kind: kind,
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": appLabel},
		},
	}
	proxy.Spec.AuthProxyContainer = &v1alpha1.AuthProxyContainerSpec{
		Image: cc.ProxyImageURL,
		Resources: &corev1.ResourceRequirements{
			Requests: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceCPU: *resource.NewMilliQuantity(500, resource.DecimalExponent),
			},
		},
	}
	err := cc.Client.Create(ctx, proxy)
	if err != nil {
		return fmt.Errorf("Unable to create entity %v", err)
	}
	return nil
}

// GetConditionStatus finds a condition where Condition.Type == condType and returns
// the status, or "" if no condition was found.
func GetConditionStatus(conditions []*metav1.Condition, condType string) metav1.ConditionStatus {
	for i := range conditions {
		if conditions[i].Type == condType {
			return conditions[i].Status
		}
	}
	return ""
}
