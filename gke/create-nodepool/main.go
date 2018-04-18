package main

import (
	"log"
	"context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/container/v1"
	"os"

)

type client struct {
	containerClient *container.Service
}

func NewClient(ctx context.Context) *client {

	hc, err := google.DefaultClient(ctx, container.CloudPlatformScope)
	if err != nil {
		log.Panic(err)
	}
	containerSvc, err := container.New(hc)

	if err != nil {
		log.Panic(err)
	}


	return &client{
		containerClient: containerSvc,
	}
}

func main() {
	ctx := context.Background()

	projectID, ok := os.LookupEnv("GKE_PROJECT_ID")
	if !ok {
		log.Fatal("set GKE_PROJECT_ID")
	}
	zone, ok := os.LookupEnv("GKE_ZONE")
	if !ok {
		log.Fatal("set GKE_ZONE")
	}
	clusterID, ok := os.LookupEnv("GKE_CLUSTER_ID")
	if !ok {
		log.Fatal("set GKE_CLUSTER_ID")
	}
	nodePoolID, ok := os.LookupEnv("GKE_NODE_POOL_ID")
	if !ok {
		log.Fatal("set GKE_NODE_POOL_ID")
	}

	machineType, ok := os.LookupEnv("GKE_MACHINE_TYPE")

	cl := NewClient(ctx)

	req := container.CreateNodePoolRequest{
		NodePool: &container.NodePool{
			InitialNodeCount: int64(1),
			Name: nodePoolID,
			Config: &container.NodeConfig{
				MachineType: machineType,
			},
			Version: "latest",
		},
	}

	_, err := cl.containerClient.Projects.Zones.Clusters.NodePools.Create(projectID, zone, clusterID, &req).Do()
	if err != nil {
		log.Fatal("failed to get nodepools")
	}

}