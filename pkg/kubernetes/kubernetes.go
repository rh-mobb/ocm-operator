package kubernetes

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	optimisticLockErrorMessage = "the object has been modified; please apply your changes to the latest version and try again"
)

type Client interface {
	Get(context.Context, types.NamespacedName, client.Object, ...client.GetOption) error
	List(context.Context, client.ObjectList, ...client.ListOption) error
	Status() client.SubResourceWriter
}

// PatchStatus patches the status of a kubernetes resource.
func PatchStatus(
	ctx context.Context,
	reconciler Client,
	current, patched client.Object,
) error {
	// run the patch
	if err := reconciler.Status().Patch(ctx, patched, client.MergeFrom(current)); err != nil {
		if isOptimisticLockError(err) {
			return nil
		}

		return fmt.Errorf("unable to patch status - %w", err)
	}

	return nil
}

func isOptimisticLockError(err error) bool {
	return strings.Contains(err.Error(), optimisticLockErrorMessage)
}
