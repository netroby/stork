package migration

import (
	"fmt"

	"github.com/libopenstorage/stork/drivers/volume"
	"github.com/libopenstorage/stork/pkg/migration/controllers"
	"github.com/sirupsen/logrus"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/rest"
)

// Migration migration
type Migration struct {
	Driver                volume.Driver
	clusterPairController *controllers.ClusterPairController
	migrationController   *controllers.MigrationController
}

// Init init
func (m *Migration) Init() error {
	logrus.Infof("Init migration")
	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("Error getting cluster config: %v", err)
	}

	aeclientset, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("Error getting apiextention client, %v", err)
	}

	m.clusterPairController = &controllers.ClusterPairController{
		Driver: m.Driver}
	err = m.clusterPairController.Init(aeclientset)
	if err != nil {
		return fmt.Errorf("error initiliazling clusterpair controller: %v", err)
	}

	m.migrationController = &controllers.MigrationController{
		Driver: m.Driver}
	err = m.migrationController.Init(config, aeclientset)
	if err != nil {
		return fmt.Errorf("error initiliazling clusterpair controller: %v", err)
	}
	return nil
}
