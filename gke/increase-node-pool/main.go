package main

import (
	"log"
	"context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/compute/v1"
	"os"

	"flag"
	"path/filepath"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"strings"
	"fmt"
)

type client struct {
	containerClient *container.Service
	computeClient *compute.Service
	kubeClient *kubernetes.Clientset
}


func GetConfigInCluster() (*restclient.Config, error) {
	config, err := restclient.InClusterConfig()
	if err != nil {
		return nil, err
	}

	return config, nil
}

func GetConfigOutOfCluster() (*restclient.Config, error) {
	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}


func NewClient(ctx context.Context) *client {

	hc, err := google.DefaultClient(ctx, container.CloudPlatformScope)
	if err != nil {
		log.Panic(err)
	}
	containerSvc, err := container.New(hc)
	computeSvc, err := compute.New(hc)

	if err != nil {
		log.Panic(err)
	}


	config, err := GetConfigInCluster()

	if err != nil {
		config, err = GetConfigOutOfCluster()
		if err != nil {
			panic(err.Error())
		}
	}

	k8sClient, err := kubernetes.NewForConfig(config)

	if err != nil {
		log.Fatal(err.Error())
	}

	return &client{
		containerClient: containerSvc,
		computeClient: computeSvc,
		kubeClient: k8sClient,
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

	cl := NewClient(ctx)


	np, err := cl.containerClient.Projects.Zones.Clusters.NodePools.Get(projectID, zone, clusterID, nodePoolID).Do()
	if err != nil {
		log.Fatal("failed to get nodepools")
	}

	parts := strings.Split(np.InstanceGroupUrls[0], "/")
	igUrl := parts[len(parts)-1]

	if err != nil {
		log.Fatal("failed to get instance group managers")
	}


	ig, err := cl.computeClient.InstanceGroupManagers.Get(projectID, zone, igUrl).Do()

	fmt.Printf("ig %s, size %d\n", ig.Name, ig.TargetSize)
	_, err = cl.computeClient.InstanceGroupManagers.Resize(projectID, zone, igUrl, ig.TargetSize + 1).Do()

	if err != nil {
		log.Fatal("failed to resize nodepool")
	}


}