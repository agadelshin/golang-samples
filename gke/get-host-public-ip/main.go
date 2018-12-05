package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
	container "google.golang.org/api/container/v1"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type client struct {
	computeClient   *compute.Service
	containerClient *container.Service
	kubeClient      *kubernetes.Clientset
	projectID       string
	zone            string
}

func newClient(ctx context.Context) *client {

	hc, err := google.DefaultClient(ctx, container.CloudPlatformScope)
	if err != nil {
		log.Panic(err)
	}
	containerSvc, err := container.New(hc)
	computeSvc, err := compute.New(hc)

	if err != nil {
		log.Panic(err)
	}

	config, err := getConfigInCluster()

	if err != nil {
		config, err = getConfigOutOfCluster()
		if err != nil {
			panic(err.Error())
		}
	}

	k8sClient, err := kubernetes.NewForConfig(config)

	if err != nil {
		log.Fatal(err.Error())
	}

	projectID, ok := os.LookupEnv("GKE_PROJECT_ID")
	if !ok {
		log.Fatal("set GKE_PROJECT_ID")
	}
	zone, ok := os.LookupEnv("GKE_ZONE")
	if !ok {
		log.Fatal("set GKE_ZONE")
	}

	return &client{
		containerClient: containerSvc,
		computeClient:   computeSvc,
		kubeClient:      k8sClient,
		projectID:       projectID,
		zone:            zone,
	}
}

// GetNodeIPFromGCP returns external IP from GCP or Kubernetes API
func (cl *client) GetNodeIP(hostname string) (externalAddress string, exists bool, err error) {
	externalAddress, exists, err = cl.getNodeIPFromGCP(hostname)
	if err != nil {
		log.Println(fmt.Errorf("failed to get ip from GCP: %+v", err))
		externalAddress, err = cl.getNodeIPFromKube(hostname, "ExternalIP")
		if err != nil {
			return "", false, err
		}
		return externalAddress, true, err
	}
	return externalAddress, exists, err
}

func (cl *client) getNodeIPFromGCP(hostname string) (externalAddress string, exists bool, err error) {
	instance, err := cl.computeClient.Instances.Get(cl.projectID, cl.zone, hostname).Do()
	if err != nil {
		return "", false, err
	}
	for _, ni := range instance.NetworkInterfaces {
		for _, ac := range ni.AccessConfigs {
			if ac.Type == "ONE_TO_ONE_NAT" {
				return ac.NatIP, true, nil
			}
		}
	}

	return "", false, nil
}

func (cl *client) getNodeIPFromKube(hostname string, ipType v1.NodeAddressType) (string, error) {
	node, err := cl.kubeClient.CoreV1().Nodes().Get(hostname, metav1.GetOptions{})

	if err != nil {
		log.Fatal(err.Error())
	}

	for _, a := range node.Status.Addresses {
		if a.Type == ipType {
			return a.Address, nil
		}
	}

	return "", fmt.Errorf("Failed to return node %s IP", ipType)
}

func getConfigInCluster() (*restclient.Config, error) {
	config, err := restclient.InClusterConfig()
	if err != nil {
		return nil, err
	}

	return config, nil
}

func getConfigOutOfCluster() (*restclient.Config, error) {
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

func main() {
	ctx := context.Background()
	cl := newClient(ctx)

	hostname, ok := os.LookupEnv("HOSTNAME")

	if !ok {
		log.Fatal("set HOSTNAME")
	}

	externalAddress, exists, err := cl.GetNodeIP(hostname)
	if err != nil {
		log.Fatal(err)
	}
	if !exists {
		log.Fatal("node doesn't have an external IP")
	}

	fmt.Println(externalAddress)
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
