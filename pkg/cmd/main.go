package main

import (
	"os"
	"os/signal"
	"syscall"

	"git.solsynth.dev/hydrogen/messaging/pkg/services"
	"github.com/robfig/cron/v3"

	"git.solsynth.dev/hydrogen/messaging/pkg/grpc"
	"git.solsynth.dev/hydrogen/messaging/pkg/server"

	messaging "git.solsynth.dev/hydrogen/messaging/pkg"
	"git.solsynth.dev/hydrogen/messaging/pkg/database"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
}

func main() {
	// Configure settings
	viper.AddConfigPath(".")
	viper.AddConfigPath("..")
	viper.SetConfigName("settings")
	viper.SetConfigType("toml")

	// Load settings
	if err := viper.ReadInConfig(); err != nil {
		log.Panic().Err(err).Msg("An error occurred when loading settings.")
	}

	// Connect to database
	if err := database.NewSource(); err != nil {
		log.Fatal().Err(err).Msg("An error occurred when connect to database.")
	} else if err := database.RunMigration(database.C); err != nil {
		log.Fatal().Err(err).Msg("An error occurred when running database auto migration.")
	}

	// Connect other services
	services.SetupLiveKit()
	if err := grpc.ConnectPassport(); err != nil {
		log.Fatal().Err(err).Msg("An error occurred when connecting to passport...")
	}
	if err := grpc.ConnectPaperclip(); err != nil {
		log.Fatal().Err(err).Msg("An error occurred when connecting to paperclip...")
	}

	// Server
	server.NewServer()
	go server.Listen()

	// Configure timed tasks
	quartz := cron.New(cron.WithLogger(cron.VerbosePrintfLogger(&log.Logger)))
	quartz.AddFunc("@every 60m", services.DoAutoDatabaseCleanup)
	quartz.Start()

	// Messages
	log.Info().Msgf("Messaging v%s is started...", messaging.AppVersion)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msgf("Messaging v%s is quitting...", messaging.AppVersion)

	quartz.Stop()
}
