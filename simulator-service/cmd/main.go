package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/DiegoJohnson25/climate-control/simulator-service/internal/config"
	"github.com/DiegoJohnson25/climate-control/simulator-service/internal/mqtt"
	"github.com/DiegoJohnson25/climate-control/simulator-service/internal/provisioning"
	"github.com/DiegoJohnson25/climate-control/simulator-service/internal/simulator"
)

func main() {
	mode := flag.String("mode", "run", "mode: run or teardown")
	simulation := flag.String("simulation", "", "simulation config to run")
	flag.Parse()

	if *simulation == "" {
		log.Fatal("--simulation flag is required")
	}

	cfg, err := config.Load(*simulation)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	switch *mode {
	case "run":
		if err := run(cfg); err != nil {
			log.Fatalf("run: %v", err)
		}
	case "teardown":
		if err := teardown(cfg); err != nil {
			log.Fatalf("teardown: %v", err)
		}
	default:
		log.Fatalf("unknown mode %q — must be run or teardown", *mode)
	}
}

func run(cfg *config.Config) error {
	log.Printf("starting simulator — simulation: %s", cfg.Simulation.Name)

	users, err := provisioning.Run(cfg)
	if err != nil {
		return fmt.Errorf("provisioning: %w", err)
	}

	log.Printf("provisioning complete — %d user(s) ready", len(users))

	mqttClient, err := mqtt.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("mqtt: %w", err)
	}
	defer mqttClient.Disconnect()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		errCh <- simulator.Run(ctx, cfg, users, mqttClient)
	}()

	log.Printf("simulator running — tick interval: %ds", cfg.Simulation.TickIntervalSeconds)

	select {
	case <-quit:
		log.Println("shutting down")
		cancel()
	case err := <-errCh:
		return fmt.Errorf("simulator: %w", err)
	}

	<-errCh
	return nil
}

func teardown(cfg *config.Config) error {
	log.Printf("tearing down simulation: %s", cfg.Simulation.Name)

	// TODO: implement teardown (requires api/client.go DeleteMe)
	log.Printf("teardown complete")
	return nil
}
