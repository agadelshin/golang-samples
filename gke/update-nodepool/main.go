package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/hashicorp/go-version"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/container/v1"
)

func NewClient(ctx context.Context) (*container.Service, error) {
	hc, err := google.DefaultClient(ctx, container.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	svc, err := container.New(hc)

	if err != nil {
		return nil, err
	}

	return svc, nil

}

func main() {
	ctx := context.Background()
	svc, err := NewClient(ctx)

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

	if err != nil {
		log.Fatal(err)
	}

	cl, err := svc.Projects.Zones.Clusters.Get(projectID, zone, clusterID).Do()

	if err != nil {
		log.Fatal(err)
	}



	fmt.Printf("Cluster %q (%s) master_version: v%s, node_count: %d\n", cl.Name, cl.Status,
		cl.CurrentMasterVersion, cl.CurrentNodeCount)

	ops, err := svc.Projects.Zones.Operations.List(projectID, zone).Do()

	if err != nil {
		log.Fatal(err)
	}

	masterVersion, err := version.NewVersion(cl.CurrentMasterVersion)

	for _, np := range cl.NodePools {
		nodeVersion, err := version.NewVersion(np.Version)
		if err != nil {
			log.Printf("Failed to parse node version %s for %s nodepool", np.Version, np.Name)
			continue
		}

		fmt.Printf("\tNodePool %q (%s), np_version: v%s, labels: %v, need_to_upgrade: %t\n",
			np.Name, np.Status, np.Version, np.Config.Labels, nodeVersion.LessThan(masterVersion))

	}

	opsInProgress := make([]*container.Operation, 0)

	for _, op := range ops.Operations {
		if op.Status != "DONE" {
			opsInProgress = append(opsInProgress, op)
		}
	}

	if len(opsInProgress) > 0 {
		for _, op := range opsInProgress {
			fmt.Printf("Operation in progress id: %s, type: %s, status: %s\n", op.Name, op.OperationType, op.Status)
		}

		if len(nodePoolID) > 0 {
			fmt.Println("Skipping upgrade")
		}
		return
	}

	if len(nodePoolID) > 0 {
		upRequest := container.UpdateClusterRequest{
			Name: fmt.Sprintf("projects/%s/locations/%s/clusters/%s", projectID, zone, clusterID),
			Update: &container.ClusterUpdate{
				DesiredNodePoolId:  nodePoolID,
				DesiredNodeVersion: cl.CurrentMasterVersion,
			},
		}

		op, err := svc.Projects.Zones.Clusters.Update(projectID, zone, clusterID, &upRequest).Do()

		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("We're going to %s, status %s, id: %s\n", op.OperationType, op.Status, op.Name)
	}

}
