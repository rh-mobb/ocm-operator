package kubernetes

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Client interface {
	List(context.Context, client.ObjectList, ...client.ListOption) error
	Status() client.SubResourceWriter
}
