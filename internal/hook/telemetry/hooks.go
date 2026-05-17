package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"dolphin/internal/agent"
	"dolphin/internal/hook"
	"dolphin/internal/mcp"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "dolphin/agent"

// spanStore bridges span references across hook points that don't share Values maps.
var spanStore sync.Map

func sessionKey(sid string) string              { return "session:" + sid }
func turnKey(sid string, turn int) string       { return fmt.Sprintf("turn:%s:%d", sid, turn) }
func turnErrorKey(sid string, turn int) string  { return fmt.Sprintf("turn_error:%s:%d", sid, turn) }
func llmKey(sid string, turn int) string        { return fmt.Sprintf("llm:%s:%d", sid, turn) }
func llmTimingKey(sid string, turn int) string  { return fmt.Sprintf("llm_time:%s:%d", sid, turn) }
func llmModelKey(sid string, turn int) string   { return fmt.Sprintf("llm_model:%s:%d", sid, turn) }
func toolKey(sid string, turn int) string       { return fmt.Sprintf("tool:%s:%d", sid, turn) }
func toolTimingKey(sid string, turn int) string { return fmt.Sprintf("tool_time:%s:%d", sid, turn) }
func responseKey(sid string, turn int) string   { return fmt.Sprintf("response:%s:%d", sid, turn) }
func schedulerTimingKey(sid, name string) string {
	return fmt.Sprintf("scheduler_time:%s:%s", sid, name)
}

// childContext returns a context carrying parent's SpanContext so that a new
// span started with it becomes a child of parent.
func childContext(parent trace.Span) context.Context {
	return trace.ContextWithSpanContext(context.Background(), parent.SpanContext())
}

// RegisterHooks registers all OTel hook handlers on the given registry.
func RegisterHooks(reg *hook.Registry) {
	reg.Register(hook.PointSessionStart, 100, sessionStartHook)
	reg.Register(hook.PointSessionEnd, 100, sessionEndHook)
	reg.Register(hook.PointUserInput, 100, userInputHook)
	reg.Register(hook.PointBeforeLLM, 100, beforeLLMHook)
	reg.Register(hook.PointAfterLLM, 100, afterLLMHook)
	reg.Register(hook.PointBeforeTool, 100, beforeToolHook)
	reg.Register(hook.PointAfterTool, 100, afterToolHook)
	reg.Register(hook.PointBeforeResponse, 100, beforeResponseHook)
	reg.Register(hook.PointOnError, 100, errorHook)

	// Scheduler
	reg.Register(hook.PointSchedulerTaskBefore, 100, schedulerTaskBeforeHook)
	reg.Register(hook.PointSchedulerTaskAfter, 100, schedulerTaskAfterHook)

	// Transport
	reg.Register(hook.PointTransportConnect, 100, transportConnectHook)
	reg.Register(hook.PointTransportDisconnect, 100, transportDisconnectHook)
	reg.Register(hook.PointTransportReceive, 100, transportReceiveHook)
	reg.Register(hook.PointTransportSend, 100, transportSendHook)
}

// ---- session ----

func sessionStartHook(ctx context.Context, hc *hook.Context) error {
	RecordSessionStart()
	tr := Tracer(tracerName)
	_, span := tr.Start(ctx, "session",
		trace.WithSpanKind(trace.SpanKindServer),
	)
	span.SetAttributes(attribute.String("session.id", hc.SessionID))
	spanStore.Store(sessionKey(hc.SessionID), span)
	return nil
}

func sessionEndHook(ctx context.Context, hc *hook.Context) error {
	sid := hc.SessionID

	// End the current turn span if still active (last turn).
	_, hadErr := spanStore.LoadAndDelete(turnErrorKey(sid, hc.Turn))
	if v, ok := spanStore.LoadAndDelete(turnKey(sid, hc.Turn)); ok {
		last := v.(trace.Span)
		if !hadErr {
			last.SetStatus(codes.Ok, "")
		}
		last.End()
	}

	if v, ok := spanStore.LoadAndDelete(sessionKey(sid)); ok {
		span := v.(trace.Span)
		span.SetAttributes(attribute.Int("turn.count", hc.Turn))
		span.SetStatus(codes.Ok, "")
		span.End()
	}
	RecordSessionEnd()
	return nil
}

// ---- turn ----

func userInputHook(ctx context.Context, hc *hook.Context) error {
	sid := hc.SessionID
	turn := hc.Turn

	// End the previous turn span.
	if turn > 1 {
		_, hadErr := spanStore.LoadAndDelete(turnErrorKey(sid, turn-1))
		if v, ok := spanStore.LoadAndDelete(turnKey(sid, turn-1)); ok {
			prev := v.(trace.Span)
			if !hadErr {
				prev.SetStatus(codes.Ok, "")
			}
			prev.End()
		}
	}

	// Parent: session span.
	var parentCtx context.Context
	if v, ok := spanStore.Load(sessionKey(sid)); ok {
		parentCtx = childContext(v.(trace.Span))
	} else {
		parentCtx = ctx
	}

	RecordTurn(ctx)
	tr := Tracer(tracerName)
	_, span := tr.Start(parentCtx, "turn",
		trace.WithSpanKind(trace.SpanKindServer),
	)
	span.SetAttributes(
		attribute.String("session.id", sid),
		attribute.Int("turn.number", turn),
		attribute.String("user.input", truncate(hc.UserInput, 256)),
	)
	spanStore.Store(turnKey(sid, turn), span)
	return nil
}

// ---- llm ----

func beforeLLMHook(ctx context.Context, hc *hook.Context) error {
	sid := hc.SessionID
	turn := hc.Turn

	// Parent: current turn span.
	var parentCtx context.Context
	if v, ok := spanStore.Load(turnKey(sid, turn)); ok {
		parentCtx = childContext(v.(trace.Span))
	} else {
		parentCtx = ctx
	}

	tr := Tracer(tracerName)
	_, span := tr.Start(parentCtx, "llm.call",
		trace.WithSpanKind(trace.SpanKindClient),
	)
	span.SetAttributes(
		attribute.String("session.id", sid),
		attribute.Int("turn.number", turn),
	)

	model := ""
	if req, ok := hc.Request.(*agent.ProviderRequest); ok {
		model = req.Model
		span.SetAttributes(
			attribute.String("gen_ai.request.model", model),
			attribute.Int("gen_ai.request.max_tokens", req.MaxTokens),
			attribute.Int("message.count", len(req.Messages)),
		)
	}

	RecordLLMRequest(ctx, model)
	spanStore.Store(llmKey(sid, turn), span)
	spanStore.Store(llmTimingKey(sid, turn), time.Now())
	spanStore.Store(llmModelKey(sid, turn), model)
	return nil
}

func afterLLMHook(ctx context.Context, hc *hook.Context) error {
	sid := hc.SessionID
	turn := hc.Turn

	// Compute elapsed time
	var elapsed time.Duration
	if v, ok := spanStore.LoadAndDelete(llmTimingKey(sid, turn)); ok {
		elapsed = time.Since(v.(time.Time))
	}

	// End response span first (created by beforeResponseHook).
	if v, ok := spanStore.LoadAndDelete(responseKey(sid, turn)); ok {
		rsp := v.(trace.Span)
		if hc.Error != nil {
			rsp.SetStatus(codes.Error, hc.Error.Error())
		} else {
			rsp.SetStatus(codes.Ok, "")
		}
		rsp.End()
	}

	v, ok := spanStore.LoadAndDelete(llmKey(sid, turn))
	if !ok {
		return nil
	}
	span := v.(trace.Span)
	defer span.End()

	model := ""
	if v, ok := spanStore.LoadAndDelete(llmModelKey(sid, turn)); ok {
		model = v.(string)
	}
	var inputTokens, outputTokens int64
	if resp, ok := hc.Response.(*agent.ProviderResponse); ok {
		if resp.Usage != nil {
			inputTokens = int64(resp.Usage.InputTokens)
			outputTokens = int64(resp.Usage.OutputTokens)
			span.SetAttributes(
				attribute.Int("gen_ai.usage.input_tokens", resp.Usage.InputTokens),
				attribute.Int("gen_ai.usage.output_tokens", resp.Usage.OutputTokens),
				attribute.String("gen_ai.response.stop_reason", resp.StopReason),
			)
		}
	}

	if hc.Error != nil {
		span.SetStatus(codes.Error, hc.Error.Error())
		span.RecordError(hc.Error)
		RecordLLMError(ctx, model)
	} else {
		span.SetStatus(codes.Ok, "")
	}

	if elapsed > 0 {
		RecordLLMLatency(ctx, model, elapsed)
	}
	if inputTokens > 0 || outputTokens > 0 {
		RecordLLMTokens(ctx, model, inputTokens, outputTokens)
	}
	return nil
}

// ---- tool ----

func beforeToolHook(ctx context.Context, hc *hook.Context) error {
	sid := hc.SessionID
	turn := hc.Turn

	var parentCtx context.Context
	if v, ok := spanStore.Load(turnKey(sid, turn)); ok {
		parentCtx = childContext(v.(trace.Span))
	} else {
		parentCtx = ctx
	}

	tr := Tracer(tracerName)
	_, span := tr.Start(parentCtx, fmt.Sprintf("tool.%s", hc.ToolName),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	span.SetAttributes(
		attribute.String("session.id", sid),
		attribute.Int("turn.number", turn),
		attribute.String("tool.name", hc.ToolName),
	)
	if len(hc.ToolArgs) > 0 {
		span.SetAttributes(attribute.String("tool.args", truncateJSON(hc.ToolArgs, 256)))
	}

	RecordToolCall(ctx, hc.ToolName)
	spanStore.Store(toolKey(sid, turn), span)
	spanStore.Store(toolTimingKey(sid, turn), time.Now())
	return nil
}

func afterToolHook(ctx context.Context, hc *hook.Context) error {
	sid := hc.SessionID
	turn := hc.Turn

	var elapsed time.Duration
	if v, ok := spanStore.LoadAndDelete(toolTimingKey(sid, turn)); ok {
		elapsed = time.Since(v.(time.Time))
	}

	v, ok := spanStore.LoadAndDelete(toolKey(sid, turn))
	if !ok {
		return nil
	}
	span := v.(trace.Span)
	defer span.End()

	hasError := false
	if result, ok := hc.ToolResult.(*mcp.ToolResult); ok {
		span.SetAttributes(attribute.Bool("tool.is_error", result.IsError))
		if result.IsError {
			span.SetStatus(codes.Error, truncate(result.Content, 256))
			hasError = true
		} else {
			span.SetAttributes(attribute.String("tool.result", truncate(result.Content, 256)))
		}
	}

	if hc.Error != nil {
		span.SetStatus(codes.Error, hc.Error.Error())
		span.RecordError(hc.Error)
		hasError = true
	}
	if hasError {
		RecordToolError(ctx, hc.ToolName)
	} else {
		span.SetStatus(codes.Ok, "")
	}
	if elapsed > 0 {
		RecordToolLatency(ctx, hc.ToolName, elapsed)
	}
	return nil
}

// ---- response ----

func beforeResponseHook(ctx context.Context, hc *hook.Context) error {
	sid := hc.SessionID
	turn := hc.Turn

	var parentCtx context.Context
	if v, ok := spanStore.Load(turnKey(sid, turn)); ok {
		parentCtx = childContext(v.(trace.Span))
	} else {
		parentCtx = ctx
	}

	tr := Tracer(tracerName)
	_, span := tr.Start(parentCtx, "response.deliver",
		trace.WithSpanKind(trace.SpanKindServer),
	)
	span.SetAttributes(
		attribute.String("session.id", sid),
		attribute.Int("turn.number", turn),
	)
	spanStore.Store(responseKey(sid, turn), span)
	return nil
}

// ---- error ----

func errorHook(ctx context.Context, hc *hook.Context) error {
	if hc.Error == nil {
		return nil
	}
	sid := hc.SessionID
	turn := hc.Turn

	// Mark turn as errored so userInputHook/sessionEndHook won't overwrite with Ok.
	spanStore.Store(turnErrorKey(sid, turn), true)

	for _, key := range []string{
		turnKey(sid, turn),
		llmKey(sid, turn),
		toolKey(sid, turn),
		responseKey(sid, turn),
	} {
		if v, ok := spanStore.Load(key); ok {
			span := v.(trace.Span)
			span.SetStatus(codes.Error, hc.Error.Error())
			span.RecordError(hc.Error)
		}
	}
	return nil
}

// ---- helpers ----

// ---- scheduler ----

func schedulerTaskBeforeHook(ctx context.Context, hc *hook.Context) error {
	tr := Tracer(tracerName)
	_, span := tr.Start(ctx, "scheduler.task",
		trace.WithSpanKind(trace.SpanKindInternal),
	)
	span.SetAttributes(
		attribute.String("task.name", hc.TaskName),
		attribute.String("session.id", hc.SessionID),
	)
	spanStore.Store(schedulerTimingKey(hc.SessionID, hc.TaskName), span)
	spanStore.Store(schedulerTimingKey(hc.SessionID, hc.TaskName+"_start"), time.Now())
	RecordSchedulerTask(ctx, hc.TaskName)
	return nil
}

func schedulerTaskAfterHook(ctx context.Context, hc *hook.Context) error {
	key := schedulerTimingKey(hc.SessionID, hc.TaskName)
	if v, ok := spanStore.LoadAndDelete(schedulerTimingKey(hc.SessionID, hc.TaskName+"_start")); ok {
		elapsed := time.Since(v.(time.Time))
		RecordSchedulerTaskLatency(ctx, hc.TaskName, elapsed)
	}
	if v, ok := spanStore.LoadAndDelete(key); ok {
		span := v.(trace.Span)
		if hc.Error != nil {
			span.SetStatus(codes.Error, hc.Error.Error())
			span.RecordError(hc.Error)
			RecordSchedulerTaskError(ctx, hc.TaskName)
		} else {
			span.SetStatus(codes.Ok, "")
		}
		span.End()
	}
	return nil
}

// ---- transport ----

func transportConnectHook(ctx context.Context, hc *hook.Context) error {
	tr := Tracer(tracerName)
	_, span := tr.Start(ctx, "transport.connect",
		trace.WithSpanKind(trace.SpanKindInternal),
	)
	span.SetAttributes(
		attribute.String("transport.name", hc.TransportName),
		attribute.String("session.id", hc.SessionID),
	)
	span.SetStatus(codes.Ok, "")
	span.End()
	RecordTransportConnect(ctx)
	return nil
}

func transportDisconnectHook(ctx context.Context, hc *hook.Context) error {
	RecordTransportDisconnect(ctx)
	return nil
}

func transportReceiveHook(ctx context.Context, hc *hook.Context) error {
	tr := Tracer(tracerName)
	_, span := tr.Start(ctx, "transport.receive",
		trace.WithSpanKind(trace.SpanKindServer),
	)
	span.SetAttributes(
		attribute.String("transport.name", hc.TransportName),
		attribute.String("session.id", hc.SessionID),
		attribute.Int("turn.number", hc.Turn),
	)
	span.SetStatus(codes.Ok, "")
	span.End()
	RecordTransportRx(ctx, hc.TransportName)
	return nil
}

func transportSendHook(ctx context.Context, hc *hook.Context) error {
	tr := Tracer(tracerName)
	_, span := tr.Start(ctx, "transport.send",
		trace.WithSpanKind(trace.SpanKindServer),
	)
	span.SetAttributes(
		attribute.String("transport.name", hc.TransportName),
		attribute.String("session.id", hc.SessionID),
		attribute.Int("turn.number", hc.Turn),
	)
	span.SetStatus(codes.Ok, "")
	span.End()
	RecordTransportTx(ctx, hc.TransportName)
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func truncateJSON(data json.RawMessage, max int) string {
	s := string(data)
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
