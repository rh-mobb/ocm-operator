package ocm

const (
	LabelPrefixManaged = "ocm.mobb.redhat.com/managed"
	LabelPrefixName    = "ocm.mobb.redhat.com/name"
)

// ManagedLabels returns all of the managed labels.
func ManagedLabels() []string {
	return []string{
		LabelPrefixManaged,
		LabelPrefixName,
	}
}
