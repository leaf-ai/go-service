// Copyright 2020-2022 (c) The Go Service Components authors. All rights reserved. Issued under the Apache 2.0 License.

package server // import "github.com/karlmutch/go-service/pkg/server"

// This file contains an open telemetry based exporter for the
// honeycomb observability service

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/karlmutch/go-service/pkg/log"
	"github.com/karlmutch/go-service/pkg/network"

	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-stack/stack"
	"github.com/karlmutch/kv"
)

var (
	hostKey  = "service/host"
	nodeKey  = "service/node"
	hostName = network.GetHostName()
)

func init() {
	// If the hosts FQDN or network name is not known use the
	// hostname reported by the Kernel
	if hostName == "localhost" || hostName == "unknown" || len(hostName) == 0 {
		hostName, _ = os.Hostname()
	}
}

func StartTelemetry(ctx context.Context, logger *log.Logger, nodeName string, serviceName string, apiKey string, dataset string) (newCtx context.Context, err kv.Error) {

	// Create an OTLP exporter, passing in Honeycomb credentials as environment variables.
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint("api.honeycomb.io:443"),
		otlptracegrpc.WithHeaders(map[string]string{
			"x-honeycomb-team":    os.Getenv("HONEYCOMB_API_KEY"),
			"x-honeycomb-dataset": os.Getenv("HONEYCOMB_DATASET"),
		}),
		otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")),
	}

	client := otlptracegrpc.NewClient(opts...)
	exp, errGo := otlptrace.New(ctx, client)
	if errGo != nil {
		return ctx, kv.Wrap(errGo).With("stack", stack.Trace().TrimRuntime())
	}

	// Create a new tracer provider with a batch span processor and the otlp exporter.
	// Add a resource attribute service.name that identifies the service in the Honeycomb UI.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceNameKey.String(serviceName))),
	)

	// Set the Tracer Provider and the W3C Trace Context propagator as globals
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}),
	)

	labels := []attribute.KeyValue{
		attribute.String(hostKey, hostName),
	}
	if len(nodeName) != 0 {
		labels = append(labels, attribute.String(nodeKey, nodeName))
	}

	ctx, span := otel.Tracer(serviceName).Start(ctx, "test-run")
	span.SetAttributes(labels...)

	go func() {
		<-ctx.Done()

		span.End()

		// Allow other processing to terminate before forcably stopping OpenTelemetry collection
		shutCtx, cancel := context.WithTimeout(context.Background(), time.Duration(10*time.Second))
		defer cancel()

		if errGo := tp.Shutdown(shutCtx); errGo != nil {
			fmt.Println(spew.Sdump(errGo), "stack", stack.Trace().TrimRuntime())
		}
	}()

	return ctx, nil
}
