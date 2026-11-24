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

	"github.com/vitorflores/fullcycle-go-observability/internal/servicob"
	"github.com/vitorflores/fullcycle-go-observability/internal/telemetria"
)

func init() {
	viper.AutomaticEnv()
	viper.SetDefault("HTTP_PORT", ":8181")
	viper.SetDefault("VIACEP_URL", "https://viacep.com.br")
	viper.SetDefault("GEOCODING_URL", "https://geocoding-api.open-meteo.com")
	viper.SetDefault("PREVISAO_URL", "https://api.open-meteo.com")
	viper.SetDefault("REQUEST_NAME_OTEL", "servico-b-request")
	viper.SetDefault("OTEL_SERVICE_NAME", "servico-b")
	viper.SetDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
}

func main() {
	shutdown, err := telemetria.InitProvider(viper.GetString("OTEL_SERVICE_NAME"), viper.GetString("OTEL_EXPORTER_OTLP_ENDPOINT"))
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	config := &servicob.Config{
		RequestNameOTEL: viper.GetString("REQUEST_NAME_OTEL"),
		OTELTracer:      otel.Tracer("servico-b-tracer"),
		ViaCEP: &servicob.ViaCEPClient{
			BaseURL: viper.GetString("VIACEP_URL"),
			Client:  client,
		},
		Clima: &servicob.ClimaClient{
			GeocodingURL: viper.GetString("GEOCODING_URL"),
			PrevisaoURL:  viper.GetString("PREVISAO_URL"),
			Client:       client,
		},
	}
	server := &http.Server{
		Addr:    viper.GetString("HTTP_PORT"),
		Handler: servicob.NewServer(config).CreateServer(),
	}

	go func() {
		log.Println("Serviço B rodando na porta", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Não foi possível escutar em %s: %v\n", server.Addr, err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("Desligando o Serviço B...")
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Erro ao desligar o servidor: %v\n", err)
	}
	if err := shutdown(ctx); err != nil {
		log.Printf("Erro ao desligar o TracerProvider: %v\n", err)
	}
}
