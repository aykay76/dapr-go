package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"

	dapr "github.com/dapr/go-sdk/client"
	"github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/http"
)

type ClientStub struct {
	ActorId    string
	ActorType  string
	Initialise func(context.Context) error
	Step       func(context.Context) (interface{}, error)
}

func (a *ClientStub) Type() string {
	return a.ActorType
}
func (a *ClientStub) ID() string {
	return a.ActorId
}

var client dapr.Client

// Simulation represents the simulation environment
type Simulation struct {
	ActualStartTime  time.Time
	ActualFinishTime time.Time
	StartTime        time.Time
	CurrentTime      time.Time
	TimeStep         time.Duration
	AutoStep         bool
	DoneChan         chan struct{}
	Mutex            sync.Mutex
	RNG              *rand.Rand
	Service          common.Service
	StepsTaken       int
	EndSteps         int
	ActorCount       int
	CurrentActors    int
}

// NewSimulation creates a new simulation instance
func NewSimulation() *Simulation {
	return &Simulation{
		AutoStep:        true,
		ActualStartTime: time.Now(),
		StartTime:       time.Date(2000, 1, 1, 0, 0, 0, 0, time.Now().UTC().Location()),
		CurrentTime:     time.Now(),
		TimeStep:        time.Hour,
		DoneChan:        make(chan struct{}),
		RNG:             rand.New(rand.NewSource(99)),
		EndSteps:        100,
		ActorCount:      1,
		CurrentActors:   0,
	}
}

func (sim *Simulation) StartHandler(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	// do some initialisation
	sim.StepsTaken = 0
	sim.ActualStartTime = time.Now()

	// main sim loop (if auto-stepping, otherwise done and wait for step invocation)
	if sim.AutoStep {
		client.InvokeMethod(ctx, "dapr-go", "step", "get")
	} else {
		fmt.Println("Simulation started and paused, invoke next step by calling http://localhost:6001/step")
	}

	if in == nil {
		err = errors.New("invocation parameter required")
		return
	}

	sim.ActualFinishTime = time.Now()
	close(sim.DoneChan)
	fmt.Println("Simulation finished, simulation end time: ", sim.CurrentTime, "; actual finish time: ", sim.ActualFinishTime, "; run time: ", sim.ActualFinishTime.Sub(sim.ActualStartTime))

	return
}

func (sim *Simulation) StepHandler(ctx context.Context, in *common.InvocationEvent) (out *common.Content, err error) {
	sim.StepsTaken++

	if sim.StepsTaken >= sim.EndSteps {
		fmt.Println("Simulation complete.")
	} else {
		// do a step
		for i := 0; i < 10; i++ {
			id := fmt.Sprintf("die%d", i)
			myActor := &ClientStub{
				ActorId:   id,
				ActorType: "diceActorType",
			}
			client.ImplActorClientStub(myActor)
			dieValue, err := myActor.Step(ctx)
			if err != nil {
				panic(err)
			}
			fmt.Printf("Die %s result = %d\n", id, dieValue)
		}

		for i := 0; i < 10; i++ {
			id := fmt.Sprintf("coin%d", i)
			myActor := &ClientStub{
				ActorId:   id,
				ActorType: "coinActorType",
			}
			client.ImplActorClientStub(myActor)
			dieValue, err := myActor.Step(ctx)
			if err != nil {
				panic(err)
			}
			fmt.Printf("Coin %s result = %d\n", id, dieValue)
		}

		if sim.AutoStep {
			client.InvokeMethod(ctx, "dapr-go", "step", "get")
		} else {
			fmt.Printf("%d steps taken of %d - call http://localhost:6001/step\n", sim.StepsTaken, sim.EndSteps)
		}
	}

	return
}

func main() {
	var err error
	client, err = dapr.NewClient()
	if err != nil {
		panic(err)
	}
	defer client.Close()

	sim := NewSimulation()

	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "6001"
	}
	sim.Service = daprd.NewService(":" + appPort)
	sim.Service.AddServiceInvocationHandler("/start", sim.StartHandler)
	sim.Service.AddServiceInvocationHandler("/step", sim.StepHandler)

	fmt.Println("You can start a simulation by calling http://localhost:6001/start")

	// Start the server
	err = sim.Service.Start()
	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("error listening: %v", err)
	}

	fmt.Println("Simulation closed nicely")
}
