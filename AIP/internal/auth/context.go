package auth

import "context"

type ctxKey int

const userKey ctxKey = 1

// Principal é o utilizador autenticado no request.
type Principal struct {
	ID    string
	Email string
	Role  string
	Name  string
}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, userKey, p)
}

func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(userKey).(Principal)
	return p, ok
}
