package provider

var Version = "1.0.0" // needs to be exported so make file can update this
var productId = "terraform-provider-apptrust/" + Version

// Minimum required versions for AppTrust
const (
	MinArtifactoryVersion = "7.125.0" // Minimum Artifactory version required for AppTrust
	MinXrayVersion        = "3.130.5" // Minimum Xray version required for AppTrust
)
