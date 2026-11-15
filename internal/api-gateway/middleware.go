package api_gateway

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v4"
)

func jsonError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write([]byte(message))

}
func AuthMiddleware(jwtSecret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Del("X-User-Id")
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			jsonError(w, `{"error": "missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			jsonError(w, `{"error": "invalid authorization format"}`, http.StatusUnauthorized)
			return
		}
		tokenString := parts[1]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			jsonError(w, `{"error": "invalid or expired token"}`, http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			jsonError(w, `{"error": "invalid token claims"}`, http.StatusUnauthorized)
			return
		}

		userIDFloat, ok := claims["user_id"].(float64)
		if !ok {
			jsonError(w, `{"error": "user_id not found in token"}`, http.StatusUnauthorized)
			return
		}

		userIDStr := fmt.Sprintf("%d", int64(userIDFloat))
		r.Header.Set("X-User-Id", userIDStr)

		next.ServeHTTP(w, r)
	})
}
