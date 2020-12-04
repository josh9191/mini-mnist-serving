package constants

const (
	ProdNamespace   = "mnist-prod"
	CanaryNamespace = "mnist-canary"
)

const (
	IngressName     = "mnist-ingress"
	ModelSecretName = "mnist-secret"
	DeploymentName  = "mnist-deploy"
	ServiceName     = "mnist-svc"
)

const (
	LabelAppSelector = "mnist"
)

type Strategy int

const (
	None Strategy = -1 + iota
	CurrentModelOnly
	NewModelOnly
	Canary
)

const (
	CanaryHeader = "UseCanary"
)
