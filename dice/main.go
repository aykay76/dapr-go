package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"

	"github.com/dapr/go-sdk/actor"
	dapr "github.com/dapr/go-sdk/client"
	daprd "github.com/dapr/go-sdk/service/http"
)

type SimulationActor struct {
	actor.ServerImplBaseCtx
	daprClient dapr.Client
}

func simulationActorFactory() actor.ServerContext {
	return &SimulationActor{
		daprClient: client,
	}
}

func (t *SimulationActor) Type() string {
	return "diceActorType"
}

// perform a step in the simulation
func (t *SimulationActor) Step(ctx context.Context) (int, error) {
	return RNG.Intn(6) + 1, nil
}

var client dapr.Client
var CurrentTime float64
var RNG *rand.Rand

func main() {
	var err error
	client, err = dapr.NewClient()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer client.Close()

	RNG = rand.New(rand.NewSource(99))

	// Run the simulation
	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "7001"
	}
	service := daprd.NewService(":" + appPort)

	fmt.Println("Registering virtual actor factory")
	service.RegisterActorImplFactoryContext(simulationActorFactory)

	// Start the server
	err = service.Start()

	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("error listening: %v", err)
	}
}
