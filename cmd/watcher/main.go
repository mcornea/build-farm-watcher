package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"k8s-watcher/pkg/config"
	"k8s-watcher/pkg/watcher"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	cfg := config.Load()

	log.Printf("Starting Kubernetes watcher with config:")
	log.Printf("  RestartInterval: %v", cfg.RestartInterval)
	log.Printf("  SleepBeforeRestart: %v", cfg.SleepBeforeRestart)
	log.Printf("  NumWatchers: %d", cfg.NumWatchers)
	log.Printf("  LabelSelector: %s", cfg.LabelSelector)
	log.Printf("  LogLevel: %s", cfg.LogLevel)
	log.Printf("  EnableSecretsListing: %v", cfg.EnableSecretsListing)
	log.Printf("  SecretsListInterval: %v", cfg.SecretsListInterval)

	clientset, err := createKubernetesClient()
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	for i := 0; i < cfg.NumWatchers; i++ {
		watcherLabel := fmt.Sprintf("watcher%d", i+1)
		labelSelector := fmt.Sprintf("build-farm-watcher=%s", watcherLabel)
		w := watcher.NewWatcher(clientset, labelSelector, cfg.RestartInterval, cfg.SleepBeforeRestart, i+1, cfg.SecretsListInterval, cfg.EnableSecretsListing)
		wg.Add(1)
		go w.Start(ctx, &wg)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("Started %d watchers. Press Ctrl+C to stop.", cfg.NumWatchers)

	<-sigChan
	log.Println("Received shutdown signal, stopping watchers...")
	cancel()

	wg.Wait()
	log.Println("All watchers stopped. Exiting.")
}

func createKubernetesClient() (*kubernetes.Clientset, error) {
	var config *rest.Config
	var err error

	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}

	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}