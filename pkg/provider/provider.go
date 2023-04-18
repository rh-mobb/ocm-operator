package provider

type Provider interface {
	HasInstanceType(string) (bool, error)
}

func Fetch() Provider {
	return &ROSA{}
}
