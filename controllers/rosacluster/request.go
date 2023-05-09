package rosacluster

import (
	"context"
	"fmt"
	"reflect"

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
}

func (r *Controller) NewRequest(ctx context.Context, req ctrl.Request) (controllers.Request, error) {
	original := &ocmv1alpha1.ROSACluster{}

	// get the object (desired state) from the cluster
	//nolint:wrapcheck
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

	// set the prefix to the cluster name with a random id if it is unset
	if desired.Spec.IAM.OperatorRolesPrefix == "" {
		desired.Spec.IAM.OperatorRolesPrefix = aws.GetOperatorRolesPrefixForCluster(desired.Spec.DisplayName)
	}

	return &ROSAClusterRequest{
		Original:          original,
		Desired:           desired,
		ControllerRequest: req,
		Context:           ctx,
		Log:               log.Log,
		Trigger:           triggers.GetTrigger(original),
		Reconciler:        r,
	}, nil
}

func (request *ROSAClusterRequest) GetObject() controllers.Workload {
	return request.Original
}

// execute executes a variety of different phases for the request.
//
//nolint:wrapcheck
func (request *ROSAClusterRequest) execute(phases ...Phase) (ctrl.Result, error) {
	for execute := range phases {
		// run each phase function and return if we receive any errors
		result, err := phases[execute].Function(request)
		if err != nil || result.Requeue {
			return result, controllers.ReconcileError(
				request.ControllerRequest,
				fmt.Sprintf("%s phase reconciliation error", phases[execute].Name),
				err,
			)
		}
	}

	return controllers.NoRequeue(), nil
}

// TODO: centralize this function into controllers or conditions package.
func (request *ROSAClusterRequest) updateCondition(condition *metav1.Condition) error {
	if err := conditions.Update(
		request.Context,
		request.Reconciler,
		request.Original,
		condition,
	); err != nil {
		return fmt.Errorf("unable to update condition - %w", err)
	}

	return nil
}

// logValues produces a consistent set of log values for this request.
func (request *ROSAClusterRequest) logValues() []interface{} {
	return []interface{}{
		"resource", fmt.Sprintf("%s/%s", request.Desired.Namespace, request.Desired.Name),
		"name", request.Desired.Spec.DisplayName,
	}
}

// desired returns whether or not the request is in its current desired state.
func (request *ROSAClusterRequest) desired() bool {
	if request.Desired == nil || request.Current == nil {
		return false
	}

	// ignore the account id as it does not show up in the api request
	request.Current.Spec.AccountID = request.Desired.Spec.AccountID

	return reflect.DeepEqual(
		request.Desired.Spec,
		request.Current.Spec,
	)
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
	request.Log.Info("creating operator roles", request.logValues()...)
	if err := request.ensureOperatorRoles(oidc); err != nil {
		return err
	}

	// create the cluster
	request.Log.Info("creating rosa cluster", request.logValues()...)
	cluster, err := request.OCMClient.Create(request.Desired.Builder(oidc))
	if err != nil {
		return fmt.Errorf("unable to create rosa cluster in ocm - %w", err)
	}

	// store the required provider data in the status
	request.Original.Status.ClusterID = cluster.ID()

	if err := kubernetes.PatchStatus(request.Context, request.Reconciler, original, request.Original); err != nil {
		return fmt.Errorf("unable to update status providerID=%s - %w", cluster.ID(), err)
	}

	// create an event indicating that the rosa cluster has been created
	events.RegisterAction(
		events.Created,
		request.Original,
		request.Reconciler.Recorder,
		request.Desired.Spec.DisplayName,
		request.Original.Status.ClusterID,
	)

	return nil
}

// updateCluster performs all necessary actions for updating a ROSA cluster.
func (request *ROSAClusterRequest) updateCluster() error {
	// retrieve oidc config
	oidc, err := ocm.NewOIDCConfigClient(request.Reconciler.Connection).Get(request.Original.Status.OIDCConfigID)
	if err != nil {
		return fmt.Errorf("unable to get oidc config from ocm - %w", err)
	}

	// update the rosa cluster if it does exist
	request.Log.Info("updating rosa cluster", request.logValues()...)
	_, err = request.OCMClient.Update(request.Desired.Builder(oidc))
	if err != nil {
		return fmt.Errorf("unable to update rosa cluster in ocm - %w", err)
	}

	// create an event indicating that the rosa cluster has been updated
	events.RegisterAction(
		events.Updated,
		request.Original,
		request.Reconciler.Recorder,
		request.Desired.Spec.DisplayName,
		request.Original.Status.ClusterID,
	)

	return nil
}

// ensureOIDCProvider creates the OIDC Provider in AWS.
func (request *ROSAClusterRequest) ensureOIDCProvider() (config *clustersmgmtv1.OidcConfig, err error) {
	original := request.Original.DeepCopy()

	// create oidc config only if we have not created it already
	if request.Original.Status.OIDCConfigID == "" {
		request.Log.Info("creating oidc config", request.logValues()...)
		config, err := ocm.NewOIDCConfigClient(request.Reconciler.Connection).Create()
		if err != nil {
			return config, fmt.Errorf("unable to create oidc config - %w", err)
		}

		// update the status with the oidc config id
		request.Original.Status.OIDCConfigID = config.ID()
		if err := kubernetes.PatchStatus(request.Context, request.Reconciler, original, request.Original); err != nil {
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
		request.Log.Info("creating oidc provider", request.logValues()...)
		providerARN, err := aws.CreateOIDCProvider(config.IssuerUrl())
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

// ensureOperatorRoles creates the operator roles in AWS.
func (request *ROSAClusterRequest) ensureOperatorRoles(oidc *clustersmgmtv1.OidcConfig) error {
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
	if err := stsClient.CreateOperatorRoles(request.Desired.Spec.OpenShiftVersion, requests...); err != nil {
		return fmt.Errorf("unable to create operator roles - %w", err)
	}

	return nil
}
