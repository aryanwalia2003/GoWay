package cmd

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"awb-gen/internal/assembler"
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
		// Initialize logger based on DEBUG env var
		debug := os.Getenv("DEBUG") == "true"
		logger.InitLogger(debug)
		defer logger.Sync()

		mux := http.NewServeMux()
		// Set up handlers
		genHandler := http.HandlerFunc(handler.HandleGenerate)
		authMiddleware := middleware.Auth(os.Getenv("API_KEYS"))

		concurrencyLimit := 4
		if val := os.Getenv("CONCURRENCY_LIMIT"); val != "" {
			if parsed, err := strconv.Atoi(val); err == nil {
				concurrencyLimit = parsed
			}
		}
		concurrencyMiddleware := middleware.NewConcurrency(concurrencyLimit)

		templateDir := "./templates"
		if val := os.Getenv("TEMPLATE_DIR"); val != "" {
			templateDir = val
		}

		folioAssembler := assembler.NewFolioAssembler(templateDir)
		healthH := handler.NewHealthHandler(folioAssembler)

		mux.HandleFunc("/generate", middleware.Logging(authMiddleware(concurrencyMiddleware.Wrap(genHandler))).ServeHTTP)
		mux.HandleFunc("/healthz", healthH.HandleHealthz)
		mux.HandleFunc("/livez", healthH.HandleLivez)
		mux.HandleFunc("/readyz", healthH.HandleReadyz)

		latexHandler := handler.NewLaTeXHandler(folioAssembler)
		mux.HandleFunc("/render/forward-shipping-label", middleware.Logging(authMiddleware(concurrencyMiddleware.Wrap(latexHandler))).ServeHTTP)

		// Use PORT from env if not explicitly overridden by flag
		addrPort := port
		if !cmd.Flags().Changed("port") {
			if val := os.Getenv("PORT"); val != "" {
				if parsed, err := strconv.Atoi(val); err == nil {
					addrPort = parsed
				}
			}
		}

		addr := fmt.Sprintf(":%d", addrPort)
		srv := server.NewServer(addr, mux)

		logger.Log.Info("Starting HTTP server", zap.String("addr", addr), zap.Bool("debug", debug))

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
