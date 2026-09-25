package utils

type DeploymentMap struct {
	Name           string
	ContainerName  string
	DockerHubImage string
}

var Deployments = map[string]DeploymentMap{
	"frontend": {Name: "deployment/girus-frontend", ContainerName: "frontend", DockerHubImage: "linuxtips/girus-frontend"},
	"backend":  {Name: "deployment/girus-backend", ContainerName: "backend", DockerHubImage: "linuxtips/girus-backend"},
}
