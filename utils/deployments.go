package utils

type deployment struct {
	Name           string
	ContainerName  string
	DockerHubImage string
}

var Deployments = []deployment{
	{Name: "deployment/girus-frontend", ContainerName: "frontend", DockerHubImage: "linuxtips/girus-frontend"},
	{Name: "deployment/girus-backend", ContainerName: "backend", DockerHubImage: "linuxtips/girus-backend"}}
