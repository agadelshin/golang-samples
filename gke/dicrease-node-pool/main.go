package main

import (
	"log"
	"context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/container/v1"
	compute "google.golang.org/api/compute/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	//"strings"
	"fmt"
	"flag"
	"path/filepath"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/apimachinery/pkg/labels"
	"strings"
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

	set := labels.Set(map[string]string{"cloud.google.com/gke-nodepool": nodePoolID})

	nodes, err := cl.kubeClient.CoreV1().Nodes().List(metav1.ListOptions{
		LabelSelector:  labels.SelectorFromSet(set).String(),
		FieldSelector: "spec.unschedulable==true",
	})

	if err != nil {
		log.Fatal("failed to get nodes")
	}

	req := &compute.InstanceGroupManagersDeleteInstancesRequest{}

	for _, n := range nodes.Items {
		req.Instances = append(req.Instances, fmt.Sprintf("zones/%s/instances/%s", zone, n.Labels["kubernetes.io/hostname"]))
		fmt.Printf("to remove: %s\n", n.Labels["kubernetes.io/hostname"])
	}

	if len(req.Instances) > 0 {
		_, err := cl.computeClient.InstanceGroupManagers.DeleteInstances(projectID, zone, igUrl, req).Do()
		if err != nil {
			log.Fatal("failed to delete nodes")
		}
	} else {
		fmt.Println("empty list")
	}

}