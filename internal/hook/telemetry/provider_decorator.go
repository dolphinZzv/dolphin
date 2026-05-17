package telemetry

import (
	"context"
	"fmt"

	"dolphin/internal/agent"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedProvider wraps an agent.Provider with OTel tracing.
type InstrumentedProvider struct {
	inner agent.Provider
}

// NewInstrumentedProvider creates an OTel-instrumented wrapper around a provider.
func NewInstrumentedProvider(p agent.Provider) agent.Provider {
	return &InstrumentedProvider{inner: p}
}

func (p *InstrumentedProvider) Type() agent.ProviderType { return p.inner.Type() }
func (p *InstrumentedProvider) Name() string             { return p.inner.Name() }

func (p *InstrumentedProvider) HealthCheck(ctx context.Context) error {
	return p.inner.HealthCheck(ctx)
}

func (p *InstrumentedProvider) Complete(ctx context.Context, req agent.ProviderRequest) (*agent.ProviderResponse, error) {
	tr := Tracer(tracerName)
	ctx, span := tr.Start(ctx, "llm.complete",
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	span.SetAttributes(
		attribute.String("gen_ai.request.model", req.Model),
		attribute.Int("gen_ai.request.max_tokens", req.MaxTokens),
		attribute.Int("message.count", len(req.Messages)),
	)

	resp, err := p.inner.Complete(ctx, req)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return nil, err
	}

	if resp.Usage != nil {
		span.SetAttributes(
			attribute.Int("gen_ai.usage.input_tokens", resp.Usage.InputTokens),
			attribute.Int("gen_ai.usage.output_tokens", resp.Usage.OutputTokens),
			attribute.String("gen_ai.response.stop_reason", resp.StopReason),
		)
	}
	return resp, nil
}

func (p *InstrumentedProvider) CompleteStream(ctx context.Context, req agent.ProviderRequest) (<-chan agent.StreamChunk, error) {
	tr := Tracer(tracerName)
	ctx, span := tr.Start(ctx, "llm.complete_stream",
		trace.WithSpanKind(trace.SpanKindClient),
	)

	span.SetAttributes(
		attribute.String("gen_ai.request.model", req.Model),
		attribute.Int("gen_ai.request.max_tokens", req.MaxTokens),
		attribute.Int("message.count", len(req.Messages)),
	)

	innerCh, err := p.inner.CompleteStream(ctx, req)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		span.End()
		return nil, err
	}

	// Wrap channel to detect completion and finalize span
	out := make(chan agent.StreamChunk)
	go func() {
		defer span.End()
		defer close(out)
		var totalInput, totalOutput int
		stopReason := ""
		hasError := false
		for chunk := range innerCh {
			if chunk.Usage != nil {
				totalInput = chunk.Usage.InputTokens
				totalOutput = chunk.Usage.OutputTokens
			}
			if chunk.Done {
				stopReason = fmt.Sprintf("%v", chunk.Content) // may contain final text
			}
			out <- chunk
			select {
			case <-ctx.Done():
				span.SetStatus(codes.Error, ctx.Err().Error())
				return
			default:
			}
		}
		if !hasError {
			span.SetAttributes(
				attribute.Int("gen_ai.usage.input_tokens", totalInput),
				attribute.Int("gen_ai.usage.output_tokens", totalOutput),
				attribute.String("gen_ai.response.stop_reason", stopReason),
			)
		}
	}()
	return out, nil
}
