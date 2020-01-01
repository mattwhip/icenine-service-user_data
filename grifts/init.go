package grifts

import (
	"github.com/mattwhip/icenine-service-user_data/actions"
	"github.com/gobuffalo/buffalo"
)

func init() {
	buffalo.Grifts(actions.App())
}
