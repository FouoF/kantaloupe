package env

import "github.com/dynamia-ai/kantaloupe/pkg/constants"

var (
	InitImageEnvName = Register(constants.InitImageEnvKeyName,
		"ghcr.io/dynamia-ai/init-image:v0.0.2",
		"InitImageEnvName is the environment variable name of the init image.")

	EnvLibCudaLogLevel = Register(constants.EnvLibCudaLogLevel,
		"2",
		"EnvLibCudaLogLevel is the environment variable name of the HAMi log level.")

	CleanupInactiveWorkloadThreshold = Register(constants.CleanupInactiveWorkloadThreshold,
		"3600",
		"CleanupInactiveWorkloadThreshold is the seconds of to cleanup invactive workload.")

	GatewayEnvBaseURL = Register(constants.GatewayEnvBaseURL,
		"/kantaloupe.dynamia.ai/",
		"GatewayEnvBaseURL is the environment variable name of the gateway base url.")

	GatewayEndpoint = Register(constants.GatewayEndpointEnv,
		"",
		"GatewayEndpoint is the environment variable name of the gateway endpoint.")

	GatewayEnvNamePortStart = Register(constants.GatewayEnvNamePortStart,
		30000,
		"GatewayEnvNamePortStart is the environment variable name of the gateway port start.")

	GatewayEnvNamePortCount = Register(constants.GatewayEnvNamePortCount,
		30000,
		"GatewayEnvNamePortCount is the environment variable name of the gateway port count.")

	SkipCheckClusterKubesystemID = Register(constants.SkipCheckClusterKubesystemID,
		"false",
		"SkipCheckClusterKubesystemID is the environment variable name of to skip check cluster kubesystem ID.")
)
