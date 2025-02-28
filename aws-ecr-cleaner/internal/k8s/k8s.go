package k8s

import (
	"bufio"
	"context"
	"log"
	"os"
	"path/filepath"

	"aws-ecr-cleaner/internal/util"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// FetchInUseImages 从 Kubernetes 集群中获取所有工作负载的镜像，并写入 imageListFile
func FetchInUseImages(imageListFile string) map[string]bool {
	inUse := make(map[string]bool)
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to create in-cluster k8s config: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create k8s client: %v", err)
	}

	imageSet := make(map[string]struct{})

	// Pods
	podList, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err == nil {
		for _, pod := range podList.Items {
			for _, container := range pod.Spec.Containers {
				imageSet[util.TrimRegistry(container.Image)] = struct{}{}
			}
			for _, container := range pod.Spec.InitContainers {
				imageSet[util.TrimRegistry(container.Image)] = struct{}{}
			}
		}
	}

	// Deployments
	deployList, err := clientset.AppsV1().Deployments("").List(context.TODO(), metav1.ListOptions{})
	if err == nil {
		for _, deploy := range deployList.Items {
			for _, container := range deploy.Spec.Template.Spec.Containers {
				imageSet[util.TrimRegistry(container.Image)] = struct{}{}
			}
			for _, container := range deploy.Spec.Template.Spec.InitContainers {
				imageSet[util.TrimRegistry(container.Image)] = struct{}{}
			}
		}
	}

	// StatefulSets
	stsList, err := clientset.AppsV1().StatefulSets("").List(context.TODO(), metav1.ListOptions{})
	if err == nil {
		for _, sts := range stsList.Items {
			for _, container := range sts.Spec.Template.Spec.Containers {
				imageSet[util.TrimRegistry(container.Image)] = struct{}{}
			}
			for _, container := range sts.Spec.Template.Spec.InitContainers {
				imageSet[util.TrimRegistry(container.Image)] = struct{}{}
			}
		}
	}

	// Jobs
	jobsList, err := clientset.BatchV1().Jobs("").List(context.TODO(), metav1.ListOptions{})
	if err == nil {
		for _, job := range jobsList.Items {
			for _, container := range job.Spec.Template.Spec.Containers {
				imageSet[util.TrimRegistry(container.Image)] = struct{}{}
			}
			for _, container := range job.Spec.Template.Spec.InitContainers {
				imageSet[util.TrimRegistry(container.Image)] = struct{}{}
			}
		}
	}

	// DaemonSets
	dsList, err := clientset.AppsV1().DaemonSets("").List(context.TODO(), metav1.ListOptions{})
	if err == nil {
		for _, ds := range dsList.Items {
			for _, container := range ds.Spec.Template.Spec.Containers {
				imageSet[util.TrimRegistry(container.Image)] = struct{}{}
			}
			for _, container := range ds.Spec.Template.Spec.InitContainers {
				imageSet[util.TrimRegistry(container.Image)] = struct{}{}
			}
		}
	}

	// CronJobs（使用 BatchV1 CronJobs）
	cronList, err := clientset.BatchV1().CronJobs("").List(context.TODO(), metav1.ListOptions{})
	if err == nil {
		for _, cj := range cronList.Items {
			for _, container := range cj.Spec.JobTemplate.Spec.Template.Spec.Containers {
				imageSet[util.TrimRegistry(container.Image)] = struct{}{}
			}
			for _, container := range cj.Spec.JobTemplate.Spec.Template.Spec.InitContainers {
				imageSet[util.TrimRegistry(container.Image)] = struct{}{}
			}
		}
	}

	// 将结果写入文件
	if err := os.MkdirAll(filepath.Dir(imageListFile), os.ModePerm); err != nil {
		log.Fatalf("Failed to create directory for image list file: %v", err)
	}
	f, err := os.Create(imageListFile)
	if err != nil {
		log.Fatalf("Failed to create image list file '%s': %v", imageListFile, err)
	}
	defer f.Close()
	writer := bufio.NewWriter(f)
	for image := range imageSet {
		inUse[image] = true
		writer.WriteString(image + "\n")
	}
	writer.Flush()
	log.Printf("Fetched %d unique images from k8s cluster and saved to %s", len(imageSet), imageListFile)
	return inUse
}

// LoadInUseImages 从已有文件中加载 inUse 映射
func LoadInUseImages(imageListFile string) map[string]bool {
	inUse := make(map[string]bool)
	f, err := os.Open(imageListFile)
	if err != nil {
		log.Fatalf("Failed to open image list file '%s': %v", imageListFile, err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			inUse[line] = true
		}
	}
	return inUse
}
