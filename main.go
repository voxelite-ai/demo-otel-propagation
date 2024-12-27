package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/voxelite-ai/demo-service/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

func main() {
	ctx := context.Background()

	t, err := tracing.Start(ctx)
	if err != nil {
		panic(err)
	}

	// dead simple http echo server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Received request for /")

		if err := getResources(ctx); err != nil {
			slog.ErrorContext(ctx, "Failed to get resources")
			return
		}

		w.Write([]byte("Hello, World!"))
	})

	slog.Info("Starting server on :8070")
	if err := http.ListenAndServe(":8070", nil); err != nil {
		slog.ErrorContext(ctx, "Failed to start server")
		panic(err)
	}

	defer func() { _ = t.Shutdown(ctx) }()
}

var tracer = otel.Tracer("demo-service")

func getResources(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "getResources")
	defer span.End()

	slog.Info("Getting resources")

	client := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/api/v1/resources", nil)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to create request")

		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to create request")

		return err
	}

	var jsonResponse map[string]interface{}
	response, err := client.Do(req)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to get resources")

		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to get resources")

		return err
	}

	if err := json.NewDecoder(response.Body).Decode(&jsonResponse); err != nil {
		slog.ErrorContext(ctx, "Failed to decode response")
		return err
	}

	slog.Info("Response", slog.Any("status", response.Status), slog.Any("body", jsonResponse))

	return nil
}
