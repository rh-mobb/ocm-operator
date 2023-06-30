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
	"sigs.k8s.io/controller-runtime/pkg/log"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/controllers"
	"github.com/rh-mobb/ocm-operator/pkg/aws"
	"github.com/rh-mobb/ocm-operator/pkg/conditions"
	"github.com/rh-mobb/ocm-operator/pkg/events"
	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
	"github.com/rh-mobb/ocm-operator/pkg/triggers"
	"github.com/rh-mobb/ocm-operator/pkg/workload"
)

// ROSAClusterRequest is an object that is unique to each reconciliation
// request.
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

func (r *Controller) NewRequest(ctx context.Context, req ctrl.Request) (controllers.Request, error) {
	original := &ocmv1alpha1.ROSACluster{}

	// get the object (desired state) from the cluster
	if err := r.Get(ctx, req.NamespacedName, original); err != nil {
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
	request := &ROSAClusterRequest{
		Original:          original,
		Desired:           desired,
		ControllerRequest: req,
		Context:           ctx,
		Log:               log.Log,
		Trigger:           triggers.GetTrigger(original),
		Reconciler:        r,
	}

	// set the version
	if err := request.setVersion(); err != nil {
		return request, fmt.Errorf("unable to determine openshift version - %w", err)
	}

	return request, nil
}

// GetObject returns the original object to satisfy the controllers.Request interface.
func (request *ROSAClusterRequest) GetObject() workload.Workload {
	return request.Original
}

// GetName returns the name as it should appear in OCM.
func (request *ROSAClusterRequest) GetName() string {
	return request.Desired.Spec.DisplayName
}

// desired returns whether or not the request is in its current desired state.
func (request *ROSAClusterRequest) desired() bool {
	if request.Desired == nil || request.Current == nil {
		return false
	}

	// ignore the account id as it does not show up in the api request
	request.Current.Spec.AccountID = request.Desired.Spec.AccountID

	// ignore the tags as there are red hat managed tags that get added
	// that are not a part of the spec.  only compare the tags that are
	// in our desired spec.
	for desiredKey, desiredValue := range request.Desired.Spec.Tags {
		if request.Current.Spec.Tags[desiredKey] != desiredValue {
			return false
		}
	}
	request.Current.Spec.Tags = request.Desired.Spec.Tags

	return reflect.DeepEqual(
		request.Desired.Spec,
		request.Current.Spec,
	)
}

// setVersion sets the desired requested OpenShift version for the request.  If
// a version is requested in the spec, it is validated against a list of versions
// from the OCM API.  If one is not requested in the spec, the latest available
// version is automatically selected.
func (request *ROSAClusterRequest) setVersion() (err error) {
	// ensure the desired version is set on the spec
	if request.Desired.Spec.OpenShiftVersion == "" {
		// get the default version if we have not stored a valid version
		// in the status.
		if request.Desired.Status.OpenShiftVersion == "" {
			version, err := ocm.GetDefaultVersion(request.Reconciler.Connection)
			if err != nil {
				return fmt.Errorf("unable to retrieve default version - %w", err)
			}

			request.Desired.Spec.OpenShiftVersion = version.RawID()
			request.Version = version
		} else {
			// get the version from the status.  at this point we know the version has
			// been validated if it is stored on the status.
			request.Desired.Spec.OpenShiftVersion = request.Desired.Status.OpenShiftVersion
		}
	}

	// get the version object from our desired version
	if request.Version == nil {
		version, err := ocm.GetVersionObject(request.Reconciler.Connection, request.Desired.Spec.OpenShiftVersion)
		if err != nil {
			return fmt.Errorf(
				"found invalid version [%s] - %w",
				request.Desired.Spec.OpenShiftVersion,
				err,
			)
		}

		request.Version = version
	}

	// set the id needed for the api call in the status.
	if request.Original.Status.OpenShiftVersionID == "" {
		// update the status to include the proper version id and the desired version id
		original := request.Original.DeepCopy()
		request.Original.Status.OpenShiftVersion = request.Desired.Spec.OpenShiftVersion
		request.Original.Status.OpenShiftVersionID = request.Version.ID()
		if err := kubernetes.PatchStatus(request.Context, request.Reconciler, original, request.Original); err != nil {
			return fmt.Errorf("unable to update status operatorRolesCreated=true - %w", err)
		}
	}

	return nil
}

// createCluster performs all operations necessary for creating a ROSA cluster.
func (request *ROSAClusterRequest) createCluster() error {
	original := request.Original.DeepCopy()

	// create oidc provider and config
	oidc, err := request.ensureOIDCProvider()
	if err != nil {
		return err
	}

	// create the operator roles
	if !request.Original.Status.OperatorRolesCreated {
		request.Log.Info("creating operator roles", controllers.LogValues(request)...)
		if createErr := request.createOperatorRoles(oidc); createErr != nil {
			return createErr
		}
	}

	// get the availability zones if we provided subnets
	var availabilityZones []string
	if request.Desired.HasSubnets() {
		availabilityZones, err = request.Reconciler.AWSClient.GetAvailabilityZonesBySubnet(request.Desired.Spec.Network.Subnets)
		if err != nil {
			return fmt.Errorf("unable to retrieve availability zones from provided subnets - %w", err)
		}
	}

	// create the cluster
	request.Log.Info("creating rosa cluster", controllers.LogValues(request)...)
	cluster, err := request.OCMClient.Create(request.Desired.Builder(
		oidc,
		request.Original.Status.OpenShiftVersionID,
		availabilityZones,
	))
	if err != nil {
		return fmt.Errorf("unable to create rosa cluster in ocm - %w", err)
	}

	// store the required provider data in the status
	request.Original.Status.ClusterID = cluster.ID()
	request.Cluster = cluster

	if err := kubernetes.PatchStatus(request.Context, request.Reconciler, original, request.Original); err != nil {
		return fmt.Errorf("unable to update status providerID=%s - %w", cluster.ID(), err)
	}

	return nil
}

// updateCluster performs all necessary actions for updating a ROSA cluster.
func (request *ROSAClusterRequest) updateCluster() error {
	// retrieve oidc config
	oidc, err := ocm.NewOIDCConfigClient(request.Reconciler.Connection).Get(request.Original.Status.OIDCConfigID)
	if err != nil {
		return fmt.Errorf("unable to get oidc config from ocm - %w", err)
	}

	// get the availability zones if we provided subnets
	var availabilityZones []string
	if request.Desired.HasSubnets() {
		availabilityZones, err = request.Reconciler.AWSClient.GetAvailabilityZonesBySubnet(request.Desired.Spec.Network.Subnets)
		if err != nil {
			return fmt.Errorf("unable to retrieve availability zones from provided subnets - %w", err)
		}
	}

	// update the rosa cluster if it does exist
	request.Log.Info("updating rosa cluster", controllers.LogValues(request)...)
	cluster, err := request.OCMClient.Update(request.Desired.Builder(
		oidc,
		request.Original.Status.OpenShiftVersionID,
		availabilityZones,
	).ID(request.Original.Status.ClusterID),
	)
	if err != nil {
		return fmt.Errorf("unable to update rosa cluster in ocm - %w", err)
	}

	request.Cluster = cluster

	return nil
}

// ensureOIDCProvider creates the OIDC Provider in AWS.
func (request *ROSAClusterRequest) ensureOIDCProvider() (config *clustersmgmtv1.OidcConfig, err error) {
	original := request.Original.DeepCopy()

	// create oidc config only if we have not created it already
	if request.Original.Status.OIDCConfigID == "" {
		request.Log.Info("creating oidc config", controllers.LogValues(request)...)
		config, err = ocm.NewOIDCConfigClient(request.Reconciler.Connection).Create()
		if err != nil {
			return config, fmt.Errorf("unable to create oidc config - %w", err)
		}

		// update the status with the oidc config id
		request.Original.Status.OIDCConfigID = config.ID()
		if err = kubernetes.PatchStatus(request.Context, request.Reconciler, original, request.Original); err != nil {
			return config, fmt.Errorf("unable to update status oidcConfigID=%s - %w", config.ID(), err)
		}
	} else {
		// get the oidc config
		config, err = ocm.NewOIDCConfigClient(request.Reconciler.Connection).Get(request.Original.Status.OIDCConfigID)
		if err != nil {
			return config, fmt.Errorf("unable to get oidc config [%s] - %w", request.Original.Status.OIDCConfigID, err)
		}
	}

	// create the oidc provider if we have not created it already
	if request.Original.Status.OIDCProviderARN == "" {
		request.Log.Info("creating oidc provider", controllers.LogValues(request)...)
		providerARN, err := request.Reconciler.AWSClient.CreateOIDCProvider(config.IssuerUrl())
		if err != nil {
			return config, fmt.Errorf("unable to create oidc provider - %w", err)
		}

		// update the status with the oidc provider arn
		request.Original.Status.OIDCProviderARN = providerARN
		if err := kubernetes.PatchStatus(request.Context, request.Reconciler, original, request.Original); err != nil {
			return config, fmt.Errorf("unable to update status oidcProviderARN=%s - %w", providerARN, err)
		}
	}

	return config, nil
}

// createOperatorRoles creates the operator roles in AWS.
func (request *ROSAClusterRequest) createOperatorRoles(oidc *clustersmgmtv1.OidcConfig) error {
	// create the sts client
	stsClient := ocm.NewSTSClient(
		request.Reconciler.Connection,
		request.Desired.Spec.HostedControlPlane,
		request.Desired.Spec.IAM.EnableManagedPolicies,
		request.Desired.Spec.IAM.OperatorRolesPrefix,
		request.Desired.Spec.AccountID,
		oidc.IssuerUrl(),
	)

	// retrieve the credential requests
	requests, err := stsClient.GetCredentialRequests()
	if err != nil {
		return fmt.Errorf("unable to retrieve sts credential requests - %w", err)
	}

	// create the operator roles
	if err := stsClient.CreateOperatorRoles(request.Reconciler.AWSClient, request.Version, requests...); err != nil {
		return fmt.Errorf("unable to create operator roles - %w", err)
	}

	// update the status indicating the operator roles have been created.  additionally
	// update the status for the openshift id needed on both the desired and original
	// objects.
	original := request.Original.DeepCopy()
	request.Original.Status.OperatorRolesCreated = true
	if err := kubernetes.PatchStatus(request.Context, request.Reconciler, original, request.Original); err != nil {
		return fmt.Errorf("unable to update status operatorRolesCreated=true - %w", err)
	}

	return nil
}

// destroyOperatorRoles deletes the operator roles in AWS.
func (request *ROSAClusterRequest) destroyOperatorRoles() error {
	// create the sts client
	stsClient := ocm.NewSTSClient(
		request.Reconciler.Connection,
		request.Desired.Spec.HostedControlPlane,
		request.Desired.Spec.IAM.EnableManagedPolicies,
		request.Desired.Spec.IAM.OperatorRolesPrefix,
		request.Desired.Spec.AccountID,
		"",
	)

	// retrieve the credential requests
	requests, err := stsClient.GetCredentialRequests()
	if err != nil {
		return fmt.Errorf("unable to retrieve sts credential requests - %w", err)
	}

	// delete the operator roles
	if err := stsClient.DeleteOperatorRoles(request.Reconciler.AWSClient, requests...); err != nil {
		return fmt.Errorf("unable to delete operator roles - %w", err)
	}

	return nil
}

// notify notifies the user via a condition update and an event creation that something has happened.
func (request *ROSAClusterRequest) notify(event events.Event, condition *metav1.Condition, name string) error {
	// create an event registered to the resource notifying the consumer that something important
	// has happened
	events.RegisterAction(
		event,
		request.Original,
		request.Reconciler.Recorder,
		name,
		request.Original.Status.ClusterID,
	)

	// update the status with the condition
	return conditions.Update(request.Context, request.Reconciler, request.Original, condition)
}

// provisionRequeueTime determines the requeue time when the cluster prior to the cluster being ready.
func (request *ROSAClusterRequest) provisionRequeueTime() time.Duration {
	// change the requeue time based on whether we have a hosted control plane or
	// not. this is because hosted control plane comes up much faster and should
	// be reconciled more frequently.
	if request.Desired.Spec.HostedControlPlane {
		return defaultClusterRequeueHostedPostProvision
	}

	return defaultClusterRequeueClassicPostProvision
}
