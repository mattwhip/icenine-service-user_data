package grifts

import (
	dao "github.com/mattwhip/icenine-database/user_data"
	"github.com/mattwhip/icenine-service-user_data/models"
	"github.com/markbates/grift/grift"
	"github.com/pkg/errors"
)

var _ = grift.Namespace("db", func() {

	grift.Desc("seed", "Seeds a database")
	grift.Add("seed", func(c *grift.Context) error {
		// Check for existing configuration
		existingConfs := []dao.UdConfig{}
		if err := models.DB.All(&existingConfs); err != nil {
			if err != nil {
				return errors.Wrap(err, "failed to check for existing user accounts configs")
			}
		}
		// Create config if one does not exist
		if len(existingConfs) < 1 {
			conf := &dao.UdConfig{
				InitialCoins:            10000,
				InitialRating:           1500,
				InitialRatingDeviation:  350,
				InitialRatingVolatility: 0.06,
			}
			if err := models.DB.Create(conf); err != nil {
				return errors.Wrap(err, "ffailed to create user accounts config")
			}
		}
		return nil
	})

})
