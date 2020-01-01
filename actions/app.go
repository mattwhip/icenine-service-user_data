package actions

import (
	"os"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/buffalo-pop/pop/popmw"
	"github.com/gobuffalo/envy"
	forcessl "github.com/gobuffalo/mw-forcessl"
	paramlogger "github.com/gobuffalo/mw-paramlogger"
	"github.com/unrolled/secure"

	"github.com/gobuffalo/x/sessions"

	"github.com/mattwhip/icenine-service-user_data/models"
	myMiddleware "github.com/mattwhip/icenine-services/middleware"
)

// ENV is used to help switch settings based on where the
// application is being run. Default is "development".
var ENV = envy.Get("GO_ENV", "development")
var app *buffalo.App

var jwtSigningSecret = os.Getenv("JWT_SIGNING_SECRET")

// App is where all routes and middleware for buffalo
// should be defined. This is the nerve center of your
// application.
func App() *buffalo.App {
	if app == nil {
		app = buffalo.New(buffalo.Options{
			Env:          ENV,
			SessionStore: sessions.Null{},
			SessionName:  "_userdata_session",
		})
		app.Use(forcessl.Middleware(secure.Options{
			SSLRedirect:     ENV == "production",
			SSLProxyHeaders: map[string]string{"X-Forwarded-Proto": "https"},
		}))
		if ENV == "development" {
			app.Use(paramlogger.ParameterLogger)
		}

		// Add JWT verification to entire app
		app.Use(myMiddleware.JWTVerification(jwtSigningSecret))

		// Wrapp all handlers in a DB transaction
		app.Use(popmw.Transaction(models.DB))
	}

	return app
}
