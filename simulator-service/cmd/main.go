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
	"github.com/DiegoJohnson25/climate-control/simulator-service/internal/provisioning"
)

func main() {
	mode := flag.String("mode", "run", "mode: run or teardown")
	simulation := flag.String("simulation", "default", "simulation config to run")
	flag.Parse()

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

	ctx, cancel := context.WithCancel(context.Background())
	_ = ctx
	defer cancel()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// TODO: start simulator.Run(ctx, cfg, users, mqttClient) here

	<-quit
	log.Println("shutting down")
	return nil
}

func teardown(cfg *config.Config) error {
	log.Printf("tearing down simulation: %s", cfg.Simulation.Name)

	// TODO: implement teardown (requires api/client.go DeleteMe)
	log.Printf("teardown complete")
	return nil
}
