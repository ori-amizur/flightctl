package healthchecker

import (
	"context"
	"errors"

	api "github.com/flightctl/flightctl/api/v1alpha1"
	"github.com/flightctl/flightctl/internal/instrumentation/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func startSpan(ctx context.Context, method string) (context.Context, trace.Span) {
	ctx, span := tracing.StartSpan(ctx, "flightctl/healthcheck", method)
	return ctx, span
}

func endSpan(span trace.Span, st api.Status) {
	span.SetAttributes(attribute.Int("status.code", int(st.Code)))

	if st.Status != "Success" {
		span.RecordError(errors.New(st.Message))
		span.SetStatus(codes.Error, st.Message)
	}

	span.End()
}
