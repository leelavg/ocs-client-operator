/*
Copyright 2023 Red Hat, Inc.

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
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/red-hat-storage/ocs-client-operator/api/v1alpha1"
	"github.com/red-hat-storage/ocs-client-operator/pkg/utils"

	"github.com/go-logr/logr"
	opv1a1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-lib/conditions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ocsClientOperatorSubscriptionPackageName = "ocs-client-operator"
	subscriptionChannelNamePrefix            = "stable-"
	controllerName                           = "upgrade"
	upgradeConditionReason                   = "ConstraintsSatisfied"
)

type UpgradeReconciler struct {
	client.Client
	ctx context.Context
	log logr.Logger

	Scheme            *runtime.Scheme
	OperatorCondition conditions.Condition
	OperatorNamespace string
	PlatformVersion   string
}

func (r *UpgradeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	enqueueNamespace := handler.EnqueueRequestsFromMapFunc(
		func(_ context.Context, _ client.Object) []reconcile.Request {
			return []reconcile.Request{
				{
					NamespacedName: types.NamespacedName{
						Namespace: r.OperatorNamespace,
					},
				},
			}
		},
	)
	storageClientOperatorVersionPredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.ObjectOld == nil || e.ObjectNew == nil {
				return false
			}

			oldObj, _ := e.ObjectOld.(*v1alpha1.StorageClient)
			newObj, _ := e.ObjectNew.(*v1alpha1.StorageClient)
			if oldObj == nil || newObj == nil {
				return false
			}

			currentVersion, _ := semver.Make(newObj.Status.Operator.CurrentVersion)
			desiredVersion, _ := semver.Make(newObj.Status.Operator.DesiredVersion)
			if currentVersion.Major != desiredVersion.Major || currentVersion.Minor != desiredVersion.Minor {
				return true
			}

			return false
		},
	}
	ocsClientOperatorSubscriptionPredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.ObjectOld == nil || e.ObjectNew == nil {
				return false
			}

			oldObj, _ := e.ObjectOld.(*opv1a1.Subscription)
			newObj, _ := e.ObjectNew.(*opv1a1.Subscription)
			if oldObj == nil || newObj == nil {
				return false
			}

			if oldObj.Spec.Package != ocsClientOperatorSubscriptionPackageName {
				return false
			}

			return oldObj.Spec.Channel != newObj.Spec.Channel
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		Named(controllerName).
		Watches(&v1alpha1.StorageClient{}, enqueueNamespace, builder.WithPredicates(storageClientOperatorVersionPredicate)).
		Watches(&opv1a1.Subscription{}, enqueueNamespace, builder.WithPredicates(ocsClientOperatorSubscriptionPredicate)).
		Complete(r)
}

//+kubebuilder:rbac:groups=operators.coreos.com,resources=subscriptions,verbs=get;list;patch

func (r *UpgradeReconciler) Reconcile(ctx context.Context, _ ctrl.Request) (ctrl.Result, error) {
	r.ctx = ctx
	r.log = log.FromContext(ctx)
	r.log.Info("starting reconcile")

	conditionStatus := metav1.ConditionTrue
	conditionMessage := "No component reported errors."
	result, err := r.reconcilePhases()
	if err != nil {
		// TODO (lgangava): we are being very restrictive as of now, not upgrading if we hit any error
		// differentiate between reconciler errors and not upgradeable errors
		conditionStatus = metav1.ConditionFalse
		conditionMessage = fmt.Sprintf("Operator couldn't be upgraded due to: %v", err)
		r.log.Error(err, "an error occured during reconcilePhases")
	}
	err = errors.Join(err, r.setOperatorCondition(conditionStatus, conditionMessage))

	r.log.Info("reconciling completed")
	return result, err
}

func (r *UpgradeReconciler) reconcilePhases() (ctrl.Result, error) {
	if err := r.reconcileSubscriptionChannel(); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *UpgradeReconciler) reconcileSubscriptionChannel() error {
	storageClientList := &v1alpha1.StorageClientList{}
	if err := r.list(storageClientList); err != nil {
		return fmt.Errorf("failed to list storageclients %v", err)
	}

	if len(storageClientList.Items) == 0 {
		r.log.Info("no storageclients exist")
		return nil
	}

	desiredVersions := make([]string, 0, len(storageClientList.Items))
	for idx := range storageClientList.Items {
		desiredVersion := storageClientList.Items[idx].Status.Operator.DesiredVersion
		if desiredVersion != "" {
			desiredVersions = append(desiredVersions, desiredVersion)
		}
	}
	if len(storageClientList.Items) != len(desiredVersions) {
		return fmt.Errorf("not all StorageClients status has operator desiredVersion set")
	}

	desiredVersion, err := getOldestVersion(desiredVersions)
	if err != nil {
		return fmt.Errorf("unable to find oldest desiredVersion that operator should be in")
	}

	subscriptionList := &opv1a1.SubscriptionList{}
	if err = r.list(subscriptionList, client.InNamespace(r.OperatorNamespace)); err != nil {
		return fmt.Errorf("failed to list subscriptions in namespace %q: %v", r.OperatorNamespace, err)
	}

	ocsClientOperatorSubscription := utils.Find(subscriptionList.Items, func(sub *opv1a1.Subscription) bool {
		return sub.Spec.Package == ocsClientOperatorSubscriptionPackageName
	})

	if ocsClientOperatorSubscription == nil {
		return fmt.Errorf("unable to find subscription matching the package name %q", ocsClientOperatorSubscriptionPackageName)
	}

	channel := ocsClientOperatorSubscription.Spec.Channel
	channelVersion, err := getVersionFromChannel(channel)
	if err != nil {
		return fmt.Errorf("failed to convert channel name %q to semver %v", channel, err)
	}

	// decide whether to patch the subscription or not
	platformVersion, _ := semver.Make(r.PlatformVersion)
	if desiredVersion.Major > platformVersion.Major || desiredVersion.Minor > platformVersion.Minor {
		r.log.Info("backing off from updating subscription as operator version can become ahead of platform version")
	} else if desiredVersion.Major == channelVersion.Minor && desiredVersion.Minor == channelVersion.Minor+1 {
		// initiate auto upgrade by patching the channel name of the subscription
		subscription := &opv1a1.Subscription{}
		subscription.Name = ocsClientOperatorSubscription.Name
		subscription.Namespace = ocsClientOperatorSubscription.Namespace
		subscription.Spec.Channel = getChannelFromVersion(desiredVersion)
		jsonPatch, _ := json.Marshal(subscription)
		if err = r.patch(subscription, client.RawPatch(types.MergePatchType, jsonPatch)); err != nil {
			return fmt.Errorf("failed to update subscription %q/%q channel name to %q. %v",
				subscription.Namespace, subscription.Name, subscription.Spec.Channel, err)
		}
		r.log.Info("updated subscription channel name",
			"namespace", subscription.Namespace, "name", subscription.Name,
			"from", ocsClientOperatorSubscription.Spec.Channel, "to", subscription.Spec.Channel)
	}

	// decide whether to surface we are upgradeable or not
	if desiredVersion.Major < channelVersion.Major || desiredVersion.Minor < channelVersion.Minor {
		// someone manually might have initiated an upgrade, best effort to stop it via conditions
		return fmt.Errorf("subscription channel name is changed outside of operator")
	}

	// TODO (lgangava): as of now there's no way to distingush desiredVersion vs stale desiredVersion

	return nil
}

func (r *UpgradeReconciler) setOperatorCondition(isUpgradeable metav1.ConditionStatus, message string) error {
	return r.OperatorCondition.
		Set(r.ctx, isUpgradeable, conditions.WithReason(upgradeConditionReason), conditions.WithMessage(message))
}

func (r *UpgradeReconciler) list(list client.ObjectList, opts ...client.ListOption) error {
	return r.Client.List(r.ctx, list, opts...)
}

func (r *UpgradeReconciler) patch(obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	return r.Client.Patch(r.ctx, obj, patch, opts...)
}

func getOldestVersion(versions []string) (semver.Version, error) {
	if len(versions) < 1 {
		return semver.Version{}, fmt.Errorf("empty list of versions supplied")
	}

	semverVersions := make([]semver.Version, len(versions))
	for i := range versions {
		// these versions are operator set which are queried from values set during build time
		// and so discarding the error
		semverVersions[i], _ = semver.Make(versions[i])
	}

	semver.Sort(semverVersions)
	return semverVersions[0], nil
}

// returns the name of the subscription channel corresponding to the supplied operator version
func getChannelFromVersion(version semver.Version) string {
	return fmt.Sprintf("%s%d.%d", subscriptionChannelNamePrefix, version.Major, version.Minor)
}

// returns the semver version of operator version corresponding to the subscription channel name
func getVersionFromChannel(name string) (semver.Version, error) {
	version, _ := strings.CutPrefix(name, subscriptionChannelNamePrefix)
	return semver.Make(version)
}
