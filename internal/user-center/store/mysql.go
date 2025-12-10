package store

import (
	"fmt"
	"sync"

	"gorm.io/gorm"

	"github.com/kart-io/logger"
	"github.com/kart-io/sentinel-x/pkg/component/mysql"
	"github.com/kart-io/sentinel-x/pkg/infra/datasource"
)

var (
	clientFactory Factory
	once          sync.Once
)

// datastore implements the Factory interface.
type datastore struct {
	db *gorm.DB
}

// GetFactory returns the storage factory.
func GetFactory(dsManager *datasource.Manager) (Factory, error) {
	var err error
	var db *gorm.DB

	once.Do(func() {
		// Use "primary" database by default
		var client *mysql.Client
		client, err = dsManager.GetMySQL("primary")
		if err != nil {
			logger.Errorf("failed to get mysql connection: %s", err.Error())
			return
		}
		db = client.DB()

		clientFactory = &datastore{db}
	})

	if clientFactory == nil || err != nil {
		return nil, fmt.Errorf("failed to get mysql factory: %w", err)
	}

	return clientFactory, nil
}

// Users returns the user store.
func (ds *datastore) Users() UserStore {
	return newUsers(ds.db)
}

// Close closes the factory and underlying connections.
func (ds *datastore) Close() error {
	return nil
}
