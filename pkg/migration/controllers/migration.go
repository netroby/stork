package controllers

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/heptio/ark/pkg/backup"
	"github.com/heptio/ark/pkg/discovery"
	"github.com/heptio/ark/pkg/util/collections"
	"github.com/libopenstorage/stork/drivers/volume"
	stork "github.com/libopenstorage/stork/pkg/apis/stork"
	stork_crd "github.com/libopenstorage/stork/pkg/apis/stork/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/portworx/sched-ops/k8s"
	"github.com/sirupsen/logrus"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	resyncPeriod           = 30
	storkMigrationReplicas = "stork.openstorage.org/migrationReplicas"
)

// MigrationController migrationcontroller
type MigrationController struct {
	Driver            volume.Driver
	backupper         backup.Backupper
	discoveryHelper   discovery.Helper
	dynamicClientPool dynamic.ClientPool
}

// Init init
func (m *MigrationController) Init(config *rest.Config, client apiextensionsclient.Interface) error {
	err := m.createCRD(client)
	if err != nil {
		return err
	}

	discoveryClient := client.Discovery()
	m.discoveryHelper, err = discovery.NewHelper(discoveryClient, logrus.New())
	if err != nil {
		return err
	}
	err = m.discoveryHelper.Refresh()
	if err != nil {
		return err
	}
	m.dynamicClientPool = dynamic.NewDynamicClientPool(config)

	sdk.Watch(stork_crd.SchemeGroupVersion.String(), reflect.TypeOf(stork_crd.Migration{}).Name(), "", resyncPeriod)
	return nil
}

// Handle updates for Migration objects
func (m *MigrationController) Handle(ctx context.Context, event sdk.Event) error {
	switch o := event.Object.(type) {
	case *stork_crd.Migration:
		//logrus.Debugf("Update for migration %v Deleted: %v", o, event.Deleted)
		migration := o
		if event.Deleted {
			return m.Driver.CancelMigration(migration)
		}

		if migration.Spec.ClusterPair == "" {
			return fmt.Errorf("clusterPair to migrate to cannot be empty")
		}

		if migration.Status.Stage == "" {
			migration.Status = stork_crd.MigrationStatus{
				Stage:  stork_crd.MigrationStageInitializing,
				Status: stork_crd.MigrationStatusPending,
			}
		}
		switch migration.Status.Stage {

		case stork_crd.MigrationStageInitializing,
			stork_crd.MigrationStageVolumes:
			migration.Status.Stage = stork_crd.MigrationStageVolumes
			// Trigger the migration if we don't have any status
			if migration.Status.Volumes == nil {
				volumeInfos, err := m.Driver.StartMigration(migration)
				if err != nil {
					return err
				}
				if volumeInfos == nil {
					volumeInfos = make([]*stork_crd.VolumeInfo, 0)
				}
				migration.Status.Volumes = volumeInfos
				err = sdk.Update(migration)
				if err != nil {
					return err
				}
			}

			// Now check the status
			volumeInfos, err := m.Driver.GetMigrationStatus(migration)
			if err != nil {
				return err
			}
			if volumeInfos == nil {
				volumeInfos = make([]*stork_crd.VolumeInfo, 0)
			}
			migration.Status.Volumes = volumeInfos
			// Store the new status
			err = sdk.Update(migration)
			if err != nil {
				return err
			}

			// Now check if there is any failure or success
			// TODO: On failure of one volume cancel other migrations?
			for _, vInfo := range volumeInfos {
				// Return if we have any volume migrations still in progress
				if vInfo.Status == stork_crd.MigrationStatusInProgress {
					logrus.Infof("Volume Migration still in progress: %v", migration.Name)
					return nil
				} else if vInfo.Status == stork_crd.MigrationStatusFailed {
					migration.Status.Stage = stork_crd.MigrationStageFinal
					migration.Status.Status = stork_crd.MigrationStatusFailed
				}
			}

			// If the migration hasn't failed move on to the next stage.
			if migration.Status.Status != stork_crd.MigrationStatusFailed {
				if migration.Spec.IncludeResources {
					migration.Status.Stage = stork_crd.MigrationStageApplications
					migration.Status.Status = stork_crd.MigrationStatusInProgress
					// Update the current state and then move on to migrating
					// resources
					err = sdk.Update(migration)
					if err != nil {
						return err
					}
					err = m.migrateResources(migration)
					if err != nil {
						logrus.Errorf("Error migrating resources: %v", err)
						return err
					}
				}
				migration.Status.Stage = stork_crd.MigrationStageFinal
				migration.Status.Status = stork_crd.MigrationStatusSuccessful
			}
			err = sdk.Update(migration)
			if err != nil {
				return err
			}
		case stork_crd.MigrationStageApplications:
			err := m.migrateResources(migration)
			if err != nil {
				logrus.Errorf("Error migrating resources: %v", err)
				return err
			}

		case stork_crd.MigrationStageFinal:
			// Do Nothing
			return nil
		default:
			logrus.Errorf("Invalid stage for migration: %v", migration.Status.Stage)
		}
	}
	return nil
}

func resourceToBeMigrated(migration *stork_crd.Migration, resource metav1.APIResource) bool {
	if resource.Group == "extensions" && resource.Kind == "Deployment" {
		return false
	}

	switch resource.Kind {
	case "PersistentVolumeClaim",
		"PersistentVolume",
		"Deployment",
		"StatefulSet",
		"ConfigMap",
		"Service",
		"Secret":
		return true
	default:
		return false
	}
}

func objectToBeMigrated(object runtime.Unstructured) (bool, error) {
	objectType, err := meta.TypeAccessor(object)
	if err != nil {
		return false, err
	}

	// Don't migrate the kubernetes service
	if objectType.GetKind() == "Service" {

		metadata, err := meta.Accessor(object)
		if err != nil {
			return false, err
		}
		if metadata.GetName() == "kubernetes" {
			return false, nil
		}
	}

	return true, nil
}

func (m *MigrationController) migrateResources(migration *stork_crd.Migration) error {
	err := m.discoveryHelper.Refresh()
	if err != nil {
		return err
	}
	allObjects := make([]runtime.Unstructured, 0)
	resourceInfos := make([]*stork_crd.ResourceInfo, 0)

	for _, group := range m.discoveryHelper.Resources() {
		groupVersion, err := schema.ParseGroupVersion(group.GroupVersion)
		if err != nil {
			return err
		}
		if groupVersion.Group == "extensions" {
			continue
		}
		for _, resource := range group.APIResources {
			if !resourceToBeMigrated(migration, resource) {
				continue
			}

			for _, ns := range migration.Spec.Namespaces {
				dynamicClient, err := m.dynamicClientPool.ClientForGroupVersionKind(groupVersion.WithKind(""))
				if err != nil {
					return err
				}
				client := dynamicClient.Resource(&resource, ns)

				objectsList, err := client.List(metav1.ListOptions{})
				if err != nil {
					return err
				}
				objects, err := meta.ExtractList(objectsList)
				if err != nil {
					return err
				}
				for _, o := range objects {
					runtimeObject, ok := o.(runtime.Unstructured)
					if !ok {
						return fmt.Errorf("Error casting object: %v", o)
					}

					migrate, err := objectToBeMigrated(runtimeObject)
					if err != nil {
						return fmt.Errorf("Error processing object %v: %v", runtimeObject, err)
					}
					if !migrate {
						continue
					}
					metadata, err := meta.Accessor(runtimeObject)
					if err != nil {
						return err
					}
					resourceInfo := &stork_crd.ResourceInfo{
						Name:      metadata.GetName(),
						Namespace: metadata.GetNamespace(),
						Status:    stork_crd.MigrationStatusInProgress,
					}
					resourceInfo.Kind = resource.Kind
					resourceInfo.Group = groupVersion.Group
					if resourceInfo.Group == "" {
						resourceInfo.Group = "core"
					}
					resourceInfo.Version = groupVersion.Version
					resourceInfos = append(resourceInfos, resourceInfo)
					allObjects = append(allObjects, runtimeObject)
				}
			}
		}
		migration.Status.Resources = resourceInfos
		err = sdk.Update(migration)
		if err != nil {
			return err
		}
	}
	err = m.prepareResources(migration, allObjects)
	if err != nil {
		logrus.Errorf("Error preparing resources: %v", err)
		return err
	}
	err = m.applyResources(migration, allObjects)
	if err != nil {
		logrus.Errorf("Error applying resources: %v", err)
		return err
	}
	migration.Status.Stage = stork_crd.MigrationStageFinal
	migration.Status.Status = stork_crd.MigrationStatusSuccessful
	err = sdk.Update(migration)
	if err != nil {
		return err
	}
	return nil
}

func (m *MigrationController) prepareResources(
	migration *stork_crd.Migration,
	objects []runtime.Unstructured,
) error {
	for _, o := range objects {
		content := o.UnstructuredContent()
		delete(content, "status")
		switch o.GetObjectKind().GroupVersionKind().Kind {
		case "PersistentVolume":
			updatedObject, err := m.preparePVResource(migration, o)
			if err != nil {
				m.updateResourceStatus(
					migration,
					o,
					stork_crd.MigrationStatusFailed,
					fmt.Sprintf("Error preparing PV resource: %v", err))
				continue
			}
			o = updatedObject
		case "Deployment", "StatefulSet":
			updatedObject, err := m.prepareApplicationResource(migration, o)
			if err != nil {
				m.updateResourceStatus(
					migration,
					o,
					stork_crd.MigrationStatusFailed,
					fmt.Sprintf("Error preparing Application resource: %v", err))
				continue
			}
			o = updatedObject
		}
		metadata, err := collections.GetMap(content, "metadata")
		if err != nil {
			m.updateResourceStatus(
				migration,
				o,
				stork_crd.MigrationStatusFailed,
				fmt.Sprintf("Error getting metadata for resource: %v", err))
			continue
		}
		for key := range metadata {
			switch key {
			case "name", "namespace", "labels", "annotations":
			default:
				delete(metadata, key)
			}
		}
	}
	return nil
}

func (m *MigrationController) updateResourceStatus(
	migration *stork_crd.Migration,
	object runtime.Unstructured,
	status stork_crd.MigrationStatusType,
	errorMessage string,
) {

}

func (m *MigrationController) preparePVResource(
	migration *stork_crd.Migration,
	object runtime.Unstructured,
) (runtime.Unstructured, error) {
	spec, err := collections.GetMap(object.UnstructuredContent(), "spec")
	if err != nil {
		return nil, err
	}
	delete(spec, "claimRef")
	delete(spec, "storageClassName")

	portworxSpec, err := collections.GetMap(object.UnstructuredContent(), "spec.portworxVolume")
	if err != nil {
		return nil, err
	}

	metadata, err := meta.Accessor(object)
	if err != nil {
		return nil, err
	}

	portworxSpec["volumeID"] = metadata.GetName()
	return object, nil
}

func (m *MigrationController) prepareApplicationResource(
	migration *stork_crd.Migration,
	object runtime.Unstructured,
) (runtime.Unstructured, error) {
	if migration.Spec.StartApplications {
		return object, nil
	}

	// Reset the replicas to 0 and store the current replicas in an annotation
	content := object.UnstructuredContent()
	spec, err := collections.GetMap(content, "spec")
	if err != nil {
		return nil, err
	}
	replicas := spec["replicas"].(int64)
	annotations, err := collections.GetMap(content, "metadata.annotations")
	annotations[storkMigrationReplicas] = strconv.FormatInt(replicas, 10)
	spec["replicas"] = 0
	return object, nil
}

func (m *MigrationController) applyResources(
	migration *stork_crd.Migration,
	objects []runtime.Unstructured,
) error {
	clusterPair, err := k8s.Instance().GetClusterPair(migration.Spec.ClusterPair)
	if err != nil {
		return fmt.Errorf("error getting clusterpair: %v", err)
	}
	remoteClientConfig := clientcmd.NewNonInteractiveClientConfig(
		clusterPair.Spec.Config,
		clusterPair.Spec.Config.CurrentContext,
		&clientcmd.ConfigOverrides{},
		nil)
	remoteConfig, err := remoteClientConfig.ClientConfig()
	if err != nil {
		return err
	}

	remoteDynamicClientPool := dynamic.NewDynamicClientPool(remoteConfig)
	for _, o := range objects {
		dynamicClient, err := remoteDynamicClientPool.ClientForGroupVersionKind(o.GetObjectKind().GroupVersionKind())
		if err != nil {
			return err
		}
		metadata, err := meta.Accessor(o)
		if err != nil {
			return err
		}
		objectType, err := meta.TypeAccessor(o)
		if err != nil {
			return err
		}
		logrus.Infof("Applying %v %v", objectType.GetKind(), metadata.GetName())
		resource := &metav1.APIResource{
			Name:       strings.ToLower(objectType.GetKind()) + "s",
			Namespaced: len(metadata.GetNamespace()) > 0,
		}
		client := dynamicClient.Resource(resource, metadata.GetNamespace())
		unstructured, ok := o.(*unstructured.Unstructured)
		if !ok {
			return fmt.Errorf("Unable to cast object to unstructured: %v", o)
		}
		_, err = client.Create(unstructured)
		if err != nil && apierrors.IsAlreadyExists(err) {
			switch objectType.GetKind() {
			// Don't want to delete the Volume resources
			case "PersistentVolumeClaim", "PersistentVolume":
				break
			default:
				err = client.Delete(metadata.GetName(), &metav1.DeleteOptions{})
				_, err = client.Create(unstructured)
				if err != nil {
					m.updateResourceStatus(
						migration,
						o,
						stork_crd.MigrationStatusFailed,
						fmt.Sprintf("Failed to migrate resource: %v", err))
					continue
				}
			}

		}
		m.updateResourceStatus(
			migration,
			o,
			stork_crd.MigrationStatusSuccessful,
			"Resource migrated successfully")
	}
	return nil
}

func (m *MigrationController) createCRD(client apiextensionsclient.Interface) error {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: stork_crd.StorkMigrationResourcePlural + "." + stork.GroupName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   stork.GroupName,
			Version: stork.Version,
			Scope:   apiextensionsv1beta1.NamespaceScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: stork_crd.StorkMigrationResourcePlural,
				Kind:   reflect.TypeOf(stork_crd.Migration{}).Name(),
			},
		},
	}
	_, err := client.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)

	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}
