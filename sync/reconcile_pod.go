package sync

import (
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/rancher/netes-agent/labels"
	"github.com/rancher/netes-agent/watch"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
)

// TODO: return erros
func reconcilePod(clientset *kubernetes.Clientset, watchClient *watch.Client, pod v1.Pod) (v1.Pod, error) {
	revision := pod.Labels[labels.RevisionLabel]
	existingPod, ok := watchClient.GetPod(pod.Name)
	if ok {
		if existingRevision, ok := existingPod.Labels[labels.RevisionLabel]; ok {
			if revision != existingRevision {
				log.Infof("Pod %s has old revision", pod.Name)
				if err := deletePod(clientset, pod); err != nil {
					log.Error(err)
				}
			}
		}
	} else {
		if err := createPod(clientset, pod); err != nil {
			log.Error(err)
		}
	}

	for {
		if existingPod, ok := watchClient.GetPod(pod.Name); ok && existingPod.Spec.NodeName != "" {
			allContainersReady := true
			for _, containerStatus := range existingPod.Status.ContainerStatuses {
				if !containerStatus.Ready {
					allContainersReady = false
					break
				}
			}
			if allContainersReady {
				return existingPod, nil
			}
		}

		log.Infof("Waiting for containers of pod %s to be ready", pod.Name)
		time.Sleep(time.Second)
	}
}

func createPod(clientset *kubernetes.Clientset, pod v1.Pod) error {
	log.Infof("Creating pod %s", pod.Name)
	_, err := clientset.Pods(v1.NamespaceDefault).Create(&pod)
	return err
}

func deletePod(clientset *kubernetes.Clientset, pod v1.Pod) error {
	log.Infof("Deleting pod %s", pod.Name)
	return clientset.Pods(v1.NamespaceDefault).Delete(pod.Name, &metav1.DeleteOptions{})
}