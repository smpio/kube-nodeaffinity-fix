package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

var (
	minWatchTimeout = 5 * time.Minute
	clientset       *kubernetes.Clientset
)

func main() {
	masterURL := flag.String("master", "", "kubernetes api server url")
	kubeconfigPath := flag.String("kubeconfig", "", "path to kubeconfig file")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags(*masterURL, *kubeconfigPath)
	if err != nil {
		log.Fatalln(err)
	}

	clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalln(err)
	}

	podCh := make(chan *v1.Pod, 128)
	go podWatcher(podCh)

	for pod := range podCh {
		err := clientset.CoreV1().Pods(pod.Namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})
		if err == nil {
			log.Printf("Deleted pod %v/%v", pod.Namespace, pod.Name)
		} else {
			log.Println(err)
		}
	}
}

func podWatcher(c chan *v1.Pod) {
	for {
		err := internalPodWatcher(c)
		if statusErr, ok := err.(*apierrs.StatusError); ok {
			if statusErr.ErrStatus.Reason == metav1.StatusReasonExpired {
				log.Println("podWatcher:", err, "Restarting watch")
				continue
			}
		}

		log.Fatalln(err)
	}
}

func internalPodWatcher(c chan *v1.Pod) error {
	list, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, pod := range list.Items {
		if shouldDelete(&pod) {
			c <- pod.DeepCopy()
		}
	}

	resourceVersion := list.ResourceVersion

	for {
		log.Println("podWatcher: watching since", resourceVersion)

		timeoutSeconds := int64(minWatchTimeout.Seconds() * (rand.Float64() + 1.0))
		watcher, err := clientset.CoreV1().Pods("").Watch(context.TODO(), metav1.ListOptions{
			ResourceVersion: resourceVersion,
			TimeoutSeconds:  &timeoutSeconds,
		})
		if err != nil {
			return err
		}

		for watchEvent := range watcher.ResultChan() {
			if watchEvent.Type == watch.Error {
				return apierrs.FromObject(watchEvent.Object)
			}

			pod, ok := watchEvent.Object.(*v1.Pod)
			if !ok {
				log.Println("podWatcher: unexpected kind:", watchEvent.Object.GetObjectKind().GroupVersionKind())
				continue
			}

			resourceVersion = pod.ResourceVersion

			if watchEvent.Type != watch.Deleted && shouldDelete(pod) {
				c <- pod
			}
		}
	}
}

func shouldDelete(pod *v1.Pod) bool {
	return pod.Status.Phase == v1.PodFailed && pod.Status.Reason == "NodeAffinity"
}
