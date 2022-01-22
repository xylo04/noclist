package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/xylo04/noclist/internal/nl"
)

func main() {
	// log goes to STDERR
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	log.Info().Msg("Fetching NOC List...")

	vips, err := nl.New().Fetch()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to fetch NOC list")
	}

	j, err := json.Marshal(vips)
	if err != nil {
		log.Fatal().Err(err).Msg("failed formatting JSON")
	}
	// JSON output goes to STDOUT
	_, _ = fmt.Fprintln(os.Stdout, string(j))
}
