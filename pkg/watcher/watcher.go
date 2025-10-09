package watcher

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type Watcher struct {
	clientset            *kubernetes.Clientset
	labelSelector        string
	restartTimer         time.Duration
	sleepBeforeRestart   time.Duration
	watcherID            int
	secretsListInterval  time.Duration
	enableSecretsListing bool
	jobsListInterval     time.Duration
	enableJobsListing    bool
	namespaceFilterRegex *regexp.Regexp
	enablePodWatcher     bool
	enableJobWatcher     bool
}

func NewWatcher(clientset *kubernetes.Clientset, labelSelector string, restartTimer time.Duration, sleepBeforeRestart time.Duration, watcherID int, secretsListInterval time.Duration, enableSecretsListing bool, jobsListInterval time.Duration, enableJobsListing bool, namespaceFilterRegex string, enablePodWatcher bool, enableJobWatcher bool) *Watcher {
	var compiledRegex *regexp.Regexp
	if namespaceFilterRegex != "" {
		var err error
		compiledRegex, err = regexp.Compile(namespaceFilterRegex)
		if err != nil {
			log.Printf("Watcher %d: Failed to compile namespace filter regex '%s': %v. Ignoring filter.", watcherID, namespaceFilterRegex, err)
			compiledRegex = nil
		}
	}

	return &Watcher{
		clientset:            clientset,
		labelSelector:        labelSelector,
		restartTimer:         restartTimer,
		sleepBeforeRestart:   sleepBeforeRestart,
		watcherID:            watcherID,
		secretsListInterval:  secretsListInterval,
		enableSecretsListing: enableSecretsListing,
		jobsListInterval:     jobsListInterval,
		enableJobsListing:    enableJobsListing,
		namespaceFilterRegex: compiledRegex,
		enablePodWatcher:     enablePodWatcher,
		enableJobWatcher:     enableJobWatcher,
	}
}

func (w *Watcher) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Watcher %d: Stopping due to context cancellation", w.watcherID)
			return
		default:
			log.Printf("Watcher %d: Starting watch cycle", w.watcherID)
			w.runWatchCycle(ctx)

			if ctx.Err() != nil {
				return
			}

			log.Printf("Watcher %d: Sleeping for %v before restart", w.watcherID, w.sleepBeforeRestart)
			select {
			case <-time.After(w.sleepBeforeRestart):
			case <-ctx.Done():
				return
			}
		}
	}
}

func (w *Watcher) runWatchCycle(ctx context.Context) {
	watchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup

	if w.enablePodWatcher {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.watchPods(watchCtx)
		}()
	}

	if w.enableJobWatcher {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.watchJobs(watchCtx)
		}()
	}

	if w.enableSecretsListing {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.listSecrets(watchCtx)
		}()
	}

	if w.enableJobsListing {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.listJobs(watchCtx)
		}()
	}

	select {
	case <-time.After(w.restartTimer):
		log.Printf("Watcher %d: Restart timer expired, gracefully stopping watchers", w.watcherID)
		// Give informers time to complete ongoing operations
		time.Sleep(3 * time.Second)
		cancel()
	case <-ctx.Done():
		log.Printf("Watcher %d: Context cancelled, stopping watchers", w.watcherID)
		cancel()
	}

	wg.Wait()
}

func (w *Watcher) watchPods(ctx context.Context) {
	log.Printf("Watcher %d: Starting pod watcher with label selector: %s", w.watcherID, w.labelSelector)

	tweakListOptions := func(options *metav1.ListOptions) {
		if w.labelSelector != "" {
			options.LabelSelector = w.labelSelector
		}
	}

	factory := informers.NewSharedInformerFactoryWithOptions(
		w.clientset,
		time.Second*30,
		informers.WithNamespace(metav1.NamespaceAll),
		informers.WithTweakListOptions(tweakListOptions),
	)

	podInformer := factory.Core().V1().Pods().Informer()

	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod := obj.(*corev1.Pod)
			log.Printf("Watcher %d: Pod added: %s/%s, Phase: %s", w.watcherID, pod.Namespace, pod.Name, pod.Status.Phase)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldPod := oldObj.(*corev1.Pod)
			newPod := newObj.(*corev1.Pod)
			if oldPod.Status.Phase != newPod.Status.Phase {
				log.Printf("Watcher %d: Pod updated: %s/%s, Phase: %s -> %s", w.watcherID, newPod.Namespace, newPod.Name, oldPod.Status.Phase, newPod.Status.Phase)
			}
		},
		DeleteFunc: func(obj interface{}) {
			var pod *corev1.Pod
			switch t := obj.(type) {
			case *corev1.Pod:
				pod = t
			case cache.DeletedFinalStateUnknown:
				pod, _ = t.Obj.(*corev1.Pod)
			}
			if pod != nil {
				log.Printf("Watcher %d: Pod deleted: %s/%s", w.watcherID, pod.Namespace, pod.Name)
			}
		},
	})

	go factory.Start(ctx.Done())

	syncCtx, syncCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer syncCancel()

	if !cache.WaitForCacheSync(syncCtx.Done(), podInformer.HasSynced) {
		log.Printf("Watcher %d: Failed to sync pod cache within 5 minutes - cluster may be overloaded", w.watcherID)
		return
	}

	<-ctx.Done()
	log.Printf("Watcher %d: Pod watcher stopped", w.watcherID)
}

func (w *Watcher) watchJobs(ctx context.Context) {
	log.Printf("Watcher %d: Starting job watcher with label selector: %s", w.watcherID, w.labelSelector)

	tweakListOptions := func(options *metav1.ListOptions) {
		if w.labelSelector != "" {
			options.LabelSelector = w.labelSelector
		}
	}

	factory := informers.NewSharedInformerFactoryWithOptions(
		w.clientset,
		time.Second*30,
		informers.WithNamespace(metav1.NamespaceAll),
		informers.WithTweakListOptions(tweakListOptions),
	)

	jobInformer := factory.Batch().V1().Jobs().Informer()

	jobInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			job := obj.(*batchv1.Job)
			log.Printf("Watcher %d: Job added: %s/%s", w.watcherID, job.Namespace, job.Name)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldJob := oldObj.(*batchv1.Job)
			newJob := newObj.(*batchv1.Job)

			oldConditions := getJobConditions(oldJob)
			newConditions := getJobConditions(newJob)

			if oldConditions != newConditions {
				log.Printf("Watcher %d: Job updated: %s/%s, Conditions: %s", w.watcherID, newJob.Namespace, newJob.Name, newConditions)
			}
		},
		DeleteFunc: func(obj interface{}) {
			var job *batchv1.Job
			switch t := obj.(type) {
			case *batchv1.Job:
				job = t
			case cache.DeletedFinalStateUnknown:
				job, _ = t.Obj.(*batchv1.Job)
			}
			if job != nil {
				log.Printf("Watcher %d: Job deleted: %s/%s", w.watcherID, job.Namespace, job.Name)
			}
		},
	})

	go factory.Start(ctx.Done())

	syncCtx, syncCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer syncCancel()

	if !cache.WaitForCacheSync(syncCtx.Done(), jobInformer.HasSynced) {
		log.Printf("Watcher %d: Failed to sync job cache within 5 minutes - cluster may be overloaded", w.watcherID)
		return
	}

	<-ctx.Done()
	log.Printf("Watcher %d: Job watcher stopped", w.watcherID)
}

func getJobConditions(job *batchv1.Job) string {
	conditions := []string{}
	for _, condition := range job.Status.Conditions {
		if condition.Status == corev1.ConditionTrue {
			conditions = append(conditions, string(condition.Type))
		}
	}
	if len(conditions) == 0 {
		return "Running"
	}
	return fmt.Sprintf("%v", conditions)
}

func (w *Watcher) listSecrets(ctx context.Context) {
	filterMsg := "all namespaces"
	if w.namespaceFilterRegex != nil {
		filterMsg = fmt.Sprintf("namespaces matching '%s'", w.namespaceFilterRegex.String())
	}
	log.Printf("Watcher %d: Starting secrets listing with interval: %v, filter: %s", w.watcherID, w.secretsListInterval, filterMsg)

	ticker := time.NewTicker(w.secretsListInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Watcher %d: Secrets lister stopped", w.watcherID)
			return
		case <-ticker.C:
			start := time.Now()
			totalSecrets := 0

			if w.namespaceFilterRegex == nil {
				// List across all namespaces
				secrets, err := w.clientset.CoreV1().Secrets(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
				duration := time.Since(start)

				if err != nil {
					log.Printf("Watcher %d: Failed to list secrets: %v (took %v)", w.watcherID, err, duration)
				} else {
					log.Printf("Watcher %d: Listed %d secrets across all namespaces (took %v)", w.watcherID, len(secrets.Items), duration)
				}
			} else {
				// List namespaces first, filter by regex, then list secrets in matching namespaces
				namespaces, err := w.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
				if err != nil {
					duration := time.Since(start)
					log.Printf("Watcher %d: Failed to list namespaces: %v (took %v)", w.watcherID, err, duration)
					continue
				}

				matchedNamespaces := 0
				for _, ns := range namespaces.Items {
					if w.namespaceFilterRegex.MatchString(ns.Name) {
						matchedNamespaces++
						secrets, err := w.clientset.CoreV1().Secrets(ns.Name).List(ctx, metav1.ListOptions{})
						if err != nil {
							log.Printf("Watcher %d: Failed to list secrets in namespace %s: %v", w.watcherID, ns.Name, err)
						} else {
							totalSecrets += len(secrets.Items)
						}
					}
				}
				duration := time.Since(start)
				log.Printf("Watcher %d: Listed %d secrets across %d namespaces matching '%s' (took %v)", w.watcherID, totalSecrets, matchedNamespaces, w.namespaceFilterRegex.String(), duration)
			}
		}
	}
}

func (w *Watcher) listJobs(ctx context.Context) {
	filterMsg := "all namespaces"
	if w.namespaceFilterRegex != nil {
		filterMsg = fmt.Sprintf("namespaces matching '%s'", w.namespaceFilterRegex.String())
	}
	log.Printf("Watcher %d: Starting jobs listing with interval: %v, filter: %s", w.watcherID, w.jobsListInterval, filterMsg)

	ticker := time.NewTicker(w.jobsListInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Watcher %d: Jobs lister stopped", w.watcherID)
			return
		case <-ticker.C:
			start := time.Now()
			totalJobs := 0

			if w.namespaceFilterRegex == nil {
				// List across all namespaces
				jobs, err := w.clientset.BatchV1().Jobs(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
				duration := time.Since(start)

				if err != nil {
					log.Printf("Watcher %d: Failed to list jobs: %v (took %v)", w.watcherID, err, duration)
				} else {
					log.Printf("Watcher %d: Listed %d jobs across all namespaces (took %v)", w.watcherID, len(jobs.Items), duration)
				}
			} else {
				// List namespaces first, filter by regex, then list jobs in matching namespaces
				namespaces, err := w.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
				if err != nil {
					duration := time.Since(start)
					log.Printf("Watcher %d: Failed to list namespaces: %v (took %v)", w.watcherID, err, duration)
					continue
				}

				matchedNamespaces := 0
				for _, ns := range namespaces.Items {
					if w.namespaceFilterRegex.MatchString(ns.Name) {
						matchedNamespaces++
						jobs, err := w.clientset.BatchV1().Jobs(ns.Name).List(ctx, metav1.ListOptions{})
						if err != nil {
							log.Printf("Watcher %d: Failed to list jobs in namespace %s: %v", w.watcherID, ns.Name, err)
						} else {
							totalJobs += len(jobs.Items)
						}
					}
				}
				duration := time.Since(start)
				log.Printf("Watcher %d: Listed %d jobs across %d namespaces matching '%s' (took %v)", w.watcherID, totalJobs, matchedNamespaces, w.namespaceFilterRegex.String(), duration)
			}
		}
	}
}
