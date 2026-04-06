package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"awb-gen/internal/handler"
	"awb-gen/internal/logger"
	"awb-gen/internal/middleware"
	"awb-gen/internal/server"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var port int

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the GoWay HTTP microservice",
	RunE: func(cmd *cobra.Command, args []string) error {
		mux := http.NewServeMux()
		// Set up handlers
		genHandler := http.HandlerFunc(handler.HandleGenerate)
		authMiddleware := middleware.Auth(os.Getenv("API_KEYS"))

		mux.HandleFunc("/generate", middleware.Logging(authMiddleware(genHandler)).ServeHTTP)

		addr := fmt.Sprintf(":%d", port)
		srv := server.NewServer(addr, mux)

		logger.Log.Info("Starting HTTP server", zap.String("addr", addr))

		go func() {
			if err := srv.Start(); err != nil {
				logger.Log.Error("Server shutdown with error", zap.Error(err))
			}
		}()

		// Wait for SIGTERM / interrupts.
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		sig := <-c

		logger.Log.Info("Got shutdown signal", zap.String("signal", sig.String()))
		srv.Signal(sig)
		srv.Wait()
		logger.Log.Info("Server exited cleanly")
		return nil
	},
}

func init() {
	serveCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to run the HTTP service on")
	rootCmd.AddCommand(serveCmd)
}
