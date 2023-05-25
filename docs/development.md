# Developer Guide

This guide aims to document the procedures and tools needed to develop against this project.

## Development Prerequisites

The following tooling and/or infrastructure is required prior to starting development
against this project.

* Kubernetes Cluster (tested against [ROSA](https://mobb.ninja/docs/quickstart-rosa/))
* Make
* [Go => 1.20](https://go.dev/doc/install)
* [oc](https://docs.openshift.com/container-platform/4.12/cli_reference/openshift_cli/getting-started-cli.html)
* [operator-sdk](https://sdk.operatorframework.io/docs/installation/)
* [aws-cli](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-configure.html)
* [Retrieve your access token](https://mobb.ninja/docs/quickstart-rosa/#get-a-red-hat-offline-access-token)

## Getting Started

1. Configure your KUBECONFIG appropriately.  The KUBECONFIG is used to authenticate against your Kubernetes API.  Please 
note that the followon steps assume you are using OpenShift.  While some commands may work on regular Kubernetes, it 
is not tested or validated.

2. Set the `OCM_TOKEN` environment variable as retrieved in the [prereqs](#development-prerequisites) guide.  This
OCM token allows you to authenticate against the OCM API to perform various functions.

```bash
export OCM_TOKEN=<my_ocm_token>
```

3. Ensure you are able to login to AWS via the CLI.  This ensures that your AWS configuration is correct.  It should 
be noted that we only interact with AWS when installing clusters at this time.  If testing other objects, you may not
need to perform this step.

```bash
aws sts get-caller-identity
```

4. Generate and install the CRDs to the cluster.  It is always wise to generate and install before running, as code that 
you change may affect the CRDs.

```bash
make install
```

5. Run the operator.  This will run a local copy of the controller against the Kubernetes API that you have configured 
in step 1.  This avoids the issues of having to generate an image, pushing to a registry, and installing the controller 
in the cluster prior to testing.

```bash
make run
```

6. Test the operator.  Samples are located in the `config/samples` directory for various different configurations
that exist for the controlled objects (deploying a ROSA cluster is the example used below).

```bash
oc apply -f config/samples/cluster/rosa_sample.yaml
```

7. In order to cleanup.

```bash
make uninstall
```

## Generating a new API

From time to time, you will need to generate a new API for the controller to reconcile against.  This 
API is a go representation of a Kubernetes CRD.  When generating a new API, you get some boilerplate API 
code and a boilerplate controller.

1. To generate a new API:

```
operator-sdk create api --group ocm --kind <kind> --version v1alpha1 --controller --resource

Example:
operator-sdk create api --group ocm --kind OIDCIdentityProvider --version v1alpha1 --controller --resource
```

2. For code organization, the controllers for each project belong into their own package.

```
export NAME=oidcidentityprovider
mkdir -p controllers/$NAME
mv controllers oidcidentityprovider.go controllers/$NAME/controller.go
```

3. The controller interface is defined meaning that your controller needs to implement several different functions.

```go
// Controller represents the object that is performing the reconciliation
// action.
type Controller interface {
	kubernetes.Client

	NewRequest(ctx context.Context, req ctrl.Request) (Request, error)
	Reconcile(context.Context, ctrl.Request) (ctrl.Result, error)
	ReconcileCreate(Request) (ctrl.Result, error)
	ReconcileUpdate(Request) (ctrl.Result, error)
	ReconcileDelete(Request) (ctrl.Result, error)
	SetupWithManager(mgr ctrl.Manager) error
}
```

You should put these functions in the `controller.go` file that you copied in the previous step.  You can see a good 
example of these in the `controllers/rosacluster/controller.go` file.

The controller should also call the centralized `Reconcile` function in the `controller.go` file:

```go
func (r *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return controllers.Reconcile(ctx, r, req)
}
```

4. For each time the reconcile loop happens, a request is generated.  You need to define what this request
looks like via a `NewRequest` function to satisfy the `Controller` interface above.  An example of this can
be found in the `controllers/rosacluster/request.go` file.

```go
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
	AWSClient         *aws.Client

	// data obtained during request reconciliation
	Cluster *clustersmgmtv1.Cluster
	Version *clustersmgmtv1.Version
}

func (r *Controller) NewRequest(ctx context.Context, req ctrl.Request) (controllers.Request, error) {
    // logic to create your request goes here
}
```

5. Next, you need to define your phases in a `controllers/my-controller/phases.go` file.  These are 
individual steps of the reconciliation process.  They execute against the controller and operate 
against the request.  The request is used to store data to pass to other phases of reconciliation.
Here is an example phase:

```go
// GetCurrentState gets the current state of the LDAPIdentityProvider resource.  The current state of the LDAPIdentityProvider resource
// is stored in OpenShift Cluster Manager.  It will be compared against the desired state which exists
// within the OpenShift cluster in which this controller is reconciling against.
func (r *Controller) GetCurrentState(request *ROSAClusterRequest) (ctrl.Result, error) {
	// retrieve the cluster
	request.OCMClient = ocm.NewClusterClient(request.Reconciler.Connection, request.Desired.Spec.DisplayName)

	cluster, err := request.OCMClient.Get()
	if err != nil {
		return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf(
			"unable to retrieve cluster from ocm [name=%s] - %w",
			request.Desired.Spec.DisplayName,
			err,
		)
	}

	// return immediately if we have no cluster
	if cluster == nil {
		return controllers.NoRequeue(), nil
	}

	// store the current state
	request.Current = &ocmv1alpha1.ROSACluster{}
	request.Current.Spec.DisplayName = request.Desired.Spec.DisplayName
	request.Current.CopyFrom(cluster)
	request.Cluster = cluster

	return controllers.NoRequeue(), nil
}
```

6. Finally, you need to define the phases, in order, that the controller will execute for each create, update, or 
delete operation.  These functions need to be named `ReconcileCreate`, `ReconcileUpdate` and `ReconcileDelete` 
respectively to satisfy the `Controller`.  Here is an example using `ReconcileCreate`:

```go
// ReconcileCreate performs the reconciliation logic when a create event triggered
// the reconciliation.
func (r *Controller) ReconcileCreate(req controllers.Request) (ctrl.Result, error) {
	// type cast the request to a ldap identity provider request
	request, ok := req.(*ROSAClusterRequest)
	if !ok {
		return controllers.RequeueAfter(defaultClusterRequeue), ErrClusterRequestConvert
	}

	// add the finalizer
	if err := controllers.AddFinalizer(request.Context, r, request.Original); err != nil {
		return controllers.RequeueAfter(defaultClusterRequeue), fmt.Errorf("unable to register delete hooks - %w", err)
	}

	// execute the phases
	return controllers.Execute(request, request.ControllerRequest, []controllers.Phase{
		{Name: "GetCurrentState", Function: func() (ctrl.Result, error) { return r.GetCurrentState(request) }},
		{Name: "ApplyCluster", Function: func() (ctrl.Result, error) { return r.ApplyCluster(request) }},
		{Name: "WaitUntilReady", Function: func() (ctrl.Result, error) { return r.WaitUntilReady(request) }},
		{Name: "Complete", Function: func() (ctrl.Result, error) { return r.Complete(request) }},
	}...)
}
```

7. Be sure to lint your work to ensure we conform to common coding standards once done!

```bash
make lint
```