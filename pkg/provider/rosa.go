package provider

type ROSA struct {
}

// HasInstanceType returns whether or not a provider has a specific instance
// type based on a string input.
func (rosa *ROSA) HasInstanceType(instanceType string) (bool, error) {
	return true, nil
}
