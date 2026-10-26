package middleware

import (
	"net/http"
	"os"

	"github.com/ffleming/api-beneficios/domain"
	"github.com/golang-jwt/jwt/v5"
)

// GetJWTSecret obtiene la clave secreta desde las variables de entorno
func GetJWTSecret() []byte {
	secret := os.Getenv("API_JWT_SECRET_KEY")
	if secret == "" {
		panic("JWT_SECRET_KEY no esta configurada en el sistema")
	}
	return []byte(secret)
}

func ValidarJWT(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}

		if tokenString == "" {
			http.Error(w, "Acceso denegado: Token faltante", http.StatusUnauthorized)
			return
		}

		claims := &domain.Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
			return GetJWTSecret(), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Acceso denegado: Token inválido o expirado", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

// responseWriterWrapper captura el código de estado HTTP para auditoría
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
