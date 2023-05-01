package gitlabidentityprovider

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	gitlab "github.com/xanzy/go-gitlab"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	ocmv1alpha1 "github.com/rh-mobb/ocm-operator/api/v1alpha1"
	"github.com/rh-mobb/ocm-operator/controllers"
	"github.com/rh-mobb/ocm-operator/pkg/conditions"
	"github.com/rh-mobb/ocm-operator/pkg/identityprovider"
	"github.com/rh-mobb/ocm-operator/pkg/kubernetes"
	"github.com/rh-mobb/ocm-operator/pkg/ocm"
	"github.com/rh-mobb/ocm-operator/pkg/triggers"
)

var (
	ErrMissingClusterID                     = errors.New("unable to find cluster id")
	ErrMissingAccessToken                   = errors.New("unable to locate gitlab api access token data")
	ErrMissingClientSecret                  = errors.New("unable to locate client secret data")
	ErrMissingCA                            = errors.New("ca specified but unable to locate ca data")
	ErrGitLabIdentityProviderRequestConvert = errors.New("unable to convert generic request to machine pool request")
)

// GitLabIdentityProviderRequest is an object that is unique to each reconciliation
// request.
type GitLabIdentityProviderRequest struct {
	Context           context.Context
	ControllerRequest ctrl.Request
	Current           *ocmv1alpha1.GitLabIdentityProvider
	Original          *ocmv1alpha1.GitLabIdentityProvider
	Desired           *ocmv1alpha1.GitLabIdentityProvider
	Log               logr.Logger
	Trigger           triggers.Trigger
	Reconciler        *Controller
	GitLabClient      *identityprovider.GitLab
	OCMClient         *ocm.GitLabIdentityProviderClient

	// data obtained during request reconciliation
	AccessToken  string
	ClientID     string
	ClientSecret string
}

func (r *Controller) NewRequest(ctx context.Context, req ctrl.Request) (controllers.Request, error) {
	original := &ocmv1alpha1.GitLabIdentityProvider{}

	// get the object (desired state) from the cluster
	//nolint:wrapcheck
	if err := r.Get(ctx, req.NamespacedName, original); err != nil {
		if !apierrs.IsNotFound(err) {
			return &GitLabIdentityProviderRequest{}, fmt.Errorf("unable to fetch cluster object - %w", err)
		}

		return &GitLabIdentityProviderRequest{}, err
	}

	// get the secret access token data from the cluster
	accessToken, err := kubernetes.GetSecretData(ctx, r, original.Spec.AccessTokenSecret, req.Namespace, ocmv1alpha1.AccessTokenKey)
	if accessToken == "" {
		if err == nil {
			return &GitLabIdentityProviderRequest{}, accessTokenError(original, ErrMissingAccessToken)
		}

		return &GitLabIdentityProviderRequest{}, accessTokenError(original, err)
	}

	// create the api client used to interact with gitlab
	gitlabClient, err := gitlab.NewClient(accessToken, gitlab.WithBaseURL(original.Spec.URL+"/api/v4"))
	if err != nil {
		return &GitLabIdentityProviderRequest{}, fmt.Errorf("error creating gitlab api client - %w", err)
	}

	// create the desired state of the request based on the inputs
	desired := original.DeepCopy()
	if desired.Spec.DisplayName == "" {
		desired.Spec.DisplayName = desired.Name
	}

	return &GitLabIdentityProviderRequest{
		Original:          original,
		Desired:           desired,
		ControllerRequest: req,
		Context:           ctx,
		Log:               log.Log,
		Trigger:           triggers.GetTrigger(original),
		Reconciler:        r,
		GitLabClient:      &identityprovider.GitLab{Client: gitlabClient},

		// data obtained from cluster
		AccessToken: accessToken,
	}, nil
}

func (request *GitLabIdentityProviderRequest) GetObject() controllers.Workload {
	return request.Original
}

// execute executes a variety of different phases for the request.
//
//nolint:wrapcheck
func (request *GitLabIdentityProviderRequest) execute(phases ...Phase) (ctrl.Result, error) {
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
func (request *GitLabIdentityProviderRequest) updateCondition(condition *metav1.Condition) error {
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

// updateStatusCluster updates fields related to the cluster in which the machine pool resides in.
// TODO: centralize this function into controllers or conditions package.
func (request *GitLabIdentityProviderRequest) updateStatusCluster() error {
	// retrieve the cluster id
	clusterClient := ocm.NewClusterClient(request.Reconciler.Connection, request.Desired.Spec.ClusterName)
	cluster, err := clusterClient.Get()
	if err != nil {
		return fmt.Errorf(
			"unable to retrieve cluster from ocm [name=%s] - %w",
			request.Desired.Spec.ClusterName,
			err,
		)
	}

	// if the cluster id is missing return an error
	if cluster.ID() == "" {
		return fmt.Errorf("missing cluster id in response - %w", ErrMissingClusterID)
	}

	// keep track of the original object
	original := request.Original.DeepCopy()
	request.Original.Status.ClusterID = cluster.ID()
	request.Original.Status.CallbackURL = ocm.GetCallbackURL(cluster, request.Desired.Spec.DisplayName)

	// store the cluster id in the status
	if err := kubernetes.PatchStatus(request.Context, request.Reconciler, original, request.Original); err != nil {
		return fmt.Errorf(
			"unable to update status.clusterID=%s - %w",
			cluster.ID(),
			err,
		)
	}

	return nil
}

// logValues produces a consistent set of log values for this request.
func (request *GitLabIdentityProviderRequest) logValues() []interface{} {
	return []interface{}{
		"resource", fmt.Sprintf("%s/%s", request.Desired.Namespace, request.Desired.Name),
		"cluster", request.Desired.Spec.ClusterName,
		"name", request.Desired.Spec.DisplayName,
		"type", "gitlab",
	}
}

func (request *GitLabIdentityProviderRequest) desired() bool {
	if request.Desired == nil || request.Current == nil {
		return false
	}

	return reflect.DeepEqual(
		request.Desired.Spec,
		request.Current.Spec,
	)
}

func accessTokenError(from *ocmv1alpha1.GitLabIdentityProvider, err error) error {
	return fmt.Errorf(
		"unable to retrieve access token from [%s/%s] at key [%s] - %w",
		from.Namespace,
		from.Spec.AccessTokenSecret,
		ocmv1alpha1.AccessTokenKey,
		err,
	)
}
