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

package tests

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testServiceAccountName                     = "rt-checkup-sa"
	testKiagnoseConfigMapAccessRoleName        = "kiagnose-configmap-access"
	testKiagnoseConfigMapAccessRoleBindingName = testKiagnoseConfigMapAccessRoleName
	testConfigMapName                          = "rt-checkup-config"
	testCheckupJobName                         = "rt-checkup"
)

var _ = Describe("Checkup execution", func() {
	var (
		configMap  *corev1.ConfigMap
		checkupJob *batchv1.Job
	)

	BeforeEach(func() {
		setupCheckupPermissions()

		var err error
		configMap = newConfigMap()
		_, err = client.CoreV1().ConfigMaps(testNamespace).Create(context.Background(), configMap, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		DeferCleanup(func() {
			err = client.CoreV1().ConfigMaps(testNamespace).Delete(context.Background(), testConfigMapName, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred())
		})

		checkupJob = newCheckupJob()
		_, err = client.BatchV1().Jobs(testNamespace).Create(context.Background(), checkupJob, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		DeferCleanup(func() {
			backgroundPropagationPolicy := metav1.DeletePropagationBackground
			err = client.BatchV1().Jobs(testNamespace).Delete(
				context.Background(),
				testCheckupJobName,
				metav1.DeleteOptions{PropagationPolicy: &backgroundPropagationPolicy},
			)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	It("should complete successfully", func() {
		Eventually(getJobConditions, 5*time.Minute, 5*time.Second).Should(
			ContainElement(MatchFields(IgnoreExtras, Fields{
				"Type":   Equal(batchv1.JobComplete),
				"Status": Equal(corev1.ConditionTrue),
			})))

		configMap, err := client.CoreV1().ConfigMaps(testNamespace).Get(context.Background(), testConfigMapName, metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())

		Expect(configMap.Data).NotTo(BeNil())
		Expect(configMap.Data["status.succeeded"]).To(Equal("true"), fmt.Sprintf("should succeed %+v", configMap.Data))
		Expect(configMap.Data["status.failureReason"]).To(BeEmpty(), fmt.Sprintf("should be empty %+v", configMap.Data))
	})
})

func setupCheckupPermissions() {
	var (
		err                                error
		checkupServiceAccount              *corev1.ServiceAccount
		kiagnoseConfigMapAccessRole        *rbacv1.Role
		kiagnoseConfigMapAccessRoleBinding *rbacv1.RoleBinding
	)

	checkupServiceAccount = newServiceAccount()
	_, err = client.CoreV1().ServiceAccounts(testNamespace).Create(context.Background(), checkupServiceAccount, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	DeferCleanup(func() {
		err = client.CoreV1().ServiceAccounts(testNamespace).Delete(context.Background(), testServiceAccountName, metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	kiagnoseConfigMapAccessRole = newKiagnoseConfigMapAccessRole()
	_, err = client.RbacV1().Roles(testNamespace).Create(context.Background(), kiagnoseConfigMapAccessRole, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())

	DeferCleanup(func() {
		err = client.RbacV1().Roles(testNamespace).Delete(context.Background(), testKiagnoseConfigMapAccessRoleName, metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	kiagnoseConfigMapAccessRoleBinding = newKiagnoseConfigMapAccessRoleBinding()
	_, err = client.RbacV1().RoleBindings(testNamespace).Create(
		context.Background(),
		kiagnoseConfigMapAccessRoleBinding,
		metav1.CreateOptions{},
	)
	Expect(err).NotTo(HaveOccurred())

	DeferCleanup(func() {
		err = client.RbacV1().RoleBindings(testNamespace).Delete(
			context.Background(),
			testKiagnoseConfigMapAccessRoleBindingName,
			metav1.DeleteOptions{},
		)
		Expect(err).NotTo(HaveOccurred())
	})
}

func newServiceAccount() *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name: testServiceAccountName,
		},
	}
}

func newKiagnoseConfigMapAccessRole() *rbacv1.Role {
	return &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name: testKiagnoseConfigMapAccessRoleName,
		},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"get", "update"},
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
			},
		},
	}
}

func newKiagnoseConfigMapAccessRoleBinding() *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: testKiagnoseConfigMapAccessRoleBindingName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: rbacv1.ServiceAccountKind,
				Name: testServiceAccountName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "Role",
			Name:     testKiagnoseConfigMapAccessRoleName,
		},
	}
}

func newConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: testConfigMapName,
		},
		Data: map[string]string{
			"spec.timeout":                                 "1m",
			"spec.param.targetNode":                        "my-node",
			"spec.param.guestImageSourcePVCNamespace":      testNamespace,
			"spec.param.guestImageSourcePVCName":           "my-rt-vm",
			"spec.param.oslatDuration":                     "6m",
			"spec.param.oslatLatencyThresholdMicroSeconds": "45",
		},
	}
}

func newCheckupJob() *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: testCheckupJobName,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: pointer(int32(0)),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					ServiceAccountName: testServiceAccountName,
					RestartPolicy:      corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:            "rt-checkup",
							Image:           testImageName,
							ImagePullPolicy: corev1.PullAlways,
							SecurityContext: newSecurityContext(),
							Env: []corev1.EnvVar{
								{
									Name:  "CONFIGMAP_NAMESPACE",
									Value: testNamespace,
								},
								{
									Name:  "CONFIGMAP_NAME",
									Value: testConfigMapName,
								},
							},
						},
					},
				},
			},
		},
	}
}

func newSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		AllowPrivilegeEscalation: pointer(false),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
		RunAsNonRoot: pointer(true),
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
	}
}

func pointer[T any](v T) *T {
	return &v
}

func getJobConditions() []batchv1.JobCondition {
	checkupJob, err := client.BatchV1().Jobs(testNamespace).Get(context.Background(), testCheckupJobName, metav1.GetOptions{})
	if err != nil {
		return []batchv1.JobCondition{}
	}

	return checkupJob.Status.Conditions
}