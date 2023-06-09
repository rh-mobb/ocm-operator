package rosacluster

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/controllers/conditions"
	"github.com/rh-mobb/ocm-operator/controllers/events"
	"github.com/rh-mobb/ocm-operator/controllers/request"
	"github.com/rh-mobb/ocm-operator/controllers/triggers"
	"github.com/rh-mobb/ocm-operator/controllers/workload"
	"github.com/rh-mobb/ocm-operator/pkg/aws"
	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
)

// ROSAClusterRequest is an object that is unique to each reconciliation
// req.
type ROSAClusterRequest struct {
	Context           context.Context
	ControllerRequest ctrl.Request
	Current           *ocmv1alpha1.ROSACluster
	Original          *ocmv1alpha1.ROSACluster
	Desired           *ocmv1alpha1.ROSACluster
	Log               logr.Logger
	Trigger           triggers.Trigger
	Reconciler        *Controller
	OCMClient         *ocm.ClusterClient

	// data obtained during request reconciliation
	Cluster *clustersmgmtv1.Cluster
	Version *clustersmgmtv1.Version
}

func (r *Controller) NewRequest(ctx context.Context, ctrlReq ctrl.Request) (request.Request, error) {
	original := &ocmv1alpha1.ROSACluster{}

	// get the object (desired state) from the cluster
	if err := r.Get(ctx, ctrlReq.NamespacedName, original); err != nil {
		if !apierrs.IsNotFound(err) {
			return &ROSAClusterRequest{}, fmt.Errorf("unable to fetch cluster object - %w", err)
		}

		return &ROSAClusterRequest{}, err
	}

	// create the desired state of the request based on the inputs
	desired := original.DeepCopy()
	if desired.Spec.DisplayName == "" {
		desired.Spec.DisplayName = desired.Name
	}

	// set the prefix to the cluster name with a random id if it is unset.  additionally
	// store the prefix in the status so that the user knows what their prefix was
	// which is important if the prefix was auto-generated.
	if original.Status.OperatorRolesPrefix != "" {
		desired.Spec.IAM.OperatorRolesPrefix = original.Status.OperatorRolesPrefix
	} else {
		if desired.Spec.IAM.OperatorRolesPrefix == "" {
			desired.Spec.IAM.OperatorRolesPrefix = aws.GetOperatorRolesPrefixForCluster(desired.Spec.DisplayName)
		}

		patched := original.DeepCopy()

		original.Status.OperatorRolesPrefix = desired.Spec.IAM.OperatorRolesPrefix

		if err := kubernetes.PatchStatus(ctx, r, patched, original); err != nil {
			return &ROSAClusterRequest{}, fmt.Errorf("unable to update status operatorRolesCreated=true - %w", err)
		}
	}

	// set the network config defaults if subnets are not provided
	// NOTE: this may not be required.  Found that the values for the
	// network were not being deserialized even with defaults in the CRD
	// set.  This is to ensure that when subnets are left out, that
	// defaults get set if they are not set.
	if !desired.HasSubnets() {
		desired.SetNetworkDefaults()
	}

	// create the request
	req := &ROSAClusterRequest{
		Original:          original,
		Desired:           desired,
		ControllerRequest: ctrlReq,
		Context:           ctx,
		Log:               r.Logger,
		Trigger:           triggers.GetTrigger(original),
		Reconciler:        r,
	}

	// set the version
	if err := req.setVersion(); err != nil {
		return req, fmt.Errorf("unable to determine openshift version - %w", err)
	}

	return req, nil
}

// DefaultRequeue returns the default requeue time for a request.
func (req *ROSAClusterRequest) DefaultRequeue() time.Duration {
	return defaultClusterRequeue
}

// GetObject returns the original object to satisfy the request.Request interface.
func (req *ROSAClusterRequest) GetObject() workload.Workload {
	return req.Original
}

// GetName returns the name as it should appear in OCM.
func (req *ROSAClusterRequest) GetName() string {
	return req.Desired.Spec.DisplayName
}

// GetContext returns the context of the request.
func (req *ROSAClusterRequest) GetContext() context.Context {
	return req.Context
}

// GetReconciler returns the context of the request.
func (req *ROSAClusterRequest) GetReconciler() kubernetes.Client {
	return req.Reconciler
}

// desired returns whether or not the request is in its current desired state.
func (req *ROSAClusterRequest) desired() bool {
	if req.Desired == nil || req.Current == nil {
		return false
	}

	// ignore the account id as it does not show up in the api request
	req.Current.Spec.AccountID = req.Desired.Spec.AccountID

	// ignore the tags as there are red hat managed tags that get added
	// that are not a part of the spec.  only compare the tags that are
	// in our desired spec.
	for desiredKey, desiredValue := range req.Desired.Spec.Tags {
		if req.Current.Spec.Tags[desiredKey] != desiredValue {
			return false
		}
	}
	req.Current.Spec.Tags = req.Desired.Spec.Tags

	return reflect.DeepEqual(
		req.Desired.Spec,
		req.Current.Spec,
	)
}

// setVersion sets the desired requested OpenShift version for the req.  If
// a version is requested in the spec, it is validated against a list of versions
// from the OCM API.  If one is not requested in the spec, the latest available
// version is automatically selected.
func (req *ROSAClusterRequest) setVersion() (err error) {
	// ensure the desired version is set on the spec
	if req.Desired.Spec.OpenShiftVersion == "" {
		// get the default version if we have not stored a valid version
		// in the status.
		if req.Desired.Status.OpenShiftVersion == "" {
			version, err := ocm.GetDefaultVersion(req.Reconciler.Connection)
			if err != nil {
				return fmt.Errorf("unable to retrieve default version - %w", err)
			}

			req.Desired.Spec.OpenShiftVersion = version.RawID()
			req.Version = version
		} else {
			// get the version from the status.  at this point we know the version has
			// been validated if it is stored on the status.
			req.Desired.Spec.OpenShiftVersion = req.Desired.Status.OpenShiftVersion
		}
	}

	// get the version object from our desired version
	if req.Version == nil {
		version, err := ocm.GetVersionObject(req.Reconciler.Connection, req.Desired.Spec.OpenShiftVersion)
		if err != nil {
			return fmt.Errorf(
				"found invalid version [%s] - %w",
				req.Desired.Spec.OpenShiftVersion,
				err,
			)
		}

		req.Version = version
	}

	// set the id needed for the api call in the status.
	if req.Original.Status.OpenShiftVersionID == "" {
		// update the status to include the proper version id and the desired version id
		original := req.Original.DeepCopy()
		req.Original.Status.OpenShiftVersion = req.Desired.Spec.OpenShiftVersion
		req.Original.Status.OpenShiftVersionID = req.Version.ID()
		if err := kubernetes.PatchStatus(req.Context, req.Reconciler, original, req.Original); err != nil {
			return fmt.Errorf("unable to update status operatorRolesCreated=true - %w", err)
		}
	}

	return nil
}

// createCluster performs all operations necessary for creating a ROSA cluster.
func (req *ROSAClusterRequest) createCluster() error {
	original := req.Original.DeepCopy()

	// create oidc provider and config
	oidc, err := req.ensureOIDCProvider()
	if err != nil {
		return err
	}

	// create the operator roles
	if !req.Original.Status.OperatorRolesCreated {
		req.Log.Info("creating operator roles", request.LogValues(req)...)
		if createErr := req.createOperatorRoles(oidc); createErr != nil {
			return createErr
		}
	}

	// get the availability zones if we provided subnets
	var availabilityZones []string
	if req.Desired.HasSubnets() {
		availabilityZones, err = req.Reconciler.AWSClient.GetAvailabilityZonesBySubnet(req.Desired.Spec.Network.Subnets)
		if err != nil {
			return fmt.Errorf("unable to retrieve availability zones from provided subnets - %w", err)
		}
	}

	// create the cluster
	req.Log.Info("creating rosa cluster", request.LogValues(req)...)
	cluster, err := req.OCMClient.Create(req.Desired.Builder(
		oidc,
		req.Original.Status.OpenShiftVersionID,
		availabilityZones,
	))
	if err != nil {
		return fmt.Errorf("unable to create rosa cluster in ocm - %w", err)
	}

	// store the required provider data in the status
	req.Original.Status.ClusterID = cluster.ID()
	req.Cluster = cluster

	if err := kubernetes.PatchStatus(req.Context, req.Reconciler, original, req.Original); err != nil {
		return fmt.Errorf("unable to update status providerID=%s - %w", cluster.ID(), err)
	}

	return nil
}

// updateCluster performs all necessary actions for updating a ROSA cluster.
func (req *ROSAClusterRequest) updateCluster() error {
	// retrieve oidc config
	oidc, err := ocm.NewOIDCConfigClient(req.Reconciler.Connection).Get(req.Original.Status.OIDCConfigID)
	if err != nil {
		return fmt.Errorf("unable to get oidc config from ocm - %w", err)
	}

	// get the availability zones if we provided subnets
	var availabilityZones []string
	if req.Desired.HasSubnets() {
		availabilityZones, err = req.Reconciler.AWSClient.GetAvailabilityZonesBySubnet(req.Desired.Spec.Network.Subnets)
		if err != nil {
			return fmt.Errorf("unable to retrieve availability zones from provided subnets - %w", err)
		}
	}

	// update the rosa cluster if it does exist
	req.Log.Info("updating rosa cluster", request.LogValues(req)...)
	cluster, err := req.OCMClient.Update(req.Desired.Builder(
		oidc,
		req.Original.Status.OpenShiftVersionID,
		availabilityZones,
	).ID(req.Original.Status.ClusterID),
	)
	if err != nil {
		return fmt.Errorf("unable to update rosa cluster in ocm - %w", err)
	}

	req.Cluster = cluster

	return nil
}

// ensureOIDCProvider creates the OIDC Provider in AWS.
func (req *ROSAClusterRequest) ensureOIDCProvider() (config *clustersmgmtv1.OidcConfig, err error) {
	original := req.Original.DeepCopy()

	// create oidc config only if we have not created it already
	if req.Original.Status.OIDCConfigID == "" {
		req.Log.Info("creating oidc config", request.LogValues(req)...)
		config, err = ocm.NewOIDCConfigClient(req.Reconciler.Connection).Create()
		if err != nil {
			return config, fmt.Errorf("unable to create oidc config - %w", err)
		}

		// update the status with the oidc config id
		req.Original.Status.OIDCConfigID = config.ID()
		if err = kubernetes.PatchStatus(req.Context, req.Reconciler, original, req.Original); err != nil {
			return config, fmt.Errorf("unable to update status oidcConfigID=%s - %w", config.ID(), err)
		}
	} else {
		// get the oidc config
		config, err = ocm.NewOIDCConfigClient(req.Reconciler.Connection).Get(req.Original.Status.OIDCConfigID)
		if err != nil {
			return config, fmt.Errorf("unable to get oidc config [%s] - %w", req.Original.Status.OIDCConfigID, err)
		}
	}

	// create the oidc provider if we have not created it already
	if req.Original.Status.OIDCProviderARN == "" {
		req.Log.Info("creating oidc provider", request.LogValues(req)...)
		providerARN, err := req.Reconciler.AWSClient.CreateOIDCProvider(config.IssuerUrl())
		if err != nil {
			return config, fmt.Errorf("unable to create oidc provider - %w", err)
		}

		// update the status with the oidc provider arn
		req.Original.Status.OIDCProviderARN = providerARN
		if err := kubernetes.PatchStatus(req.Context, req.Reconciler, original, req.Original); err != nil {
			return config, fmt.Errorf("unable to update status oidcProviderARN=%s - %w", providerARN, err)
		}
	}

	return config, nil
}

// createOperatorRoles creates the operator roles in AWS.
func (req *ROSAClusterRequest) createOperatorRoles(oidc *clustersmgmtv1.OidcConfig) error {
	// create the sts client
	stsClient := ocm.NewSTSClient(
		req.Reconciler.Connection,
		req.Desired.Spec.HostedControlPlane,
		req.Desired.Spec.IAM.EnableManagedPolicies,
		req.Desired.Spec.IAM.OperatorRolesPrefix,
		req.Desired.Spec.AccountID,
		oidc.IssuerUrl(),
	)

	// retrieve the credential requests
	requests, err := stsClient.GetCredentialRequests()
	if err != nil {
		return fmt.Errorf("unable to retrieve sts credential requests - %w", err)
	}

	// create the operator roles
	if err := stsClient.CreateOperatorRoles(req.Reconciler.AWSClient, req.Version, requests...); err != nil {
		return fmt.Errorf("unable to create operator roles - %w", err)
	}

	// update the status indicating the operator roles have been created.  additionally
	// update the status for the openshift id needed on both the desired and original
	// objects.
	original := req.Original.DeepCopy()
	req.Original.Status.OperatorRolesCreated = true
	if err := kubernetes.PatchStatus(req.Context, req.Reconciler, original, req.Original); err != nil {
		return fmt.Errorf("unable to update status operatorRolesCreated=true - %w", err)
	}

	return nil
}

// destroyOperatorRoles deletes the operator roles in AWS.
func (req *ROSAClusterRequest) destroyOperatorRoles() error {
	// create the sts client
	stsClient := ocm.NewSTSClient(
		req.Reconciler.Connection,
		req.Desired.Spec.HostedControlPlane,
		req.Desired.Spec.IAM.EnableManagedPolicies,
		req.Desired.Spec.IAM.OperatorRolesPrefix,
		req.Desired.Spec.AccountID,
		"",
	)

	// retrieve the credential requests
	requests, err := stsClient.GetCredentialRequests()
	if err != nil {
		return fmt.Errorf("unable to retrieve sts credential requests - %w", err)
	}

	// delete the operator roles
	if err := stsClient.DeleteOperatorRoles(req.Reconciler.AWSClient, requests...); err != nil {
		return fmt.Errorf("unable to delete operator roles - %w", err)
	}

	return nil
}

// notify notifies the user via a condition update and an event creation that something has happened.
func (req *ROSAClusterRequest) notify(event events.Event, condition *metav1.Condition, name string) error {
	// create an event registered to the resource notifying the consumer that something important
	// has happened
	events.RegisterAction(
		event,
		req.Original,
		req.Reconciler.Recorder,
		name,
		req.Original.Status.ClusterID,
	)

	// update the status with the condition
	return conditions.Update(req, condition)
}

// provisionRequeueTime determines the requeue time when the cluster prior to the cluster being ready.
func (req *ROSAClusterRequest) provisionRequeueTime() time.Duration {
	// change the requeue time based on whether we have a hosted control plane or
	// not. this is because hosted control plane comes up much faster and should
	// be reconciled more frequently.
	if req.Desired.Spec.HostedControlPlane {
		return defaultClusterRequeueHostedPostProvision
	}

	return defaultClusterRequeueClassicPostProvision
}
