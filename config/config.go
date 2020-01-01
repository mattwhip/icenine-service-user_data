package config

import (
	"sync"

	dao "github.com/mattwhip/icenine-database/user_data"
	"github.com/mattwhip/icenine-service-user_data/models"
	userData "github.com/mattwhip/icenine-services/user_data"
	"github.com/pkg/errors"
)

// Get retrieves the default config
func Get() (userData.Config, error) {
	if err := lazyInit(); err != nil {
		return userData.Config{}, errors.Wrap(err, "failed to lazy init")
	}
	return cachedConfig, nil
}

var cachedConfig userData.Config
var initialized bool
var configMutex *sync.Mutex

func init() {
	initialized = false
	configMutex = &sync.Mutex{}
}

func loadConfig() error {
	// Load config from database
	c := &dao.UdConfig{}
	if err := models.DB.First(c); err != nil {
		return errors.Wrap(err, "failed to load UdConfig from database")
	}
	// Create config
	cachedConfig = userData.Config{
		InitialCoins:            c.InitialCoins,
		InitialRating:           c.InitialRating,
		InitialRatingDeviation:  c.InitialRatingDeviation,
		InitialRatingVolatility: c.InitialRatingVolatility,
	}
	return nil
}

func lazyInit() error {
	if !initialized {
		configMutex.Lock()
		defer configMutex.Unlock()
		if !initialized {
			// Load config from database
			if err := loadConfig(); err != nil {
				return errors.Wrap(err, "failed to load initial config")
			}
			// Flag initialized
			initialized = true
		}
	}
	return nil
}
