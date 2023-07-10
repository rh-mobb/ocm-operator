package kubernetes

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FakeClient represents a fake client used to satisfy the Client interface.  This is
// used only for testing purposes.
type FakeClient struct{}

func (fake *FakeClient) Get(_ context.Context, _ types.NamespacedName, _ client.Object, _ ...client.GetOption) error {
	return nil
}

func (fake *FakeClient) List(_ context.Context, _ client.ObjectList, _ ...client.ListOption) error {
	return nil
}

func (fake *FakeClient) Patch(_ context.Context, _ client.Object, _ client.Patch, _ ...client.PatchOption) error {
	return nil
}

func (fake *FakeClient) Status() client.SubResourceWriter {
	return &fakeStatusWriter{}
}

// FakeErrorClient represents a fake client that is used to handle errors.  It is used
// to satisfy the Client interface and for testing purposes only.
type FakeErrorClient struct{}

func (fake *FakeErrorClient) Get(_ context.Context, _ types.NamespacedName, _ client.Object, _ ...client.GetOption) error {
	return fmt.Errorf("error in Get")
}

func (fake *FakeErrorClient) List(_ context.Context, _ client.ObjectList, _ ...client.ListOption) error {
	return fmt.Errorf("error in List")
}

func (fake *FakeErrorClient) Patch(_ context.Context, _ client.Object, _ client.Patch, _ ...client.PatchOption) error {
	return fmt.Errorf("error in Patch")
}

func (fake *FakeErrorClient) Status() client.SubResourceWriter {
	return &fakeErrorStatusWriter{}
}

// fakeStatusWriter represents a fake client used to satisfy the client.SubResourceWriter
// interface.
type fakeStatusWriter struct{}

func (w *fakeStatusWriter) Create(_ context.Context, _, _ client.Object, _ ...client.SubResourceCreateOption) error {
	return nil
}

func (w *fakeStatusWriter) Update(_ context.Context, _ client.Object, _ ...client.SubResourceUpdateOption) error {
	return nil
}

func (w *fakeStatusWriter) Patch(_ context.Context, _ client.Object, _ client.Patch, _ ...client.SubResourcePatchOption) error {
	return nil
}

// fakeStatusWriter represents a fake client used to satisfy the client.SubResourceWriter
// interface.  It is used to handle errors.
type fakeErrorStatusWriter struct{}

func (w *fakeErrorStatusWriter) Create(_ context.Context, _, _ client.Object, _ ...client.SubResourceCreateOption) error {
	return fmt.Errorf("error in Create")
}

func (w *fakeErrorStatusWriter) Update(_ context.Context, _ client.Object, _ ...client.SubResourceUpdateOption) error {
	return fmt.Errorf("error in Update")
}

func (w *fakeErrorStatusWriter) Patch(_ context.Context, _ client.Object, _ client.Patch, _ ...client.SubResourcePatchOption) error {
	return fmt.Errorf("error in Patch")
}
