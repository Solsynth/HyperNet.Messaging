package main

import (
	"fmt"
	"git.solsynth.dev/hypernet/nexus/pkg/nex/sec"
	"github.com/fatih/color"
	"os"
	"os/signal"
	"syscall"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/gap"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/grpc"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/services"
	"github.com/robfig/cron/v3"

	"git.solsynth.dev/hypernet/messaging/pkg/internal/http"

	pkg "git.solsynth.dev/hypernet/messaging/pkg/internal"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/cache"
	"git.solsynth.dev/hypernet/messaging/pkg/internal/database"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
}

func main() {
	// Booting screen
	fmt.Println(color.YellowString(" __  __                           _\n|  \\/  | ___  ___ ___  __ _  __ _(_)_ __   __ _\n| |\\/| |/ _ \\/ __/ __|/ _` |/ _` | | '_ \\ / _` |\n| |  | |  __/\\__ \\__ \\ (_| | (_| | | | | | (_| |\n|_|  |_|\\___||___/___/\\__,_|\\__, |_|_| |_|\\__, |\n                            |___/         |___/"))
	fmt.Printf("%s v%s\n", color.New(color.FgHiYellow).Add(color.Bold).Sprintf("Hypernet.Messaging"), pkg.AppVersion)
	fmt.Printf("The instant messaging service in Hypernet\n")
	color.HiBlack("=====================================================\n")

	// Configure settings
	viper.AddConfigPath(".")
	viper.AddConfigPath("..")
	viper.SetConfigName("settings")
	viper.SetConfigType("toml")

	// Load settings
	if err := viper.ReadInConfig(); err != nil {
		log.Panic().Err(err).Msg("An error occurred when loading settings.")
	}

	// Connect to nexus
	if err := gap.InitializeToNexus(); err != nil {
		log.Fatal().Err(err).Msg("An error occurred when connecting to nexus...")
	}

	// Load keypair
	if reader, err := sec.NewInternalTokenReader(viper.GetString("security.internal_public_key")); err != nil {
		log.Error().Err(err).Msg("An error occurred when reading internal public key for jwt. Authentication related features will be disabled.")
	} else {
		http.IReader = reader
		log.Info().Msg("Internal jwt public key loaded.")
	}

	// Connect to database
	if err := database.NewGorm(); err != nil {
		log.Fatal().Err(err).Msg("An error occurred when connect to database.")
	} else if err := database.RunMigration(database.C); err != nil {
		log.Fatal().Err(err).Msg("An error occurred when running database auto migration.")
	}

	// Initialize cache
	if err := cache.NewStore(); err != nil {
		log.Fatal().Err(err).Msg("An error occurred when initializing cache.")
	}

	// Connect other services
	services.SetupLiveKit()

	// Server
	go http.NewServer().Listen()

	go grpc.NewGrpc().Listen()

	// Configure timed tasks
	quartz := cron.New(cron.WithLogger(cron.VerbosePrintfLogger(&log.Logger)))
	quartz.AddFunc("@every 60m", services.DoAutoDatabaseCleanup)
	quartz.Start()

	// Messages
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	quartz.Stop()
}
