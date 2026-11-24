package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/viper"
	"go.opentelemetry.io/otel"

	"github.com/vitorflores/fullcycle-go-observability/internal/servicoa"
	"github.com/vitorflores/fullcycle-go-observability/internal/telemetria"
)

func init() {
	viper.AutomaticEnv()
	viper.SetDefault("HTTP_PORT", ":8080")
	viper.SetDefault("SERVICO_B_URL", "http://localhost:8181")
	viper.SetDefault("REQUEST_NAME_OTEL", "servico-a-request")
	viper.SetDefault("OTEL_SERVICE_NAME", "servico-a")
	viper.SetDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
}

func main() {
	shutdown, err := telemetria.InitProvider(viper.GetString("OTEL_SERVICE_NAME"), viper.GetString("OTEL_EXPORTER_OTLP_ENDPOINT"))
	if err != nil {
		log.Fatal(err)
	}

	config := &servicoa.Config{
		ServicoBURL:     viper.GetString("SERVICO_B_URL"),
		RequestNameOTEL: viper.GetString("REQUEST_NAME_OTEL"),
		OTELTracer:      otel.Tracer("servico-a-tracer"),
	}
	server := &http.Server{
		Addr:    viper.GetString("HTTP_PORT"),
		Handler: servicoa.NewServer(config).CreateServer(),
	}

	go func() {
		log.Println("Serviço A rodando na porta", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Não foi possível escutar em %s: %v\n", server.Addr, err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("Desligando o Serviço A...")
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Erro ao desligar o servidor: %v\n", err)
	}
	if err := shutdown(ctx); err != nil {
		log.Printf("Erro ao desligar o TracerProvider: %v\n", err)
	}
}
