package controllers

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pannoiv1alpha1 "kollektor/api/v1alpha1"
	"kollektor/utils"
)

type KollektorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *KollektorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	kollektor := &pannoiv1alpha1.Kollektor{}
	err := r.Get(ctx, req.NamespacedName, kollektor)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Kollektor resource not found. Ignoring since object must be deleted")
			return ctrl.Result{Requeue: false}, nil
		}
	}

	log.Info("Run version scan for: " + kollektor.Name)

	if kollektor.Spec.Source.Repo == "" {
		log.Info("Source repo cannot be nil")
		return ctrl.Result{Requeue: false}, err
	}

	var scrapeIntervalUnitStr, scrapeIntervalTimeStr string
	if os.Getenv("SCRAPE_INTERVAL") != "" {
		for _, char := range os.Getenv("SCRAPE_INTERVAL") {
			if unicode.IsDigit(char) {
				scrapeIntervalTimeStr += string(char)
			} else {
				scrapeIntervalUnitStr += string(char)
			}
		}
	} else {
		scrapeIntervalTimeStr = "1"
		scrapeIntervalUnitStr = "h"
	}

	var scrapeIntervalUnit time.Duration
	scrapeIntervalTime, _ := strconv.Atoi(scrapeIntervalTimeStr)

	switch scrapeIntervalUnitStr {
	case "s":
		scrapeIntervalUnit = time.Second
	case "m":
		scrapeIntervalUnit = time.Minute
	case "h":
		scrapeIntervalUnit = time.Hour
	case "d":
		scrapeIntervalUnit = 24 * time.Hour
	case "w":
		scrapeIntervalUnit = 7 * 24 * time.Hour
	default:
		scrapeIntervalUnit = time.Hour
	}

	ossVersion, err := utils.GetProjectVersion(kollektor.Spec.Source.Repo)
	if err != nil {
		log.Error(err, "Failed to scan version: "+kollektor.Spec.Source.Repo)
		return ctrl.Result{Requeue: true, RequeueAfter: time.Duration(scrapeIntervalTime) * scrapeIntervalUnit}, err
	}

	var chartScan bool
	if kollektor.Spec.Source.ChartRepo != "" {
		chartScan = true
	} else {
		chartScan = false
	}
	var chartVerion string
	if chartScan {
		chartVerion, err = utils.GetHelmChartVersion(kollektor.Spec.Source.ChartRepo)
		if err != nil {
			log.Error(err, "Failed to scan chart version: "+kollektor.Spec.Source.ChartRepo)
			return ctrl.Result{Requeue: true, RequeueAfter: time.Duration(scrapeIntervalTime) * scrapeIntervalUnit}, nil
		}
	}

	log.Info("Version gathered " + kollektor.Name)

	var container []corev1.Container
	var labels map[string]string

	switch strings.ToLower(kollektor.Spec.Resource.Type) {
	case "statefulset":
		sts := appsv1.StatefulSet{}
		err = r.Get(ctx, types.NamespacedName{Namespace: kollektor.Namespace, Name: kollektor.Spec.Resource.Name}, &sts)
		if chartScan {
			labels = sts.Spec.Template.Labels
		}
		container = sts.Spec.Template.Spec.Containers
	case "daemonset":
		ds := appsv1.DaemonSet{}
		err = r.Get(ctx, types.NamespacedName{Namespace: kollektor.Namespace, Name: kollektor.Spec.Resource.Name}, &ds)
		if chartScan {
			labels = ds.Spec.Template.Labels
		}
		container = ds.Spec.Template.Spec.Containers
	case "deployment":
		dep := appsv1.Deployment{}
		err = r.Get(ctx, types.NamespacedName{Namespace: kollektor.Namespace, Name: kollektor.Spec.Resource.Name}, &dep)
		if chartScan {
			labels = dep.Spec.Template.Labels
		}
		container = dep.Spec.Template.Spec.Containers
	case "replicaset":
		rs := appsv1.ReplicaSet{}
		err = r.Get(ctx, types.NamespacedName{Namespace: kollektor.Namespace, Name: kollektor.Spec.Resource.Name}, &rs)
		if chartScan {
			labels = rs.Spec.Template.Labels
		}
		container = rs.Spec.Template.Spec.Containers
	case "pod":
		po := corev1.Pod{}
		err = r.Get(ctx, types.NamespacedName{Namespace: kollektor.Namespace, Name: kollektor.Spec.Resource.Name}, &po)
		if chartScan {
			labels = po.Labels
		}
		container = po.Spec.Containers
	default:
		po := corev1.Pod{}
		err = r.Get(ctx, types.NamespacedName{Namespace: kollektor.Namespace, Name: kollektor.Spec.Resource.Name}, &po)
		if chartScan {
			labels = po.Labels
		}
		container = po.Spec.Containers
	}
	if err != nil {
		log.Error(err, kollektor.Spec.Resource.Type+" not found: "+kollektor.Spec.Resource.Name)
		return ctrl.Result{Requeue: true, RequeueAfter: time.Duration(scrapeIntervalTime) * scrapeIntervalUnit}, err
	}

	var containerName string
	if kollektor.Spec.Resource.ContainerName != "" {
		containerName = kollektor.Spec.Resource.ContainerName
	} else {
		containerName = kollektor.Spec.Resource.Name
	}

	var isLatest bool
	var isChartLatest bool
	var imageVersion string
	var chartLabelVersion string

	if chartScan {
		for key, val := range labels {
			if key == "helm.sh/chart" || key == "chart" {
				chartLabelSplit := strings.Split(val, "-")
				chartLabelVersion = chartLabelSplit[len(chartLabelSplit)-1]
				if chartLabelVersion == chartVerion || "v"+chartLabelVersion == chartVerion {
					isChartLatest = true
				} else {
					isChartLatest = false
					log.Info(kollektor.Spec.Resource.Name + " chart is not matching latest version: " + chartVerion)
				}
				break
			}
		}
	}

	for _, el := range container {
		if el.Name == containerName {
			imageSplit := strings.Split(el.Image, ":")
			imageVersion = imageSplit[len(imageSplit)-1]
			if imageVersion == ossVersion || "v"+imageVersion == ossVersion {
				isLatest = true
			} else {
				isLatest = false
			}
			break
		}
	}

	log.Info((time.Duration(scrapeIntervalTime) * scrapeIntervalUnit).String())

	if isLatest {
		log.Info(kollektor.Spec.Resource.Name + " image is matching latest version: " + ossVersion)
		kollektor.Status.Latest = ossVersion
		kollektor.Status.Current = imageVersion
		kollektor.Status.IsLatest = "True"
		err = r.Status().Update(ctx, kollektor)
		return ctrl.Result{Requeue: true, RequeueAfter: time.Duration(scrapeIntervalTime) * scrapeIntervalUnit}, err
	}
	kollektor.Status.Latest = ossVersion
	kollektor.Status.Current = imageVersion
	kollektor.Status.IsLatest = "False"

	if kollektor.Status.Latest == ossVersion {
		log.Info("No new releases detected for " + kollektor.Name)
		r.Status().Update(ctx, kollektor)
		return ctrl.Result{Requeue: true, RequeueAfter: time.Duration(scrapeIntervalTime) * scrapeIntervalUnit}, nil
	}

	log.Info(kollektor.Spec.Resource.Name + " image is not matching latest version: " + ossVersion)

	slackIntegrationEnabled, _ := strconv.ParseBool(os.Getenv("SLACK_INTEGRATION_ENABLED"))
	if slackIntegrationEnabled {
		releaseNotes, err := utils.GetProjectReleaseNotes(kollektor.Spec.Source.Repo)
		if err != nil {
			log.Error(err, "Failed to gather release notes for "+kollektor.Name)
			return ctrl.Result{Requeue: true, RequeueAfter: time.Duration(scrapeIntervalTime) * scrapeIntervalUnit}, nil
		}
		projectReleaseUrl := kollektor.Spec.Source.Repo + "/releases/latest"
		if !strings.HasPrefix(projectReleaseUrl, "https://") {
			projectReleaseUrl = "https://" + projectReleaseUrl
		}
		if ossVersion != kollektor.Status.Latest {
			projectReleaseTitle := fmt.Sprintf("🚀 <%s|New Release> of %s | %s => %s in cluster: %s!🚀",
				projectReleaseUrl,
				kollektor.Name,
				imageVersion,
				ossVersion,
				os.Getenv("CLUSTER_NAME"),
			)
			projectReleaseText := fmt.Sprintf("Release notes: %s", releaseNotes)
			err = utils.SendSlackMessage(os.Getenv("SLACK_WEBHOOK_URL"), projectReleaseTitle, projectReleaseText)
			if err != nil {
				log.Error(err, "Failed send slack notification")
			}
		}
		if chartScan && !isChartLatest {
			chartReleaseNotes, chartReleaseUrl, err := utils.GetHelmReleaseNotes(kollektor.Spec.Source.ChartRepo)
			if err != nil {
				log.Error(err, "Failed to gather release notes for "+kollektor.Name)
				return ctrl.Result{Requeue: true, RequeueAfter: time.Duration(scrapeIntervalTime) * scrapeIntervalUnit}, nil
			}
			chartReleaseTitle := fmt.Sprintf("📒 <%s|New Helm Chart Release> of %s | %s => %s in cluster: %s!📒",
				chartReleaseUrl,
				kollektor.Name,
				chartVerion,
				chartLabelVersion,
				os.Getenv("CLUSTER_NAME"),
			)
			chartReleaseText := fmt.Sprintf("Release notes: %s", chartReleaseNotes)
			err = utils.SendSlackMessage(os.Getenv("SLACK_WEBHOOK_URL"), chartReleaseTitle, chartReleaseText)
			if err != nil {
				log.Error(err, "Failed send slack notification")
			}
		}
	}

	r.Status().Update(ctx, kollektor)
	return ctrl.Result{Requeue: true, RequeueAfter: time.Duration(scrapeIntervalTime) * scrapeIntervalUnit}, nil
}

func (r *KollektorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pannoiv1alpha1.Kollektor{}).
		Complete(r)
}
