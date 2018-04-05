package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	"k8s.io/apimachinery/pkg/labels"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

var (
	k8sClient *kubernetes.Clientset
)

func GetPods(l map[string]string, state, namespace string) (*v1.PodList, error) {
	set := labels.Set(l)
	pods, err := k8sClient.CoreV1().Pods(namespace).List(metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(set).String(),
		FieldSelector: fmt.Sprintf("status.phase=%s", state),
	})
	if err != nil {
		return nil, err
	}
	return pods, nil
}

func main() {
	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	k8sClient, err = kubernetes.NewForConfig(config)

	if err != nil {
		panic(err.Error())
	}

	pods, err := GetPods(map[string]string{}, "Running", "default")
	for _, pod := range pods.Items {
		fmt.Println(pod.Name)
	}

}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
