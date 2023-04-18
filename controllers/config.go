package controllers

// Config represents the startup options used to start each of the controllers
// in this operator.
type Config struct {
	EnableLeaderElection bool
	MetricsAddress       string
	ProbeAddress         string
	TokenFile            string
}
