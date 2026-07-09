package main

import (
	"context"
	"log"
	"os"

	"distry/internal/config"
	"distry/internal/db"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	pool, err := db.NewPool(ctx, cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	if err := db.Migrate(ctx, pool, command, os.Args[2:]...); err != nil {
		log.Fatal(err)
	}
}
