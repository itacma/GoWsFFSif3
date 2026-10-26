package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	//"os"
	"time"

	"github.com/ffleming/api-beneficios/domain"
	"github.com/ffleming/api-beneficios/repository"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type PromotionHandler struct {
	repo repository.PromotionRepository
}

func NewPromotionHandler(repo repository.PromotionRepository) *PromotionHandler {
	return &PromotionHandler{repo: repo}
}

func (h *PromotionHandler) CuponVigenteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req domain.CuponVigenteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON Inválido", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	activo, _, err := h.repo.GetCupon(ctx, req.Cupon)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		json.NewEncoder(w).Encode(domain.CuponVigenteResponse{CuponVigente: false, Mensaje: "Cupón no encontrado"})
		return
	}

	res := domain.CuponVigenteResponse{CuponVigente: activo == "S", Mensaje: "Estatus: " + activo}
	json.NewEncoder(w).Encode(res)
}

// LoginHandler valida las credenciales y emite el Token JWT de acceso corporativo
/*func (h *PromotionHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var creds domain.Credenciales
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "JSON con sintaxis inválida"})
		return
	}

	if creds.Usuario == "" || creds.Password == "" {
		http.Error(w, "Usuario y contraseña son obligatorios", http.StatusBadRequest)
		return
	}

	// Contexto dinámico con límite de tiempo (Timeout de 10 segundos para blindar Oracle)
	// Contexto dinámico con límite de tiempo (Timeout de 10 segundos para blindar Oracle)
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// CORRECCIÓN: Eliminamos la línea "query :=" que causaba el error de no uso.
	// El controlador le solicita directamente la información procesada a la capa de persistencia.
	dbPasswordHash, err := h.repo.GetPasswordHash(ctx, creds.Usuario)
	if err == sql.ErrNoRows {
		http.Error(w, "Credenciales inválidas o usuario inactivo", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Printf("Error en Base de Datos durante Login: %v", err)
		http.Error(w, "Error interno de base de datos", http.StatusInternalServerError)
		return
	}

	// Validación criptográfica robusta mediante Bcrypt
	errPass := bcrypt.CompareHashAndPassword([]byte(dbPasswordHash), []byte(creds.Password))
	if errPass != nil {
		http.Error(w, "Credenciales inválidas", http.StatusUnauthorized)
		return
	}

	// Generación y firma del Token JWT
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &domain.Claims{
		Usuario: creds.Usuario,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	secretKey := os.Getenv("JWT_SECRET_KEY")
	if secretKey == "" {
		secretKey = "f7f1a3e5c9b2d8a4f6e0b1c2d3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2"
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, errToken := token.SignedString([]byte(secretKey))
	if errToken != nil {
		http.Error(w, "Error al generar el token de seguridad", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"token":   tokenString,
		"expires": expirationTime.Format(time.RFC3339),
	})
}
*/
/*
func (h *PromotionHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var creds domain.Credenciales
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Sintaxis JSON de entrada inválida"})
		return
	}

	if creds.Usuario == "" || creds.Password == "" {
		http.Error(w, "El usuario y la contraseña son obligatorios", http.StatusBadRequest)
		return
	}

	// 1. Obtención de la clave guardada en Oracle mediante la capa Repository
	dbPassword, err := h.repo.GetPasswordHash(r.Context(), creds.Usuario)
	if err == sql.ErrNoRows {
		http.Error(w, "Credenciales inválidas o usuario inactivo", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Printf("[ERROR LOGIN DB]: %v", err)
		http.Error(w, "Error interno de base de datos", http.StatusInternalServerError)
		return
	}

	// 2. Validación Híbrida: Bcrypt + Fallback a Texto Plano
	contrasenaValida := false

	// Evaluación A: Intentar validación por Hash Bcrypt (Como en puerto 54301)
	if errBcrypt := bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(creds.Password)); errBcrypt == nil {
		contrasenaValida = true
	} else {
		// Evaluación B: Fallback si la clave en Oracle aún está en texto plano
		if dbPassword == creds.Password {
			contrasenaValida = true
		}
	}

	if !contrasenaValida {
		http.Error(w, "Credenciales inválidas", http.StatusUnauthorized)
		return
	}

	// 3. Generación y Firma del JWT
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &domain.Claims{
		Usuario: creds.Usuario,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	secretKey := os.Getenv("JWT_SECRET_KEY")
	if secretKey == "" {
		secretKey = "f7f1a3e5c9b2d8a4f6e0b1c2d3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2"
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, errToken := token.SignedString([]byte(secretKey))
	if errToken != nil {
		http.Error(w, "Error al generar el token JWT", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"token":   tokenString,
		"expires": expirationTime.Format(time.RFC3339),
	})
}
*/

// Clave JWT Secreta fija e inmutable
var jwtKeyEstatica = []byte("f7f1a3e5c9b2d8a4f6e0b1c2d3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2")

func (h *PromotionHandler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var creds domain.Credenciales
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Sintaxis JSON de entrada inválida"})
		return
	}

	usuarioInput := strings.TrimSpace(creds.Usuario)
	passInput := strings.TrimSpace(creds.Password)

	if usuarioInput == "" || passInput == "" {
		http.Error(w, "El usuario y la contraseña son obligatorios", http.StatusBadRequest)
		return
	}

	// 1. Obtener la clave desde el repositorio
	dbPassword, err := h.repo.GetPasswordHash(r.Context(), usuarioInput)
	if err == sql.ErrNoRows {
		http.Error(w, "Credenciales inválidas", http.StatusUnauthorized)
		return
	} else if err != nil {
		log.Printf("[ERROR CRÍTICO DB]: %v", err)
		http.Error(w, "Error interno de base de datos", http.StatusInternalServerError)
		return
	}

	// 2. Proceso de Validación
	contrasenaValida := false

	// Evaluación A: Bcrypt (Para hashes encriptados)
	if errBcrypt := bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(passInput)); errBcrypt == nil {
		contrasenaValida = true
		log.Printf("[DEBUG AUTH] Autenticación Exitosa vía Bcrypt Hash para '%s'", usuarioInput)
	} else {
		// Evaluación B: Comparación directa en texto plano (Plaintext)
		dbPassLimpia := strings.TrimSpace(dbPassword)
		if dbPassLimpia == passInput {
			contrasenaValida = true
			log.Printf("[DEBUG AUTH] Autenticación Exitosa vía Texto Plano (Plaintext) para '%s'", usuarioInput)
		} else {
			log.Printf("[DEBUG AUTH FAIL] La clave proporcionada ('%s') NO coincide con la clave en BD ('%s')", passInput, dbPassLimpia)
		}
	}

	if !contrasenaValida {
		http.Error(w, "Credenciales inválidas", http.StatusUnauthorized)
		return
	}

	// 3. Generación del Token JWT
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &domain.Claims{
		Usuario: usuarioInput,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, errToken := token.SignedString(jwtKeyEstatica)
	if errToken != nil {
		http.Error(w, "Error al generar el token JWT", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"token":   tokenString,
		"expires": expirationTime.Format(time.RFC3339),
	})
}

// PromocionesVentaHandler procesa la tasación masiva de artículos y evalúa sus beneficios aplicables
func (h *PromotionHandler) PromocionesVentaHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req domain.PromocionesVentaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "JSON inválido"})
		return
	}

	// Extraemos la sucursal del JSON dinámico (Fallback a Sucursal 1 en caso de omitirse)
	sucursalID := req.SucursalID
	if sucursalID <= 0 {
		sucursalID = 1
	}

	// Inicializamos el Wrapper Global de Respuesta
	wrapper := domain.PromocionesResponseWrapper{
		ArticulosCotizados: make([]domain.PromocionesVentaResponse, 0),
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	for _, art := range req.Articulos {
		// Asignación correcta: 2 valores de retorno
		resArt, err := h.repo.EvaluateArticuloPromotions(ctx, sucursalID, art.ArticuloID, art.Cantidad)
		if err != nil {
			log.Printf("[Motor-Promociones] Fallo crítico para SKU %s en Sucursal %d: %v", art.ArticuloID, sucursalID, err)
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error":   "Fallo interno en la vista de precios de Oracle",
				"detalle": err.Error(),
			})
			return
		}

		// Extraemos la metadata de cabecera del primer artículo con oferta válida
		if resArt.ListaOftID > 0 && wrapper.NombreOferta == "" {
			wrapper.NombreOferta = resArt.NombreOferta
			wrapper.RequiereCampos = resArt.IDListaReqCampo
		}

		wrapper.ArticulosCotizados = append(wrapper.ArticulosCotizados, resArt)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(wrapper)
}

// HealthCheckHandler responde con el estado actual del servicio para pruebas de latido (Ping)
func (h *PromotionHandler) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(map[string]string{
		"status": "online",
		"server": "Go-Oracle-Ventas",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// CambiaActivoCuponHandler permite actualizar el estatus operacional de un cupón (ej. pasar de 'S' a 'G' o 'P')
func (h *PromotionHandler) CambiaActivoCuponHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.enviarRespuestaAux(w, 405, "Método no permitido")
		return
	}

	var req domain.CambiaActivoCuponRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.enviarRespuestaAux(w, 400, "JSON inválido: "+err.Error())
		return
	}

	// Validaciones primarias de campos obligatorios
	if req.Cupon == "" {
		h.enviarRespuestaAux(w, 400, "El campo 'cupon' es requerido")
		return
	}
	if req.Activo == "" {
		h.enviarRespuestaAux(w, 400, "El campo 'activo' es requerido")
		return
	}

	// Contexto con Timeout de 10 segundos para la transacción en Oracle
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Ejecución de la actualización en la capa de datos
	actualizado, err := h.repo.UpdateCuponStatus(ctx, req.Cupon, req.Activo, req.Referencia)
	if err != nil {
		log.Printf("[Handler-Cupones] Error en UpdateCuponStatus para cupón %s: %v", req.Cupon, err)
		h.enviarRespuestaAux(w, 500, "Error interno al actualizar la base de datos")
		return
	}

	if !actualizado {
		h.enviarRespuestaAux(w, 0, "Cupón no encontrado")
		return
	}

	// Éxito
	h.enviarRespuestaAux(w, 1, "Estatus de cupón actualizado correctamente")
}

// enviarRespuestaAux es un método privado para formatear y enviar respuestas JSON estandarizadas
func (h *PromotionHandler) enviarRespuestaAux(w http.ResponseWriter, code int, mensaje string) {
	w.Header().Set("Content-Type", "application/json")

	// Mantenemos StatusOK (200) para desacoplar códigos de error de negocio de la capa HTTP
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(domain.GenericResponse{
		Code:    code,
		Mensaje: mensaje,
	})
}
