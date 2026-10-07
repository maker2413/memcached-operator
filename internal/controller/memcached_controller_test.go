package controller

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cachev1alpha1 "github.com/maker2413/memcached-operator/api/v1alpha1"
)

var _ = Describe("Memcached Controller", func() {
	Context("When reconciling a resource", func() {
		It("should create and resize an owned Deployment and report availability", func() {
			resourceKey := types.NamespacedName{Name: "test-memcached", Namespace: "default"}
			memcached := &cachev1alpha1.Memcached{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceKey.Name,
					Namespace: resourceKey.Namespace,
				},
				Spec: cachev1alpha1.MemcachedSpec{Size: new(int32(2))},
			}
			Expect(k8sClient.Create(ctx, memcached)).To(Succeed())
			DeferCleanup(func() {
				// envtest does not run the garbage collector, so delete the Deployment explicitly.
				deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
					Name:      resourceKey.Name,
					Namespace: resourceKey.Namespace,
				}}
				Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, deployment))).To(Succeed())
				Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, memcached))).To(Succeed())
			})

			reconciler := &MemcachedReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
			request := ctrl.Request{NamespacedName: resourceKey}

			By("creating a Deployment for the Memcached resource")
			result, err := reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))
			deployment := &appsv1.Deployment{}
			Expect(k8sClient.Get(ctx, resourceKey, deployment)).To(Succeed())
			Expect(deployment.Spec.Replicas).To(Equal(new(int32(2))))
			Expect(metav1.IsControlledBy(deployment, memcached)).To(BeTrue())

			By("resizing the Deployment after the desired size changes")
			Expect(k8sClient.Get(ctx, resourceKey, memcached)).To(Succeed())
			memcached.Spec.Size = new(int32(3))
			Expect(k8sClient.Update(ctx, memcached)).To(Succeed())
			result, err = reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))
			Expect(k8sClient.Get(ctx, resourceKey, deployment)).To(Succeed())
			Expect(deployment.Spec.Replicas).To(Equal(new(int32(3))))

			By("reporting availability once the desired size is reconciled")
			_, err = reconciler.Reconcile(ctx, request)
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Get(ctx, resourceKey, memcached)).To(Succeed())
			Expect(meta.IsStatusConditionTrue(memcached.Status.Conditions, typeAvailableMemcached)).To(BeTrue())
		})
	})
})
