package gtscontext

import "context"

type ctxkey string

const (
	barebonesKey = ctxkey("barebones")
)

// Barebones will return whether the "barebones" flag was set in this context,
// indicating that only a barebones model was requested (e.g. database models).
func Barebones(ctx context.Context) bool {
	_, ok := ctx.Value(barebonesKey).(struct{})
	return ok
}

// SetBarebones wraps the context to set the "barebones" flag, to return true to Barebones().
func SetBarebones(ctx context.Context) context.Context {
	return context.WithValue(ctx, barebonesKey, struct{}{})
}
