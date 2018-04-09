package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func GetNodeIP(k8sClient *kubernetes.Clientset, hostname string, ipType v1.NodeAddressType) (string, error) {
	node, err := k8sClient.CoreV1().Nodes().Get(hostname, metav1.GetOptions{})

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

func main() {

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

	hostname, ok := os.LookupEnv("HOSTNAME")

	if !ok {
		log.Fatal("set HOSTNAME")
	}

	externalAddress, err := GetNodeIP(k8sClient, hostname, "ExternalIP")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(externalAddress)
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
