package kubernetes

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FakeClient represents a fake client used to satisfy the Client interface.  This is
// used only for testing purposes.
type FakeClient struct{}

func (fake *FakeClient) List(_ context.Context, _ client.ObjectList, _ ...client.ListOption) error {
	return nil
}

func (fake *FakeClient) Status() client.SubResourceWriter {
	return &fakeStatusWriter{}
}

// fakeStatusWriter represents a fake client used to satisfy the SubResourceWriter
// interface.
type fakeStatusWriter struct{}

func (w *fakeStatusWriter) Create(_ context.Context, _ client.Object, _ client.Object, _ ...client.SubResourceCreateOption) error {
	return nil
}

func (w *fakeStatusWriter) Update(_ context.Context, _ client.Object, _ ...client.SubResourceUpdateOption) error {
	return nil
}

func (w *fakeStatusWriter) Patch(_ context.Context, _ client.Object, _ client.Patch, _ ...client.SubResourcePatchOption) error {
	return nil
}
