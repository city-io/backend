// Package auth issues and validates the JWTs that authenticate RPC callers and
// provides a Connect interceptor that attaches the verified claims to the
// request context.
package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
)

// Claims is the identity carried in a token.
type Claims struct {
	UserID   string
	Username string
	Email    string
}

type contextKey struct{}

var claimsKey contextKey

// ErrNoToken is returned when a request carries no bearer token.
var ErrNoToken = errors.New("no authorization token")

// Issue mints a signed token for the given claims.
func Issue(secret string, claims Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId":   claims.UserID,
		"username": claims.Username,
		"email":    claims.Email,
	})
	return token.SignedString([]byte(secret))
}

// Validate parses and verifies a token, returning its claims.
func Validate(secret, token string) (Claims, error) {
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return Claims{}, err
	}
	mapClaims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok || !parsed.Valid {
		return Claims{}, errors.New("invalid token")
	}
	var claims Claims
	if v, ok := mapClaims["userId"].(string); ok {
		claims.UserID = v
	}
	if v, ok := mapClaims["username"].(string); ok {
		claims.Username = v
	}
	if v, ok := mapClaims["email"].(string); ok {
		claims.Email = v
	}
	return claims, nil
}

// ClaimsFromContext returns the verified claims attached by the interceptor.
// ok is false on unauthenticated requests.
func ClaimsFromContext(ctx context.Context) (Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(Claims)
	return claims, ok
}

// publicProcedures are reachable without a token.
var publicProcedures = map[string]bool{
	"/cityio.v1.UserService/Register": true,
	"/cityio.v1.UserService/Login":    true,
}

// Interceptor verifies the bearer token on every non-public procedure and
// stores the resulting claims on the context. It covers both unary and
// streaming handlers.
func Interceptor(secret string) connect.Interceptor {
	return &interceptor{secret: secret}
}

type interceptor struct {
	secret string
}

func (i *interceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if publicProcedures[req.Spec().Procedure] {
			return next(ctx, req)
		}
		claims, err := authenticate(i.secret, req.Header().Get("Authorization"))
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
		return next(context.WithValue(ctx, claimsKey, claims), req)
	}
}

func (i *interceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i *interceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if publicProcedures[conn.Spec().Procedure] {
			return next(ctx, conn)
		}
		claims, err := authenticate(i.secret, conn.RequestHeader().Get("Authorization"))
		if err != nil {
			return connect.NewError(connect.CodeUnauthenticated, err)
		}
		return next(context.WithValue(ctx, claimsKey, claims), conn)
	}
}

func authenticate(secret, header string) (Claims, error) {
	token := strings.TrimPrefix(header, "Bearer ")
	if token == "" {
		return Claims{}, ErrNoToken
	}
	return Validate(secret, token)
}
