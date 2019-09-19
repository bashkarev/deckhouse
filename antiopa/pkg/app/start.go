package app

import (
	"os"
	"regexp"
	"time"

	"github.com/romana/rlog"

	addon_operator "github.com/flant/addon-operator/pkg/addon-operator"

	"github.com/deckhouse/deckhouse/antiopa/pkg/docker_registry_manager"
)

// Start runs registry watcher and start addon_operator
func Start() {
	rlog.Debug("ANTIOPA: Start")

	// BeforeHelmInitCb is called when kube client is initialized and metrics storage is started
	addon_operator.BeforeHelmInitCb = func() {
		if FeatureWatchRegistry == "yes" {
			err := StartWatchRegistry()
			if err != nil {
				rlog.Errorf("Cannot start watch registry: %s", err)
				os.Exit(1)
			}
		} else {
			rlog.Debugf("Antiopa: registry manager disabled with ANTIOPA_WATCH_REGISTRY=%s.", FeatureWatchRegistry)
		}
	}

	addon_operator.Start()
}

// StartWatchRegistry initializes and starts a RegistryManager.
func StartWatchRegistry() error {
	LastSuccessTime := time.Now()
	RegistryManager := docker_registry_manager.NewDockerRegistryManager()
	RegistryManager.WithRegistrySecretPath(RegistrySecretPath)
	RegistryManager.WithErrorCallback(func() {
		addon_operator.MetricsStorage.SendCounterMetric("antiopa_registry_errors", 1.0, map[string]string{})
		nowTime := time.Now()
		if LastSuccessTime.Add(RegistryErrorsMaxTimeBeforeRestart).Before(nowTime) {
			rlog.Errorf("No success response from registry during %s. Forced restart.", RegistryErrorsMaxTimeBeforeRestart.String())
			os.Exit(1)
		}
		return
	})
	RegistryManager.WithSuccessCallback(func() {
		LastSuccessTime = time.Now()
	})
	RegistryManager.WithImageInfoCallback(GetCurrentPodImageInfo)
	RegistryManager.WithImageUpdatedCallback(UpdateDeploymentImage)

	err := RegistryManager.Init()
	if err != nil {
		rlog.Errorf("MAIN Fatal: Cannot initialize registry manager: %s", err)
		return err
	}
	go RegistryManager.Run()

	return nil
}

// UpdateDeploymentImage updates "antiopaImageId" label of deployment/antiopa
func UpdateDeploymentImage(newImageId string) {
	deployment, err := GetDeploymentOfCurrentPod()
	if err != nil {
		rlog.Errorf("KUBE get current deployment: %s", err)
		return
	}

	deployment.Spec.Template.Labels["antiopaImageId"] = NormalizeLabelValue(newImageId)

	err = UpdateDeployment(deployment)
	if err != nil {
		rlog.Errorf("KUBE deployment update error: %s", err)
		return
	}

	rlog.Infof("KUBE deployment update successful, exiting ...")
	os.Exit(1)
}

var NonSafeCharsRegexp = regexp.MustCompile(`[^a-zA-Z0-9]`)

func NormalizeLabelValue(value string) string {
	newVal := NonSafeCharsRegexp.ReplaceAllLiteralString(value, "_")
	labelLen := len(newVal)
	if labelLen > 63 {
		labelLen = 63
	}
	return newVal[:labelLen]
}

// GetCurrentPodImageInfo returns image name (registry:port/image_repo:image_tag) and imageID.
//
// imageID can be in two forms:
// - "imageID": "docker-pullable://registry.flant.com/sys/antiopa/dev@sha256:05f5cc14dff4fcc3ff3eb554de0e550050e65c968dc8bbc2d7f4506edfcdc5b6"
// - "imageID": "docker://sha256:e537460dd124f6db6656c1728a42cf8e268923ff52575504a471fa485c2a884a"
func GetCurrentPodImageInfo() (imageName string, imageId string) {
	res, err := GetCurrentPod()
	if err != nil {
		rlog.Debugf("KUBE Get current pod info: %v", err)
		return "", ""
	}

	// Get image name from container spec. ContainerStatus contains bad name
	// if multiple tags has one digest!
	// https://github.com/kubernetes/kubernetes/issues/51017
	for _, spec := range res.Spec.Containers {
		if spec.Name == ContainerName {
			imageName = spec.Image
			break
		}
	}

	for _, status := range res.Status.ContainerStatuses {
		if status.Name == ContainerName {
			imageId = status.ImageID
			break
		}
	}

	return
}
