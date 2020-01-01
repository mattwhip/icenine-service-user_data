package main

import (
	"log"

	"github.com/mattwhip/icenine-service-user_data/actions"
	"github.com/mattwhip/icenine-service-user_data/rpc"
)

func main() {

	// Create a channel to read any errors encountered by any server
	serverErrors := make(chan error)

	// Setup buffalo server
	app := actions.App()
	go func() {
		serverErrors <- app.Serve()
	}()

	// Setup GRPC server
	go func() {
		serverErrors <- rpc.Serve()
	}()

	// If any servers crash, kill the app
	err := <-serverErrors
	log.Fatal(err)
}
