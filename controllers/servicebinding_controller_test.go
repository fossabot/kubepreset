/*
Copyright 2020 The KubePreset Authors

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

package controllers_test

import (
	"context"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apixv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appv1alpha1 "github.com/kubepreset/kubepreset/api/v1alpha1"
)

/*
```
apiVersion: service.binding/v1alpha2
kind: ServiceBinding
metadata:
  name: account-service
spec:
  application:
    apiVersion: apps/v1
    kind:       Deployment
    name:       online-banking

  service:
    apiVersion: com.example/v1alpha1
    kind:       AccountService
    name:       prod-account-service

status:
  conditions:
  - type:   Ready
    status: 'True'
```

0. (ServiceBinding CRD must be already created)
1. Create ProvisionedService CRD (
2. Create ProvisionedService CR
3. Create Deployment
4. Create ServiceBinding CR
5. Check status conditions and ensure type `Ready` with value `True` exists
*/

var _ = Describe("ServiceBinding Controller:", func() {

	const (
		timeout  = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When creating ServiceBinding with ProvisionedService", func() {

		var testNamespace string
		var ns *corev1.Namespace
		ctx := context.Background()

		BeforeEach(func() {
			//k8sClient = k8sManager.GetClient()
			testNamespace = "testns-" + uuid.Must(uuid.NewRandom()).String()
			ns = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: testNamespace,
				},
			}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, ns)).Should(Succeed())
		})

		It("should update the ServiceBinding status conditions for type `Ready` with value `True`", func() {
			By("Creating BackingService CRD")
			backingServiceCRD := &apixv1beta1.CustomResourceDefinition{
				TypeMeta: metav1.TypeMeta{
					Kind:       "CustomResourceDefinition",
					APIVersion: "apiextensions.k8s.io/v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "backingservices.app.example.org",
					Namespace: testNamespace,
				},
				Spec: apixv1beta1.CustomResourceDefinitionSpec{
					Group: "app.example.org",
					Versions: []apixv1beta1.CustomResourceDefinitionVersion{{
						Name:    "v1alpha1",
						Served:  true,
						Storage: true,
					}},
					Names: apixv1beta1.CustomResourceDefinitionNames{
						Plural: "backingservices",
						Kind:   "BackingService",
					},
					Scope: apixv1beta1.ClusterScoped,
				},
			}
			Expect(k8sClient.Create(ctx, backingServiceCRD)).Should(Succeed())

			backingServiceCRDLookupKey := client.ObjectKey{Name: "backingservices.app.example.org", Namespace: testNamespace}
			createdBackingServiceCRD := &apixv1beta1.CustomResourceDefinition{}

			By("Verifying BackingService CRD")
			// Retry getting newly created BackingService CRD
			// Important: This is required as it is going to be used immediately
			Eventually(func() bool {
				// FIXME: `k8sClient` seems to be not working
				err := k8sClient2.Get(ctx, backingServiceCRDLookupKey, createdBackingServiceCRD)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Creating BackingService CR")
			backingServiceCR := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"kind":       "BackingService",
					"apiVersion": "app.example.org/v1alpha1",
					"Namespace":  testNamespace,
					"metadata": map[string]interface{}{
						"name": "back1",
					},
					"status": map[string]interface{}{
						"binding": map[string]interface{}{
							"name": "secret1",
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, backingServiceCR)).Should(Succeed())

			matchLabels := map[string]string{
				"environment": "test",
			}

			app := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "app",
					Labels:    matchLabels,
					Namespace: testNamespace,
				},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: matchLabels,
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "app",
							Namespace: testNamespace,
							Labels:    matchLabels,
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name:    "busybox",
								Image:   "busybox:latest",
								Command: []string{"sleep", "3600"},
							}},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, app)).Should(Succeed())

			sb := &appv1alpha1.ServiceBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sb",
					Namespace: testNamespace,
				},
				Spec: appv1alpha1.ServiceBindingSpec{
					Application: &appv1alpha1.Application{
						APIVersion: "apps/v1",
						Kind:       "Deployment",
						Name:       "app",
					},
					Service: &appv1alpha1.Service{
						APIVersion: "app.example.org/v1alpha1",
						Kind:       "BackingService",
						Name:       "back1",
					},
				},
			}
			Expect(k8sClient.Create(ctx, sb)).Should(Succeed())

			serviceBindingLookupKey := client.ObjectKey{Name: "sb", Namespace: testNamespace}
			createdServiceBinding := &appv1alpha1.ServiceBinding{}

			// Retry getting newly created ServiceBinding; the status may not be immediately reflected.
			Eventually(func() bool {
				err := k8sClient.Get(ctx, serviceBindingLookupKey, createdServiceBinding)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(len(createdServiceBinding.Status.Conditions)).To(Equal(0))
		})
	})
})

//Reference: appv1alpha1.Reference{
