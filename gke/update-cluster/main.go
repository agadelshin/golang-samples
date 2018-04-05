package main

import (
	"context"
	"fmt"
	"log"
	"os"

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
	if ! ok {
		log.Fatal("set GKE_PROJECT_ID")
	}
	zone, ok := os.LookupEnv("GKE_ZONE")
	if ! ok {
		log.Fatal("set GKE_ZONE")
	}
	clusterID, ok := os.LookupEnv("GKE_CLUSTER_ID")
	if ! ok {
		log.Fatal("set GKE_CLUSTER_ID")
	}

	if err != nil {
		log.Fatal(err)
	}

	cl, err := svc.Projects.Zones.Clusters.Get(projectID, zone, clusterID).Do()

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Cluster %q (%s) master_version: v%s, node_count: %d\n", cl.Name, cl.Status,
		cl.CurrentMasterVersion, cl.CurrentNodeCount)


	upRequest := container.UpdateClusterRequest{
		Name: fmt.Sprintf("projects/%s/locations/%s/clusters/%s", projectID, zone, clusterID),
		Update: &container.ClusterUpdate{
			DesiredMasterVersion: "latest",
		},
	}

	op, err := svc.Projects.Zones.Clusters.Update(projectID, zone, clusterID, &upRequest).Do()

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("We're going to %s, status %s, id: %s\n", op.OperationType, op.Status, op.Name)
}
