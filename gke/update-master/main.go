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

	forceUpdate, ok := os.LookupEnv("GKE_FORCE_UPDATE")

	if err != nil {
		log.Fatal(err)
	}

	cl, err := svc.Projects.Zones.Clusters.Get(projectID, zone, clusterID).Do()

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Cluster %q (%s) master_version: v%s, node_count: %d\n", cl.Name, cl.Status,
		cl.CurrentMasterVersion, cl.CurrentNodeCount)

	if err != nil {
		log.Fatal("Failed to parse master version")
	}

	ops, err := svc.Projects.Zones.Operations.List(projectID, zone).Do()

	if err != nil {
		log.Fatal(err)
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
		return
	}
	sc, err := svc.Projects.Zones.GetServerconfig(projectID, zone).Do()

	if err != nil {
		log.Fatal("failed to get server config")
	}

	versionToUpdate := "latest"

	latestMasterVersion, _ := version.NewVersion(sc.ValidMasterVersions[0])
	currentMasterVersion, _ := version.NewVersion(cl.CurrentMasterVersion)

	if (latestMasterVersion.Segments()[1] - currentMasterVersion.Segments()[1]) > 1 {
		versionToUpdate = fmt.Sprintf("%d.%d", currentMasterVersion.Segments()[0], currentMasterVersion.Segments()[1]+1)
	}

	if currentMasterVersion.LessThan(latestMasterVersion) {

		fmt.Printf("Available version: %s, current version: %s\n", versionToUpdate, cl.CurrentMasterVersion)

		if len(forceUpdate) > 0 {
			upRequest := container.UpdateClusterRequest{
				Name: fmt.Sprintf("projects/%s/locations/%s/clusters/%s", projectID, zone, clusterID),
				Update: &container.ClusterUpdate{
					DesiredMasterVersion: versionToUpdate,
				},
			}

			op, err := svc.Projects.Zones.Clusters.Update(projectID, zone, clusterID, &upRequest).Do()

			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("We're going to %s, status %s, id: %s\n", op.OperationType, op.Status, op.Name)
		}

	} else {
		fmt.Printf("Already latest version %s\n", currentMasterVersion)
	}

}
