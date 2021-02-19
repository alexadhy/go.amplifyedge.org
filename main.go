package main

import (
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"net/http"
	"os"
	"time"
)

var (
	config string
)

const (
	defaultConfig = "./config.json"
)

func main() {
	logger := zerolog.New(os.Stdout)
	rootCmd := &cobra.Command{
		Use:   "ampedge",
		Short: "ampedge -c config.json",
	}
	rootCmd.PersistentFlags().StringVarP(&config, "config", "c", defaultConfig, "config to use (json)")
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		srv := &http.Server{
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 7 * time.Second,
		}
		r := chi.NewRouter()
		// middlewares
		r.Use(middleware.RealIP)
		r.Use(middleware.Logger)
		r.Use(middleware.Recoverer)
		r.Use(middleware.RequestID)

		gconf, err := newGlobalConfig(config, &logger)
		if err != nil {
			return err
		}
		r.Route("/", func(r chi.Router) {
			r.Get("/", gconf.handleRequest)
			r.Get("/{any:.*}", gconf.handleRequest)
			r.Get("/{foo:.*}/{bar:.*}", gconf.handleRequest)
			r.Get("/{foo:.*}/{bar:.*}/{baz:.*}", gconf.handleRequest)
		})
		errChan := make(chan error, 1)

		srv.Addr = ":8080"
		srv.Handler = r
		go func() {
			errChan <- srv.ListenAndServe()
		}()

		return <-errChan
	}
	if err := rootCmd.Execute(); err != nil {
		logger.Error().Msg(err.Error())
	}
}
