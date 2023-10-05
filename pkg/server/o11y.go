// Copyright 2020-2022 (c) The Go Service Components authors. All rights reserved. Issued under the Apache 2.0 License.

package server // import "github.com/karlmutch/go-service/pkg/server"

// This file contains an open telemetry based exporter for the
// honeycomb observability service

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/karlmutch/go-service/pkg/network"

	"google.golang.org/grpc/credentials"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"

	"github.com/go-stack/stack"
	"github.com/karlmutch/kv"
)

const (
	hostKey = "service.host"
	nodeKey = "service.node"

	defaultOTelEndpoint    = "api.honeycomb.io:443"
	defaultCooldown        = time.Duration(2 * time.Second)
	defaultShutdownTimeout = time.Duration(5 * time.Second)
)

var (
	hostName = network.GetHostName()
)

func init() {
	// If the hosts FQDN or network name is not known use the
	// hostname reported by the Kernel
	if hostName == "localhost" || hostName == "unknown" || len(hostName) == 0 {
		hostName, _ = os.Hostname()
	}
}

// StartTelemetryOpts is used to specify parameters for starting the OpenTelemetry module
type StartTelemetryOpts struct {
	NodeName    string           // Logical host name for OTel entries
	ServiceName string           //
	ProjectID   string           // A project identification string, typically the Go module name
	ApiKey      string           // The OTel server API key
	Dataset     string           // The OTel dataset identifier for all OTel information
	ApiEndpoint string           // The TCP/IP endpoint for the OTel server, or collector
	Cooldown    time.Duration    // The duration of time to wait after a termination signal is received to allow other modules to send events etc and end their own spans
	Bag         *baggage.Baggage // KV Pairs to propagate to all spans
}

// StartTelemetry is used to initialize OpenTelemetry tracing, the ctx (context) is used to
// close the root span when the sever closes the channel.  The options structure contains
// parameters for the OTel code.
func StartTelemetry(ctx context.Context, options StartTelemetryOpts, logger slog.Logger) (newCtx context.Context, err kv.Error) {

	endpoint := options.ApiEndpoint
	if len(endpoint) == 0 {
		endpoint = defaultOTelEndpoint
	}
	// Create an OTLP exporter, passing in Honeycomb credentials as environment variables.
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithHeaders(map[string]string{
			"x-honeycomb-team":    options.ApiKey,
			"x-honeycomb-dataset": options.Dataset,
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
		sdktrace.WithResource(resource.NewWithAttributes(semconv.SchemaURL, semconv.ServiceNameKey.String(options.ServiceName))),
	)

	// Set the Tracer Provider and the W3C Trace Context propagator as globals
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}),
	)

	members := []baggage.Member{}

	labels := []attribute.KeyValue{}

	if len(hostName) != 0 {
		labels = append(labels, attribute.String(hostKey, hostName))
		if member, errGo := baggage.NewMember(hostKey, hostName); errGo == nil {
			members = append(members, member)
		} else {
			logger.WarnContext(ctx, "error", errGo)
		}
	}
	if len(options.NodeName) != 0 {
		labels = append(labels, attribute.String(nodeKey, options.NodeName))
		if member, errGo := baggage.NewMember(nodeKey, options.NodeName); errGo == nil {
			members = append(members, member)
		} else {
			logger.WarnContext(ctx, "error", errGo)
		}
	}

	var bag baggage.Baggage
	if nil == options.Bag {
		if bag, errGo = baggage.New(); errGo != nil {
			logger.WarnContext(ctx, "error", errGo)
		}
	} else {
		bag = *options.Bag
	}

	for _, member := range members {
		if bag, errGo = bag.SetMember(member); errGo != nil {
			logger.WarnContext(ctx, "error", errGo)
		}
	}
	ctx = baggage.ContextWithBaggage(ctx, bag)

	ctx, span := otel.Tracer(options.ServiceName).Start(ctx, options.ProjectID)
	span.SetAttributes(labels...)

	go func() {
		<-ctx.Done()

		// Allow other modules to clean up their spans
		cooldown := options.Cooldown
		if cooldown == time.Duration(0) {
			cooldown = defaultCooldown
		}
		time.Sleep(cooldown)

		span.End()

		// Allow other processing to terminate before forcably stopping OpenTelemetry collection
		shutCtx, cancel := context.WithTimeout(context.Background(), defaultShutdownTimeout)
		defer cancel()

		if errGo := tp.Shutdown(shutCtx); errGo != nil {
			logger.WarnContext(ctx, "error", errGo)
		}
	}()

	return ctx, nil
}
