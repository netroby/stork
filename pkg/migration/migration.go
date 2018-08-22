package migration

import (
	"fmt"

	"github.com/libopenstorage/stork/drivers/volume"
	"github.com/libopenstorage/stork/pkg/migration/controllers"
)

// Migration migration
type Migration struct {
	Driver                volume.Driver
	clusterPairController *controllers.ClusterPairController
	migrationController   *controllers.MigrationController
}

// Init init
func (m *Migration) Init() error {
	m.clusterPairController = &controllers.ClusterPairController{
		Driver: m.Driver}
	err := m.clusterPairController.Init()
	if err != nil {
		return fmt.Errorf("error initiliazling clusterpair controller: %v", err)
	}

	m.migrationController = &controllers.MigrationController{
		Driver: m.Driver}
	err = m.migrationController.Init()
	if err != nil {
		return fmt.Errorf("error initiliazling clusterpair controller: %v", err)
	}
	return nil
}
