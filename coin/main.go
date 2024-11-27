package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"

	"github.com/dapr/go-sdk/actor"
	dapr "github.com/dapr/go-sdk/client"
	"github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/http"
)

type SimulationActor struct {
	actor.ServerImplBaseCtx
	daprClient dapr.Client
}

func simulationActorFactory() actor.ServerContext {
	client, err := dapr.NewClient()
	if err != nil {
		panic(err)
	}
	return &SimulationActor{
		daprClient: client,
	}
}

func (t *SimulationActor) Type() string {
	return "coinActorType"
}

// perform a step in the simulation
func (t *SimulationActor) Step(context.Context) (string, error) {
	value := sim.RNG.Intn(2) + 1
	if value == 1 {
		return "heads", nil
	}

	return "tails", nil
}

type Event struct {
	Action  string       `json:"action,omitempty"`
	Reason  string       `json:"reason,omitempty"`
	Target  string       `json:"target,omitempty"`
	Payload EventPayload `json:"payload,omitempty"`
}

type EventPayload struct {
	Value int `json:"value,omitempty"`
}

func EventReceived(ctx context.Context, e *common.TopicEvent) (retry bool, err error) {
	fmt.Println("Coin received:", e.Data)
	return false, nil
}

var client dapr.Client

// Simulation represents the simulation environment
type Simulation struct {
	CurrentTime float64
	DoneChan    chan struct{}
	Mutex       sync.Mutex
	RNG         *rand.Rand
	Service     common.Service
}

func eventHandler(ctx context.Context, e *common.TopicEvent) (retry bool, err error) {
	var event Event
	err = json.Unmarshal([]byte(e.RawData), &event)
	if err != nil {
		fmt.Println("Error unmarshalling event")
		return false, err
	}

	if event.Action == "start" {
		fmt.Println("Actor received:", event.Action)

		doneEvent := &Event{
			Action: "startdone",
			Reason: "Actor finished initialisation",
		}
		err = client.PublishEvent(context.Background(), "simulation", "core", doneEvent)
		if err != nil {
			panic(err)
		}
	}

	if event.Action == "step" {
		fmt.Println("Actor received:", event.Action)

		value := sim.RNG.Intn(5) + 1
		fmt.Println("Coin value:", value)

		doneEvent := &Event{
			Action: "stepdone",
			Reason: "Actor finished step of simulation",
		}
		err = client.PublishEvent(context.Background(), "simulation", "core", doneEvent)
		if err != nil {
			panic(err)
		}
	}

	if event.Action == "stop" {
		fmt.Println("Actor received:", event.Action)

		doneEvent := &Event{
			Action: "stopdone",
			Reason: "Actor finished step of simulation",
		}
		err = client.PublishEvent(context.Background(), "simulation", "core", doneEvent)
		if err != nil {
			panic(err)
		}
	}

	return false, nil
}

var sim *Simulation

func main() {
	var err error
	client, err = dapr.NewClient()
	if err != nil {
		fmt.Println(err)
		return
	}
	defer client.Close()

	// Run the simulation
	sim = &Simulation{
		CurrentTime: 0.0,
		DoneChan:    make(chan struct{}),
		RNG:         rand.New(rand.NewSource(99)),
	}

	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "7002"
	}
	sim.Service = daprd.NewService(":" + appPort)

	var sub = &common.Subscription{
		PubsubName: "simulation",
		Topic:      "core",
		Route:      "/endpoint",
	}

	err = sim.Service.AddTopicEventHandler(sub, eventHandler)
	if err != nil {
		log.Fatalf("error adding topic subscription: %v", err)
	}

	fmt.Println("Registering virtual actor factory")
	sim.Service.RegisterActorImplFactoryContext(simulationActorFactory)

	// Start the server
	err = sim.Service.Start()

	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("error listening: %v", err)
	}
}
