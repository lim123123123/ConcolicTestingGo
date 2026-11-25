/*
Copyright 2026.

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

package reconciler

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	testv1 "github.com/lim123123123/concolic-testing/pkg/reconcilerflow/testdata/podWithConfigMapOp/api/v1"
)

const (
	// typeAvailableCronJob represents the status of the CronJob reconciliation
	typeAvailableCronJob = "Available"
	// typeProgressingCronJob represents the status used when the CronJob is being reconciled
	typeProgressingCronJob = "Progressing"
	// typeDegradedCronJob represents the status used when the CronJob has encountered an error
	typeDegradedCronJob = "Degraded"
)

const (
	ownerKey = ".metadata.controller"
)

// PodWithConfigmapReconciler reconciles a PodWithConfigmap object
type PodWithConfigmapReconciler struct {
	Client client.Client // Use a field instead of just embeded interface as embeded interface are not supported in gobra
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=test.test.garmanov,resources=podwithconfigmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=test.test.garmanov,resources=podwithconfigmaps/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=test.test.garmanov,resources=podwithconfigmaps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the PodWithConfigmap object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.25.0/pkg/reconcile
func (r *PodWithConfigmapReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	log := logr.Discard()

	var podWithCfgMap testv1.PodWithConfigmap
	if err := r.Client.Get(ctx, req.NamespacedName, &podWithCfgMap); err != nil {
		if apierrors.IsNotFound(err) {
			// If the resource was deleted do nothing
			return reconcile.Result{}, nil
		}
		log.Error(err, "can't get the CRD resource")
		return reconcile.Result{}, err
	}

	// Initialize status conditions if not yet present
	if len(podWithCfgMap.Status.Conditions) == 0 {
		meta.SetStatusCondition(&podWithCfgMap.Status.Conditions, metav1.Condition{
			Type:    typeProgressingCronJob,
			Status:  metav1.ConditionUnknown,
			Reason:  "Reconciling",
			Message: "Starting reconciliation",
		})
		if err := r.Client.Status().Update(ctx, &podWithCfgMap); err != nil {
			log.Error(err, "Failed to update CronJob status")
			return reconcile.Result{}, err
		}

		if err := r.Client.Get(ctx, req.NamespacedName, &podWithCfgMap); err != nil {
			log.Error(err, "can't fetch updated cron job")
			return reconcile.Result{}, err
		}
	}

	// var cfgMaps corev1.ConfigMapList
	// if err := r.Client.List(ctx, &cfgMaps, client.InNamespace(req.Namespace), client.MatchingFields{ownerKey: req.Name}); err != nil {
	// 	log.Error(err, "unable to list child cfgMaps")
	// 	if fetchErr := r.Client.Get(ctx, req.NamespacedName, &podWithCfgMap); fetchErr != nil {
	// 		log.Error(fetchErr, "Failed to re-fetch cfgMaps")
	// 		return reconcile.Result{}, fetchErr
	// 	}
	// 	// Update status condition to reflect the error
	// 	meta.SetStatusCondition(&podWithCfgMap.Status.Conditions, metav1.Condition{
	// 		Type:    typeDegradedCronJob,
	// 		Status:  metav1.ConditionTrue,
	// 		Reason:  "ReconciliationError",
	// 		Message: fmt.Sprintf("Failed to list child cfgMaps: %v", err),
	// 	})
	// 	if statusErr := r.Client.Status().Update(ctx, &podWithCfgMap); statusErr != nil {
	// 		log.Error(statusErr, "Failed to update PodWithCfgMap status")
	// 	}
	// 	return reconcile.Result{}, err
	// }

	// // TODO: what about finalizers?
	// // What about different versions of the resource?
	// var cfgMap *corev1.ConfigMap
	// if len(cfgMaps.Items) > 1 {
	// 	log.Error(errors.New("too many cfgMaps"), "found to many deployments: %d", len(cfgMaps.Items))
	// 	return reconcile.Result{}, nil
	// }
	// if len(cfgMaps.Items) == 0 {
	// 	newCfgMap, err := getConfigMap(r.Scheme, &podWithCfgMap)
	// 	if err != nil {
	// 		log.Error(err, "can't generate cfg map")
	// 		return reconcile.Result{}, err
	// 	}

	// 	if err := r.Client.Create(ctx, newCfgMap); err != nil {
	// 		log.Error(err, "can't create cfg map")
	// 		return reconcile.Result{}, err
	// 	}
	// 	cfgMap = newCfgMap
	// } else {
	// 	cfgMap = &cfgMaps.Items[0]
	// }

	// var deployments appsv1.DeploymentList
	// if err := r.Client.List(ctx, &deployments, client.InNamespace(req.Namespace), client.MatchingFields{ownerKey: req.Name}); err != nil {
	// 	log.Error(err, "unable to list child deployments")
	// 	if fetchErr := r.Client.Get(ctx, req.NamespacedName, &podWithCfgMap); fetchErr != nil {
	// 		log.Error(fetchErr, "Failed to re-fetch podWithConfiguration")
	// 		return reconcile.Result{}, fetchErr
	// 	}
	// 	// Update status condition to reflect the error
	// 	meta.SetStatusCondition(&podWithCfgMap.Status.Conditions, metav1.Condition{
	// 		Type:    typeDegradedCronJob,
	// 		Status:  metav1.ConditionTrue,
	// 		Reason:  "ReconciliationError",
	// 		Message: fmt.Sprintf("Failed to list child deployments: %v", err),
	// 	})
	// 	if statusErr := r.Client.Status().Update(ctx, &podWithCfgMap); statusErr != nil {
	// 		log.Error(statusErr, "Failed to update PodWithCfgMap status")
	// 	}
	// 	return reconcile.Result{}, err
	// }

	// if len(deployments.Items) > 1 {
	// 	log.Error(errors.New("too many deployments"), "found to many deployments: %d", len(deployments.Items))
	// 	return reconcile.Result{}, nil
	// }
	// if len(deployments.Items) == 0 {
	// 	deployment, err := getDeployment(r.Scheme, &podWithCfgMap, cfgMap.Name)
	// 	if err != nil {
	// 		log.Error(err, "can't generate deployment")
	// 		return reconcile.Result{}, err
	// 	}

	// 	if err := r.Client.Create(ctx, deployment); err != nil {
	// 		log.Error(err, "can't create deployment")
	// 		return reconcile.Result{}, err
	// 	}
	// }

	return reconcile.Result{}, nil
}

func getConfigMap(scheme *runtime.Scheme, podWithCfgMap *testv1.PodWithConfigmap) (*corev1.ConfigMap, error) {
	magicCfgVal := "some value"
	if podWithCfgMap.Spec.MagicCfgVal != "" {
		magicCfgVal = podWithCfgMap.Spec.MagicCfgVal
	}

	cfgMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("pod-%s-cfg", podWithCfgMap.Name),
			Namespace: podWithCfgMap.Namespace,
			// TODO: set controller reference
		},
		Data: map[string]string{
			"some-key": magicCfgVal,
		},
	}

	return cfgMap, setControllerReference(podWithCfgMap, cfgMap, scheme)
}

func getDeployment(scheme *runtime.Scheme, podWithCfgMap *testv1.PodWithConfigmap, cfgMapName string) (*appsv1.Deployment, error) {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("pod-%s-deployment", podWithCfgMap.Name),
			Namespace: podWithCfgMap.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"podWithCfgMapName": podWithCfgMap.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"podWithCfgMapName": podWithCfgMap.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "main",
							Image: podWithCfgMap.Spec.PodImage,
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: cfgMapName,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return deployment, setControllerReference(podWithCfgMap, deployment, scheme)
}

func setControllerReference(owner *testv1.PodWithConfigmap, obj metav1.Object, scheme *runtime.Scheme) error {
	gvk, err := apiutil.GVKForObject(owner, scheme)
	if err != nil {
		return err
	}
	obj.SetOwnerReferences([]metav1.OwnerReference{*metav1.NewControllerRef(owner, gvk)})
	return nil
}
