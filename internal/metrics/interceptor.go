package metrics

import (
	"context"
	"strings"
	"time"

	"connectrpc.com/connect"
)

// Interceptor returns a Connect interceptor that increments requests_total,
// observes duration_seconds, and tracks in_flight for every handler. Add it
// outermost (before auth) on the server's interceptor stack so it captures
// auth failures too.
func Interceptor() connect.Interceptor {
	return &rpcInterceptor{}
}

type rpcInterceptor struct{}

func (i *rpcInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		service, method := splitProcedure(req.Spec().Procedure)
		RPCInFlight.WithLabelValues(service, method).Inc()
		start := time.Now()

		resp, err := next(ctx, req)

		RPCInFlight.WithLabelValues(service, method).Dec()
		RPCDurationSeconds.WithLabelValues(service, method).Observe(time.Since(start).Seconds())
		RPCRequestsTotal.WithLabelValues(service, method, codeLabel(err)).Inc()
		return resp, err
	}
}

func (i *rpcInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	// The server doesn't make outbound stream calls.
	return next
}

func (i *rpcInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		service, method := splitProcedure(conn.Spec().Procedure)
		RPCInFlight.WithLabelValues(service, method).Inc()
		start := time.Now()

		err := next(ctx, conn)

		RPCInFlight.WithLabelValues(service, method).Dec()
		RPCDurationSeconds.WithLabelValues(service, method).Observe(time.Since(start).Seconds())
		RPCRequestsTotal.WithLabelValues(service, method, codeLabel(err)).Inc()
		return err
	}
}

// splitProcedure splits a Connect procedure like "/foo.bar.v1.Svc/Method" into
// ("foo.bar.v1.Svc", "Method") for use as Prometheus labels.
func splitProcedure(procedure string) (service, method string) {
	procedure = strings.TrimPrefix(procedure, "/")
	idx := strings.LastIndex(procedure, "/")
	if idx < 0 {
		return "unknown", procedure
	}
	return procedure[:idx], procedure[idx+1:]
}

func codeLabel(err error) string {
	if err == nil {
		return "ok"
	}
	return connect.CodeOf(err).String()
}
