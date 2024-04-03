/*
Copyright 2022 Red Hat, Inc.

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
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/red-hat-storage/ocs-client-operator/api/v1alpha1"
	"github.com/red-hat-storage/ocs-client-operator/pkg/utils"

	configv1 "github.com/openshift/api/config/v1"
	ocpv1 "github.com/openshift/api/template/v1"
	opv1a1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	providerClient "github.com/red-hat-storage/ocs-operator/v4/services/provider/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// grpcCallNames
	OnboardConsumer       = "OnboardConsumer"
	OffboardConsumer      = "OffboardConsumer"
	GetStorageConfig      = "GetStorageConfig"
	AcknowledgeOnboarding = "AcknowledgeOnboarding"

	storageClientAnnotationKey          = "ocs.openshift.io/storageclient"
	storageClientNameLabel              = "ocs.openshift.io/storageclient.name"
	storageClientNamespaceLabel         = "ocs.openshift.io/storageclient.namespace"
	storageClientFinalizer              = "storageclient.ocs.openshift.io"
	defaultClaimsOwnerAnnotationKey     = "ocs.openshift.io/storageclaim.owner"
	defaultClaimsProcessedAnnotationKey = "ocs.openshift.io/storageclaim.processed"
	defaultBlockStorageClaim            = "ocs-storagecluster-ceph-rbd"
	defaultSharedfileStorageClaim       = "ocs-storagecluster-cephfs"

	// indexes for caching
	storageProviderEndpointIndexName = "index:storageProviderEndpoint"
	storageClientAnnotationIndexName = "index:storageClientAnnotation"
	defaultClaimsOwnerIndexName      = "index:defaultClaimsOwner"

	csvPrefix = "ocs-client-operator"
)

// StorageClientReconciler reconciles a StorageClient object
type StorageClientReconciler struct {
	ctx context.Context
	client.Client
	Log      klog.Logger
	Scheme   *runtime.Scheme
	recorder *utils.EventReporter

	OperatorNamespace string
}

// SetupWithManager sets up the controller with the Manager.
func (s *StorageClientReconciler) SetupWithManager(mgr ctrl.Manager) error {
	ctx := context.Background()
	// Index should be registered before cache start.
	// IndexField is used to filter out the objects that already exists with
	// status.phase != failed This will help in blocking
	// the new storageclient creation if there is already with one with same
	// provider endpoint with status.phase != failed
	_ = mgr.GetCache().IndexField(ctx, &v1alpha1.StorageClient{}, storageProviderEndpointIndexName, func(o client.Object) []string {
		res := []string{}
		if o.(*v1alpha1.StorageClient).Status.Phase != v1alpha1.StorageClientFailed {
			res = append(res, o.(*v1alpha1.StorageClient).Spec.StorageProviderEndpoint)
		}
		return res
	})

	if err := mgr.GetCache().IndexField(ctx, &v1alpha1.StorageClaim{}, storageClientAnnotationIndexName, func(obj client.Object) []string {
		return []string{obj.GetAnnotations()[storageClientAnnotationKey]}
	}); err != nil {
		return fmt.Errorf("unable to set up FieldIndexer for storageclient annotation: %v", err)
	}

	if err := mgr.GetCache().IndexField(ctx, &v1alpha1.StorageClient{}, defaultClaimsOwnerIndexName, func(obj client.Object) []string {
		return []string{obj.GetAnnotations()[defaultClaimsOwnerAnnotationKey]}
	}); err != nil {
		return fmt.Errorf("unable to set up FieldIndexer for storageclient owner annotation: %v", err)
	}

	enqueueStorageClientRequest := handler.EnqueueRequestsFromMapFunc(
		func(_ context.Context, obj client.Object) []reconcile.Request {
			annotations := obj.GetAnnotations()
			if _, found := annotations[storageClaimAnnotation]; found {
				return []reconcile.Request{{
					NamespacedName: types.NamespacedName{
						Name: obj.GetName(),
					},
				}}
			}
			return []reconcile.Request{}
		})
	s.recorder = utils.NewEventReporter(mgr.GetEventRecorderFor("controller_storageclient"))
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.StorageClient{}).
		Watches(&v1alpha1.StorageClaim{}, enqueueStorageClientRequest).
		Watches(&ocpv1.Template{}, enqueueStorageClientRequest).
		Complete(s)
}

//+kubebuilder:rbac:groups=ocs.openshift.io,resources=storageclients,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ocs.openshift.io,resources=storageclients/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ocs.openshift.io,resources=storageclients/finalizers,verbs=update
//+kubebuilder:rbac:groups=config.openshift.io,resources=clusterversions,verbs=get;list;watch
//+kubebuilder:rbac:groups=batch,resources=cronjobs,verbs=get;list;create;update;watch;delete
//+kubebuilder:rbac:groups=operators.coreos.com,resources=clusterserviceversions,verbs=get;list;watch

func (s *StorageClientReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var err error
	s.ctx = ctx
	s.Log = log.FromContext(ctx, "StorageClient", req)
	s.Log.Info("Reconciling StorageClient")

	// Fetch the StorageClient instance
	instance := &v1alpha1.StorageClient{}
	instance.Name = req.Name
	instance.Namespace = req.Namespace

	if err = s.Client.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, instance); err != nil {
		if kerrors.IsNotFound(err) {
			s.Log.Info("StorageClient resource not found. Ignoring since object must be deleted.")
			return reconcile.Result{}, nil
		}
		s.Log.Error(err, "Failed to get StorageClient.")
		return reconcile.Result{}, fmt.Errorf("failed to get StorageClient: %v", err)
	}

	// Dont Reconcile the StorageClient if it is in failed state
	if instance.Status.Phase == v1alpha1.StorageClientFailed {
		return reconcile.Result{}, nil
	}

	result, reconcileErr := s.reconcilePhases(instance)

	// Apply status changes to the StorageClient
	statusErr := s.Client.Status().Update(ctx, instance)
	if statusErr != nil {
		s.Log.Error(statusErr, "Failed to update StorageClient status.")
	}
	if reconcileErr != nil {
		err = reconcileErr
	} else if statusErr != nil {
		err = statusErr
	}
	return result, err
}

func (s *StorageClientReconciler) reconcilePhases(instance *v1alpha1.StorageClient) (ctrl.Result, error) {
	storageClientListOption := []client.ListOption{
		client.MatchingFields{storageProviderEndpointIndexName: instance.Spec.StorageProviderEndpoint},
	}

	storageClientList := &v1alpha1.StorageClientList{}
	if err := s.Client.List(s.ctx, storageClientList, storageClientListOption...); err != nil {
		s.Log.Error(err, "unable to list storage clients")
		return ctrl.Result{}, err
	}

	if len(storageClientList.Items) > 1 {
		s.Log.Info("one StorageClient is allowed per namespace but found more than one. Rejecting new request.")
		instance.Status.Phase = v1alpha1.StorageClientFailed
		// Dont Reconcile again	as there is already a StorageClient with same provider endpoint
		return reconcile.Result{}, nil
	}

	externalClusterClient, err := s.newExternalClusterClient(instance)
	if err != nil {
		return reconcile.Result{}, err
	}
	defer externalClusterClient.Close()

	// deletion phase
	if !instance.GetDeletionTimestamp().IsZero() {
		return s.deletionPhase(instance, externalClusterClient)
	}

	// ensure finalizer
	if !contains(instance.GetFinalizers(), storageClientFinalizer) {
		instance.Status.Phase = v1alpha1.StorageClientInitializing
		s.Log.Info("Finalizer not found for StorageClient. Adding finalizer.", "StorageClient", klog.KRef(instance.Namespace, instance.Name))
		instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, storageClientFinalizer)
		if err := s.Client.Update(s.ctx, instance); err != nil {
			s.Log.Info("Failed to update StorageClient with finalizer.", "StorageClient", klog.KRef(instance.Namespace, instance.Name))
			return reconcile.Result{}, fmt.Errorf("failed to update StorageClient with finalizer: %v", err)
		}
	}

	if instance.Status.ConsumerID == "" {
		return s.onboardConsumer(instance, externalClusterClient)
	} else if instance.Status.Phase == v1alpha1.StorageClientOnboarding {
		return s.acknowledgeOnboarding(instance, externalClusterClient)
	}

	if res, err := s.reconcileClientStatusReporterJob(instance); err != nil {
		return res, err
	}

	if err := s.reconcileDefaultStorageClaims(instance); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (s *StorageClientReconciler) deletionPhase(instance *v1alpha1.StorageClient, externalClusterClient *providerClient.OCSProviderClient) (ctrl.Result, error) {
	// TODO Need to take care of deleting the SCC created for this
	// storageClient and also the default SCC created for this storageClient
	if contains(instance.GetFinalizers(), storageClientFinalizer) {
		instance.Status.Phase = v1alpha1.StorageClientOffboarding
		err := s.verifyNoStorageClaimsExist(instance)
		if err != nil {
			s.Log.Error(err, "still storageclaims exist for this storageclient")
			return reconcile.Result{}, fmt.Errorf("still storageclaims exist for this storageclient: %v", err)
		}
		if res, err := s.offboardConsumer(instance, externalClusterClient); err != nil {
			s.Log.Error(err, "Offboarding in progress.")
		} else if !res.IsZero() {
			// result is not empty
			return res, nil
		}

		cronJob := &batchv1.CronJob{}
		cronJob.Name = getStatusReporterName(instance.Namespace, instance.Name)
		cronJob.Namespace = s.OperatorNamespace

		if err := s.delete(cronJob); err != nil {
			s.Log.Error(err, "Failed to delete the status reporter job")
			return reconcile.Result{}, fmt.Errorf("failed to delete the status reporter job: %v", err)
		}

		if err := s.deleteDefaultStorageClaims(instance); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to delete default storageclaims: %v", err)
		}

		s.Log.Info("removing finalizer from StorageClient.", "StorageClient", klog.KRef(instance.Namespace, instance.Name))
		// Once all finalizers have been removed, the object will be deleted
		instance.ObjectMeta.Finalizers = remove(instance.ObjectMeta.Finalizers, storageClientFinalizer)
		if err := s.Client.Update(s.ctx, instance); err != nil {
			s.Log.Info("Failed to remove finalizer from StorageClient", "StorageClient", klog.KRef(instance.Namespace, instance.Name))
			return reconcile.Result{}, fmt.Errorf("failed to remove finalizer from StorageClient: %v", err)
		}
	}
	s.Log.Info("StorageClient is offboarded", "StorageClient", klog.KRef(instance.Namespace, instance.Name))
	return reconcile.Result{}, nil
}

// newExternalClusterClient returns the *providerClient.OCSProviderClient
func (s *StorageClientReconciler) newExternalClusterClient(instance *v1alpha1.StorageClient) (*providerClient.OCSProviderClient, error) {

	ocsProviderClient, err := providerClient.NewProviderClient(
		s.ctx, instance.Spec.StorageProviderEndpoint, time.Second*10)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new provider client: %v", err)
	}

	return ocsProviderClient, nil
}

// onboardConsumer makes an API call to the external storage provider cluster for onboarding
func (s *StorageClientReconciler) onboardConsumer(instance *v1alpha1.StorageClient, externalClusterClient *providerClient.OCSProviderClient) (reconcile.Result, error) {

	// TODO Need to find a way to get rid of ClusterVersion here as it is OCP
	// specific one.
	clusterVersion := &configv1.ClusterVersion{}
	err := s.Client.Get(s.ctx, types.NamespacedName{Name: "version"}, clusterVersion)
	if err != nil {
		s.Log.Error(err, "failed to get the clusterVersion version of the OCP cluster")
		return reconcile.Result{}, fmt.Errorf("failed to get the clusterVersion version of the OCP cluster: %v", err)
	}

	// TODO Have a version file corresponding to the release
	csvList := opv1a1.ClusterServiceVersionList{}
	if err = s.list(&csvList, client.InNamespace(s.OperatorNamespace)); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to list csv resources in ns: %v, err: %v", s.OperatorNamespace, err)
	}
	csv := utils.Find(csvList.Items, func(csv *opv1a1.ClusterServiceVersion) bool {
		return strings.HasPrefix(csv.Name, csvPrefix)
	})
	if csv == nil {
		return reconcile.Result{}, fmt.Errorf("unable to find csv with prefix %q", csvPrefix)
	}
	name := fmt.Sprintf("storageconsumer-%s", clusterVersion.Spec.ClusterID)
	onboardRequest := providerClient.NewOnboardConsumerRequest().
		SetConsumerName(name).
		SetOnboardingTicket(instance.Spec.OnboardingTicket).
		SetClientOperatorVersion(csv.Spec.Version.String())
	response, err := externalClusterClient.OnboardConsumer(s.ctx, onboardRequest)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			s.logGrpcErrorAndReportEvent(instance, OnboardConsumer, err, st.Code())
		}
		return reconcile.Result{}, fmt.Errorf("failed to onboard consumer: %v", err)
	}

	if response.StorageConsumerUUID == "" {
		err = fmt.Errorf("storage provider response is empty")
		s.Log.Error(err, "empty response")
		return reconcile.Result{}, err
	}

	instance.Status.ConsumerID = response.StorageConsumerUUID
	instance.Status.Phase = v1alpha1.StorageClientOnboarding

	s.Log.Info("onboarding started")
	return reconcile.Result{Requeue: true}, nil
}

func (s *StorageClientReconciler) acknowledgeOnboarding(instance *v1alpha1.StorageClient, externalClusterClient *providerClient.OCSProviderClient) (reconcile.Result, error) {

	_, err := externalClusterClient.AcknowledgeOnboarding(s.ctx, instance.Status.ConsumerID)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			s.logGrpcErrorAndReportEvent(instance, AcknowledgeOnboarding, err, st.Code())
		}
		s.Log.Error(err, "Failed to acknowledge onboarding.")
		return reconcile.Result{}, fmt.Errorf("failed to acknowledge onboarding: %v", err)
	}
	instance.Status.Phase = v1alpha1.StorageClientConnected

	s.Log.Info("Onboarding is acknowledged successfully.")
	return reconcile.Result{Requeue: true}, nil
}

// offboardConsumer makes an API call to the external storage provider cluster for offboarding
func (s *StorageClientReconciler) offboardConsumer(instance *v1alpha1.StorageClient, externalClusterClient *providerClient.OCSProviderClient) (reconcile.Result, error) {

	_, err := externalClusterClient.OffboardConsumer(s.ctx, instance.Status.ConsumerID)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			s.logGrpcErrorAndReportEvent(instance, OffboardConsumer, err, st.Code())
		}
		return reconcile.Result{}, fmt.Errorf("failed to offboard consumer: %v", err)
	}

	return reconcile.Result{}, nil
}

func (s *StorageClientReconciler) verifyNoStorageClaimsExist(instance *v1alpha1.StorageClient) error {

	storageClaims := &v1alpha1.StorageClaimList{}
	err := s.Client.List(s.ctx,
		storageClaims,
		client.MatchingFields{storageClientAnnotationIndexName: client.ObjectKeyFromObject(instance).String()},
		client.Limit(1),
	)
	if err != nil {
		return fmt.Errorf("failed to list storageClaims: %v", err)
	}

	if len(storageClaims.Items) != 0 {
		err = fmt.Errorf("Failed to cleanup resources. storageClaims are present."+
			"Delete all storageClaims corresponding to storageclient %q for the cleanup to proceed", client.ObjectKeyFromObject(instance))
		s.recorder.ReportIfNotPresent(instance, corev1.EventTypeWarning, "Cleanup", err.Error())
		s.Log.Error(err, "Waiting for all storageClaims to be deleted.")
		return err
	}

	return nil
}
func (s *StorageClientReconciler) logGrpcErrorAndReportEvent(instance *v1alpha1.StorageClient, grpcCallName string, err error, errCode codes.Code) {

	var msg, eventReason, eventType string

	if grpcCallName == OnboardConsumer {
		if errCode == codes.InvalidArgument {
			msg = "Token is invalid. Verify the token again or contact the provider admin"
			eventReason = "TokenInvalid"
			eventType = corev1.EventTypeWarning
		} else if errCode == codes.AlreadyExists {
			msg = "Token is already used. Contact provider admin for a new token"
			eventReason = "TokenAlreadyUsed"
			eventType = corev1.EventTypeWarning
		}
	} else if grpcCallName == AcknowledgeOnboarding {
		if errCode == codes.NotFound {
			msg = "StorageConsumer not found. Contact the provider admin"
			eventReason = "NotFound"
			eventType = corev1.EventTypeWarning
		}
	} else if grpcCallName == OffboardConsumer {
		if errCode == codes.InvalidArgument {
			msg = "StorageConsumer UID is not valid. Contact the provider admin"
			eventReason = "UIDInvalid"
			eventType = corev1.EventTypeWarning
		}
	} else if grpcCallName == GetStorageConfig {
		if errCode == codes.InvalidArgument {
			msg = "StorageConsumer UID is not valid. Contact the provider admin"
			eventReason = "UIDInvalid"
			eventType = corev1.EventTypeWarning
		} else if errCode == codes.NotFound {
			msg = "StorageConsumer UID not found. Contact the provider admin"
			eventReason = "UIDNotFound"
			eventType = corev1.EventTypeWarning
		} else if errCode == codes.Unavailable {
			msg = "StorageConsumer is not ready yet. Will requeue after 5 second"
			eventReason = "NotReady"
			eventType = corev1.EventTypeNormal
		}
	}

	if msg != "" {
		s.Log.Error(err, "StorageProvider:"+grpcCallName+":"+msg)
		s.recorder.ReportIfNotPresent(instance, eventType, eventReason, msg)
	}
}

func getStatusReporterName(namespace, name string) string {
	// getStatusReporterName generates a name for a StatusReporter CronJob.
	var s struct {
		StorageClientName      string `json:"storageClientName"`
		StorageClientNamespace string `json:"storageClientNamespace"`
	}
	s.StorageClientName = name
	s.StorageClientNamespace = namespace

	statusReporterName, err := json.Marshal(s)
	if err != nil {
		klog.Errorf("failed to marshal a name for a storage client based on %v. %v", s, err)
		panic("failed to marshal storage client name")
	}
	reporterName := md5.Sum([]byte(statusReporterName))
	// The name of the StorageClient is the MD5 hash of the JSON
	// representation of the StorageClient name and namespace.
	return fmt.Sprintf("storageclient-%s-status-reporter", hex.EncodeToString(reporterName[:8]))
}

func (s *StorageClientReconciler) delete(obj client.Object) error {
	if err := s.Client.Delete(s.ctx, obj); err != nil && !kerrors.IsNotFound(err) {
		return err
	}
	return nil
}

func (s *StorageClientReconciler) reconcileClientStatusReporterJob(instance *v1alpha1.StorageClient) (reconcile.Result, error) {
	// start the cronJob to ping the provider api server
	cronJob := &batchv1.CronJob{}
	cronJob.Name = getStatusReporterName(instance.Namespace, instance.Name)
	cronJob.Namespace = s.OperatorNamespace
	utils.AddLabel(cronJob, storageClientNameLabel, instance.Name)
	utils.AddLabel(cronJob, storageClientNamespaceLabel, instance.Namespace)
	var podDeadLineSeconds int64 = 120
	jobDeadLineSeconds := podDeadLineSeconds + 35
	var keepJobResourceSeconds int32 = 600
	var reducedKeptSuccecsful int32 = 1

	_, err := controllerutil.CreateOrUpdate(s.ctx, s.Client, cronJob, func() error {
		cronJob.Spec = batchv1.CronJobSpec{
			Schedule:                   "* * * * *",
			ConcurrencyPolicy:          batchv1.ForbidConcurrent,
			SuccessfulJobsHistoryLimit: &reducedKeptSuccecsful,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					ActiveDeadlineSeconds:   &jobDeadLineSeconds,
					TTLSecondsAfterFinished: &keepJobResourceSeconds,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							ActiveDeadlineSeconds: &podDeadLineSeconds,
							Containers: []corev1.Container{
								{
									Name:  "heartbeat",
									Image: os.Getenv(utils.StatusReporterImageEnvVar),
									Command: []string{
										"/status-reporter",
									},
									Env: []corev1.EnvVar{
										{
											Name:  utils.StorageClientNamespaceEnvVar,
											Value: instance.Namespace,
										},
										{
											Name:  utils.StorageClientNameEnvVar,
											Value: instance.Name,
										},
										{
											Name:  utils.OperatorNamespaceEnvVar,
											Value: s.OperatorNamespace,
										},
									},
								},
							},
							RestartPolicy:      corev1.RestartPolicyOnFailure,
							ServiceAccountName: "ocs-client-operator-status-reporter",
						},
					},
				},
			},
		}
		return nil
	})
	if err != nil {
		return reconcile.Result{Requeue: true}, fmt.Errorf("Failed to update cronJob: %v", err)
	}
	return reconcile.Result{}, nil
}

func (s *StorageClientReconciler) list(obj client.ObjectList, listOptions ...client.ListOption) error {
	return s.Client.List(s.ctx, obj, listOptions...)
}

func (s *StorageClientReconciler) reconcileDefaultStorageClaims(instance *v1alpha1.StorageClient) error {

	if instance.GetAnnotations()[defaultClaimsProcessedAnnotationKey] == "true" {
		// we already processed default claims for this client
		return nil
	}

	// try to list the default client who is the default storage claims owner
	claimOwners := &v1alpha1.StorageClientList{}
	if err := s.list(claimOwners, client.MatchingFields{defaultClaimsOwnerIndexName: "true"}); err != nil {
		return fmt.Errorf("failed to list default storage claims owner: %v", err)
	}

	if len(claimOwners.Items) == 0 {
		// no other storageclient claims as an owner and take responsibility of creating the default claims by becoming owner
		if utils.AddAnnotation(instance, defaultClaimsOwnerAnnotationKey, "true") {
			if err := s.update(instance); err != nil {
				return fmt.Errorf("not able to claim ownership of creating default storageclaims: %v", err)
			}
		}
	}

	// we successfully took the ownership to create a default claim from this storageclient, so create default claims if not created
	// after claiming as an owner no other storageclient will try to take ownership for creating default storageclaims
	annotations := instance.GetAnnotations()
	if annotations[defaultClaimsOwnerAnnotationKey] == "true" && annotations[defaultClaimsProcessedAnnotationKey] != "true" {
		if err := s.createDefaultBlockStorageClaim(instance); err != nil && !kerrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create %q storageclaim: %v", defaultBlockStorageClaim, err)
		}
		if err := s.createDefaultSharedfileStorageClaim(instance); err != nil && !kerrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create %q storageclaim: %v", defaultSharedfileStorageClaim, err)
		}
	}

	// annotate that we created default storageclaims successfully and will not retry
	if utils.AddAnnotation(instance, defaultClaimsProcessedAnnotationKey, "true") {
		if err := s.update(instance); err != nil {
			return fmt.Errorf("not able to update annotation for creation of default storageclaims: %v", err)
		}
	}

	return nil
}

func (s *StorageClientReconciler) createDefaultBlockStorageClaim(instance *v1alpha1.StorageClient) error {
	storageclaim := &v1alpha1.StorageClaim{}
	storageclaim.Name = defaultBlockStorageClaim
	storageclaim.Spec.Type = "block"
	storageclaim.Spec.StorageClient = &v1alpha1.StorageClientNamespacedName{
		Name:      instance.Name,
		Namespace: instance.Namespace,
	}
	return s.create(storageclaim)
}

func (s *StorageClientReconciler) createDefaultSharedfileStorageClaim(instance *v1alpha1.StorageClient) error {
	sharedfileClaim := &v1alpha1.StorageClaim{}
	sharedfileClaim.Name = defaultSharedfileStorageClaim
	sharedfileClaim.Spec.Type = "sharedfile"
	sharedfileClaim.Spec.StorageClient = &v1alpha1.StorageClientNamespacedName{
		Name:      instance.Name,
		Namespace: instance.Namespace,
	}
	return s.create(sharedfileClaim)
}

func (s *StorageClientReconciler) deleteDefaultStorageClaims(instance *v1alpha1.StorageClient) error {
	if instance.GetAnnotations()[defaultClaimsOwnerAnnotationKey] == "true" {
		blockClaim := &v1alpha1.StorageClaim{}
		blockClaim.Name = defaultBlockStorageClaim
		if err := s.delete(blockClaim); err != nil {
			return fmt.Errorf("failed to remove default storageclaim %q: %v", blockClaim.Name, err)
		}

		sharedfsClaim := &v1alpha1.StorageClaim{}
		sharedfsClaim.Name = defaultSharedfileStorageClaim
		if err := s.delete(sharedfsClaim); err != nil {
			return fmt.Errorf("failed to remove default storageclaim %q: %v", blockClaim.Name, err)
		}
		s.Log.Info("Successfully deleted default storageclaims")
	}
	return nil
}

func (s *StorageClientReconciler) update(obj client.Object, opts ...client.UpdateOption) error {
	return s.Update(s.ctx, obj, opts...)
}

func (s *StorageClientReconciler) create(obj client.Object, opts ...client.CreateOption) error {
	return s.Create(s.ctx, obj, opts...)
}
