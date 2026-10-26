package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	_ "github.com/sijms/go-ora/v2"
	"golang.org/x/crypto/bcrypt"
)

// --- ESTRUCTURAS ---

type GenericResponse struct {
	Code    int    `json:"code"`
	Mensaje string `json:"mensaje"`
}

type Credenciales struct {
	Usuario  string `json:"usuario"`
	Password string `json:"password"`
}

type Claims struct {
	Usuario string `json:"usuario"`
	jwt.RegisteredClaims
}

// Estructura para la auditoría
type AuditEntry struct {
	Usuario    string
	Endpoint   string
	Metodo     string
	IP         string
	StatusCode int
	Duracion   int64
}

// PermisoRequest mapea el nuevo JSON de entrada basado en el número de empleado
type PermisoRequest struct {
	NumeroEmpleado string `json:"numero_empleado"`
	Modulo         string `json:"modulo"`
	SubModulo      string `json:"sub_modulo"`
}

// PermisoResponse mapea el JSON de salida hacia la interfaz web
type PermisoResponse struct {
	NumeroEmpleado     string          `json:"numero_empleado"`
	Modulo             string          `json:"modulo"`
	SubModulo          string          `json:"sub_modulo"`
	AccionesPermitidas map[string]bool `json:"acciones_permitidas"`
}

type LoginRequestSif3 struct {
	NumeroEmpleado string `json:"numero_empleado"`
	Contrasena     string `json:"contrasena_hash"`
}

type LoginResponseSif3 struct {
	Success        bool   `json:"success"`
	Message        string `json:"message"`
	NumeroEmpleado string `json:"numero_empleado,omitempty"`
	NombreCompleto string `json:"nombre_completo,omitempty"`
}

// /Ejecucion de bases de datos jmqc 13/07/2026
// TargetDatabase define la conexión de una BD destino
type TargetDatabase struct {
	ID     string `json:"id"`
	Nombre string `json:"nombre"`
	DNS    string `json:"dns"` // oracle://user:pass@host:port/service
}

// QueryExecutionRequest es el payload recibido por la API central
type QueryExecutionRequest struct {
	Query     string   `json:"query"`      // El comando SQL o bloque PL/SQL Anónimo a ejecutar
	TargetIDs []string `json:"target_ids"` // Lista de IDs de BDs donde se ejecutará (vacío = Todas)
}

// ExecutionResult consolida el output detallado por cada nodo On-Premise
type ExecutionResult struct {
	TargetID     string                   `json:"target_id"`
	Nombre       string                   `json:"nombre"`
	FilasAfectas int64                    `json:"filas_afectadas"`
	Estado       string                   `json:"estado"` // "SUCCESS" o "ERROR"
	Detalle      string                   `json:"detalle"`
	Data         []map[string]interface{} `json:"data,omitempty"` // Null si no es un SELECT
}

var db *sql.DB

// Obtiene la clave secreta directamente de las variables de entorno
func getJWTSecret() []byte {
	secret := os.Getenv("API_JWT_SECRET_KEY")
	if secret == "" {
		// Valor por defecto en caso de no encontrarse en el sistema
		secret = "f7f1a3e5c9b2d8a4f6e0b1c2d3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2"
	}
	return []byte(secret)
}

func main() {
	// Intentar cargar .env si existe localmente; si no, toma las del sistema Windows
	//	_ = godotenv.Load()

	// 1. Cadena de conexión desde variables de entorno
	connStr := os.Getenv("API_APP_ORCL_DB_DATA2021")
	if connStr == "" {
		// Fallback por si la variable de entorno no está dada de alta
		connStr = "oracle://FF_CONT:Cl0ud.S1stemas$$0426@129.213.47.228:1521/DATA2021_PDB1.sub11092254140.vcnofaf.oraclevcn.com"
	}

	GenerarHashPrueba()

	var err error
	db, err = sql.Open("oracle", connStr)
	if err != nil {
		log.Fatal("Error conexión:", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(50) // Máximo de conexiones abiertas
	db.SetMaxIdleConns(10) // Conexiones en espera
	db.SetConnMaxLifetime(5 * time.Minute)

	// Validar comunicación con la BD
	if err := db.Ping(); err != nil {
		log.Fatalf("Error de comunicación (Ping) con la BD Oracle: %v", err)
	}

	// 2. Puerto dinámico desde variables de entorno
	puerto := os.Getenv("API_APP_PORT_SEGURIDAD")
	if puerto == "" {
		puerto = ":54301"
	} else if !strings.HasPrefix(puerto, ":") {
		puerto = ":" + puerto
	}

	// --- REGISTRO DE ENDPOINTS CONSERVADOS ---
	http.HandleFunc("/api/health", healthCheckHandler)
	http.HandleFunc("/api/PermisosSIF3", auditoriaMiddleware(validarJWT(PermisosSIF3Handler(db))))
	http.HandleFunc("/api/LoginSIF3", auditoriaMiddleware(validarJWT(LoginSIF3Handler(db))))
	http.HandleFunc("/api/execute-query", auditoriaMiddleware(validarJWT(ejecutarQueryHandler)))
	http.HandleFunc("/api/login", auditoriaMiddleware(loginHandler))

	log.Printf("Iniciando servicio en el puerto %s", puerto)
	if err := http.ListenAndServe(puerto, nil); err != nil {
		log.Fatalf("Error crítico al iniciar servidor: %v", err)
	}
}

// --- SEGURIDAD Y JWT ---

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var creds Credenciales
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var dbPasswordHash string
	query := "SELECT PASSWORD FROM FF_CONT.API_USUARIOS WHERE USUARIO = :1 AND ACTIVO = 'S'"
	err := db.QueryRow(query, creds.Usuario).Scan(&dbPasswordHash)

	if err == sql.ErrNoRows {
		http.Error(w, "Credenciales inválidas", http.StatusUnauthorized)
		return
	} else if err != nil {
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
		return
	}

	errPass := bcrypt.CompareHashAndPassword([]byte(dbPasswordHash), []byte(creds.Password))
	if errPass != nil {
		http.Error(w, "Credenciales inválidas", http.StatusUnauthorized)
		return
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Usuario: creds.Usuario,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(getJWTSecret())
	if err != nil {
		http.Error(w, "Error al firmar token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": tokenString})
}

func validarJWT(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		if len(tokenString) > 7 && strings.HasPrefix(tokenString, "Bearer ") {
			tokenString = tokenString[7:]
		}

		if tokenString == "" {
			http.Error(w, "Acceso denegado: Token faltante", http.StatusUnauthorized)
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
			return getJWTSecret(), nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Acceso denegado: Token inválido o expirado", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

// --- UTILERÍAS Y MIDDLEWARES ---

func enviarRespuesta(w http.ResponseWriter, code int, mensaje string) {
	w.Header().Set("Content-Type", "application/json")
	if code != 1 && code < 600 {
		w.WriteHeader(http.StatusOK)
	}
	json.NewEncoder(w).Encode(GenericResponse{
		Code:    code,
		Mensaje: mensaje,
	})
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "online",
		"server": "Go-Oracle-Ventas",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func guardarAuditoria(entry AuditEntry) {
	query := `INSERT INTO FF_CONT.API_AUDITORIA 
              (USUARIO, ENDPOINT, METODO, IP_CLIENTE, STATUS_CODE, DURACION_MS) 
              VALUES (:1, :2, :3, :4, :5, :6)`

	_, err := db.Exec(query,
		entry.Usuario, entry.Endpoint, entry.Metodo,
		entry.IP, entry.StatusCode, entry.Duracion)

	if err != nil {
		log.Printf("Error al guardar auditoría en 11g: %v", err)
	}
}

func auditoriaMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseWriterWrapper{w, http.StatusOK}

		next(rw, r)

		usuario := "ANONIMO"
		tokenString := r.Header.Get("Authorization")
		if len(tokenString) > 7 && strings.HasPrefix(tokenString, "Bearer ") {
			tokenString = tokenString[7:]
			claims := &Claims{}
			token, _ := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
				return getJWTSecret(), nil
			})
			if token != nil && claims.Usuario != "" {
				usuario = claims.Usuario
			}
		}

		entry := AuditEntry{
			Usuario:    usuario,
			Endpoint:   r.URL.Path,
			Metodo:     r.Method,
			IP:         r.RemoteAddr,
			StatusCode: rw.statusCode,
			Duracion:   time.Since(start).Milliseconds(),
		}

		go guardarAuditoria(entry)
	}
}

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// --- HANDLERS ACTIVOS DE SIF3 ---

func PermisosSIF3Handler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
			return
		}

		var req PermisoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON inválido en el request", http.StatusBadRequest)
			return
		}

		acciones := map[string]bool{}

		query := `
    SELECT ac.NOMBRE_ACCION, pr.PERMITIDO
    FROM FF_CONT.CAT_USUARIOS u
    INNER JOIN FF_CONT.SIF3_PERMISOS_ROLES pr ON u.ROL_ID = pr.ROL_ID
    INNER JOIN FF_CONT.SIF3_MODULOS_SUBMODULOS ms ON pr.MODULO_SUBMODULO_ID = ms.MODULO_SUBMODULO_ID
    INNER JOIN FF_CONT.SIF3_ACCIONES ac ON pr.ACCION_ID = ac.ACCION_ID
    WHERE u.NUMERO_EMPLEADO = :1 
      AND LOWER(ms.MODULO) = LOWER(:2) 
      AND LOWER(ms.SUB_MODULO) = LOWER(:3)
      AND u.ACTIVO = 'S'
`
		rows, err := db.Query(query, req.NumeroEmpleado, req.Modulo, req.SubModulo)
		if err != nil {
			http.Error(w, "Error interno al consultar permisos: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var accion string
			var permitido string
			if err := rows.Scan(&accion, &permitido); err == nil {
				acciones[accion] = (permitido == "S")
			}
		}

		res := PermisoResponse{
			NumeroEmpleado:     req.NumeroEmpleado,
			Modulo:             req.Modulo,
			SubModulo:          req.SubModulo,
			AccionesPermitidas: acciones,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(res)
	}
}

func LoginSIF3Handler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(w).Encode(LoginResponseSif3{Success: false, Message: "Método no permitido"})
			return
		}

		var req LoginRequestSif3
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(LoginResponseSif3{Success: false, Message: "JSON inválido"})
			return
		}

		var u struct {
			Nombre             string
			Apellidos          string
			ContrasenaHash     string
			IngresosErroneos   int
			FecDispIngreso     sql.NullTime
			FechaVigenciaPassw time.Time
			Activo             string
		}

		querySelect := `
            SELECT NOMBRE, APELLIDOS, CONTRASENA_HASH, INGRESOS_ERRONEOS, FEC_DISP_INGRESO, FECHA_VIGENCIA_PASSW, ACTIVO
            FROM FF_CONT.CAT_USUARIOS
            WHERE NUMERO_EMPLEADO = :1
        `
		err := db.QueryRow(querySelect, req.NumeroEmpleado).Scan(
			&u.Nombre, &u.Apellidos, &u.ContrasenaHash, &u.IngresosErroneos, &u.FecDispIngreso, &u.FechaVigenciaPassw, &u.Activo,
		)

		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(LoginResponseSif3{Success: false, Message: "Credenciales incorrectas"})
			return
		} else if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(LoginResponseSif3{Success: false, Message: "Error al consultar usuario: " + err.Error()})
			return
		}

		if u.Activo != "S" {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(LoginResponseSif3{Success: false, Message: "El usuario se encuentra inactivo"})
			return
		}

		if u.FecDispIngreso.Valid && time.Now().Before(u.FecDispIngreso.Time) {
			minutosRestantes := int(time.Until(u.FecDispIngreso.Time).Minutes()) + 1
			w.WriteHeader(http.StatusForbidden)
			msg := fmt.Sprintf("Tu cuenta está bloqueada temporalmente por exceso de intentos erróneos. Intenta de nuevo en %d minutos.", minutosRestantes)
			json.NewEncoder(w).Encode(LoginResponseSif3{Success: false, Message: msg})
			return
		}

		errPass := bcrypt.CompareHashAndPassword([]byte(u.ContrasenaHash), []byte(req.Contrasena))
		if errPass != nil {
			nuevosErrores := u.IngresosErroneos + 1
			var queryUpdateError string
			var errUpd error

			if nuevosErrores >= 3 {
				queryUpdateError = `
                    UPDATE FF_CONT.CAT_USUARIOS 
                    SET INGRESOS_ERRONEOS = :1, FEC_DISP_INGRESO = SYSDATE + (15/1440) 
                    WHERE NUMERO_EMPLEADO = :2
                `
				_, errUpd = db.Exec(queryUpdateError, nuevosErrores, req.NumeroEmpleado)
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(LoginResponseSif3{Success: false, Message: "Contraseña incorrecta. Has alcanzado el límite de intentos. Cuenta bloqueada por 15 minutos."})
			} else {
				queryUpdateError = `
                    UPDATE FF_CONT.CAT_USUARIOS 
                    SET INGRESOS_ERRONEOS = :1 
                    WHERE NUMERO_EMPLEADO = :2
                `
				_, errUpd = db.Exec(queryUpdateError, nuevosErrores, req.NumeroEmpleado)
				w.WriteHeader(http.StatusUnauthorized)
				msg := fmt.Sprintf("Contraseña incorrecta. Intentos restantes: %d", 3-nuevosErrores)
				json.NewEncoder(w).Encode(LoginResponseSif3{Success: false, Message: msg})
			}

			if errUpd != nil {
				log.Printf("Error al actualizar intentos fallidos: %v", errUpd)
			}
			return
		}

		if time.Now().After(u.FechaVigenciaPassw) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(LoginResponseSif3{Success: false, Message: "Tu contraseña ha expirado. Es necesario actualizarla."})
			return
		}

		querySuccess := `
            UPDATE FF_CONT.CAT_USUARIOS 
            SET INGRESOS_ERRONEOS = 0, FEC_DISP_INGRESO = NULL 
            WHERE NUMERO_EMPLEADO = :1
        `
		_, errReset := db.Exec(querySuccess, req.NumeroEmpleado)
		if errReset != nil {
			log.Printf("Error al reiniciar contadores de login: %v", errReset)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(LoginResponseSif3{
			Success:        true,
			Message:        "Autenticación exitosa",
			NumeroEmpleado: req.NumeroEmpleado,
			NombreCompleto: u.Nombre + " " + u.Apellidos,
		})
	}
}

func GenerarHashPrueba() {
	passwordPlano := "5619"
	hash, err := bcrypt.GenerateFromPassword([]byte(passwordPlano), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Error al generar hash:", err)
		return
	}
	fmt.Println("--------------------------------------------------")
	fmt.Printf("El hash Bcrypt para '%s' es:\n%s\n", passwordPlano, string(hash))
	fmt.Println("--------------------------------------------------")
}

// --- SIMULACIÓN DE CANALES DE NOTIFICACIÓN ---

func enviarNotificacionWhatsApp(mensaje string) {
	log.Printf("[NOTIFICACIÓN WHATSAPP] Enviada: %s", mensaje)
}

func enviarNotificacionEmail(mensaje string) {
	log.Printf("[NOTIFICACIÓN EMAIL] Enviada: %s", mensaje)
}

// --- HANDLER DE EJECUCIÓN DINÁMICA de bases de datos---
func ejecutarQueryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "Método no permitido"})
		return
	}

	var req QueryExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido en el request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Query) == "" {
		http.Error(w, "El campo 'query' es requerido y no puede estar vacío", http.StatusBadRequest)
		return
	}

	databases := []TargetDatabase{
		{
			ID:     "F2S3",
			Nombre: "Scursal CAPU",
			DNS:    "oracle://FARMACIA:FAFL02@201.165.55.66:1521/F2S3",
		},
		{
			ID:     "FARMACIA",
			Nombre: "Scursal matriz",
			DNS:    "oracle://FARMACIA:OPERADOR@172.19.24.2:1521/FARMACIA",
		},
		{
			ID:     "F4S3",
			Nombre: "Scursal Popular",
			DNS:    "oracle://FARMACIA:AFO04@201.165.55.78:1521/F4S3",
		},
		{
			ID:     "F5S3",
			Nombre: "Scursal Tecamachalco",
			DNS:    "oracle://FARMACIA:FAFL05@201.165.55.38:1521/F5S3",
		},
		{
			ID:     "F6S3",
			Nombre: "Scursal cholula",
			DNS:    "oracle://FARMACIA:201308@201.165.55.18:1521/F6S3",
		},
		{
			ID:     "F7S3",
			Nombre: "Scursal Centro",
			DNS:    "oracle://FARMACIA:FAFL07@172.19.32.3:8087/F7S3",
		},
		{
			ID:     "F8S3",
			Nombre: "Scursal Benito",
			DNS:    "oracle://FARMACIA:f8s3@172.25.36.2:1521/f8s3",
		},
		{
			ID:     "F9S3",
			Nombre: "Scursal CU",
			DNS:    "oracle://FARMACIA:f9s3@148.228.105.22:8087/f9s3",
		},
		{
			ID:     "f10s3",
			Nombre: "Scursal amalucan",
			DNS:    "oracle://FARMACIA:f11s3@201.165.55.70:1521/f11s3",
		},
		{
			ID:     "f11s3",
			Nombre: "Scursal matriz",
			DNS:    "oracle://FARMACIA:OPERADOR@172.19.24.2:1521/FARMACIA",
		},
		{
			ID:     "f12s3",
			Nombre: "Scursal acatzingo",
			DNS:    "oracle://FARMACIA/FAFL12@201.165.55.34:1521/F12S3",
		},
	}

	var targetsToExecute []TargetDatabase
	if len(req.TargetIDs) > 0 {
		for _, target := range databases {
			for _, id := range req.TargetIDs {
				if target.ID == id {
					targetsToExecute = append(targetsToExecute, target)
				}
			}
		}
	} else {
		targetsToExecute = databases
	}

	resultsChan := make(chan ExecutionResult, len(targetsToExecute))

	for _, target := range targetsToExecute {
		go func(dbInfo TargetDatabase) {
			res := ExecutionResult{
				TargetID: dbInfo.ID,
				Nombre:   dbInfo.Nombre,
				Estado:   "SUCCESS",
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			remoteDB, err := sql.Open("oracle", dbInfo.DNS)
			if err != nil {
				res.Estado = "ERROR"
				res.Detalle = fmt.Sprintf("Error crítico de red/conexión: %v", err)
				resultsChan <- res
				return
			}
			defer remoteDB.Close()

			cleanQuery := strings.ToUpper(strings.TrimSpace(req.Query))

			if strings.HasPrefix(cleanQuery, "SELECT") {
				rows, err := remoteDB.QueryContext(ctx, req.Query)
				if err != nil {
					res.Estado = "ERROR"
					res.Detalle = fmt.Sprintf("Oracle Error (QueryContext): %v", err)
					resultsChan <- res
					return
				}
				defer rows.Close()

				columns, err := rows.Columns()
				if err != nil {
					res.Estado = "ERROR"
					res.Detalle = fmt.Sprintf("Error al mapear metadatos de columnas: %v", err)
					resultsChan <- res
					return
				}

				var dataset []map[string]interface{}

				for rows.Next() {
					scanArgs := make([]interface{}, len(columns))
					values := make([]interface{}, len(columns))
					for i := range values {
						scanArgs[i] = &values[i]
					}

					if err := rows.Scan(scanArgs...); err != nil {
						log.Printf("Error al escanear fila en %s: %v", dbInfo.ID, err)
						continue
					}

					rowMap := make(map[string]interface{})
					for i, colName := range columns {
						val := values[i]

						if bytesVal, ok := val.([]byte); ok {
							rowMap[colName] = string(bytesVal)
						} else {
							rowMap[colName] = val
						}
					}
					dataset = append(dataset, rowMap)
				}

				if err := ctx.Err(); err == context.DeadlineExceeded {
					res.Estado = "ERROR"
					res.Detalle = "La consulta SELECT fue abortada: Límite de tiempo excedido (Timeout)."
					resultsChan <- res
					return
				}

				res.Data = dataset
				res.FilasAfectas = int64(len(dataset))
				res.Detalle = "Consulta SELECT ejecutada y serializada con éxito."

			} else {
				result, err := remoteDB.ExecContext(ctx, req.Query)
				if err != nil {
					res.Estado = "ERROR"
					res.Detalle = fmt.Sprintf("Oracle Error (ExecContext): %v", err)
					resultsChan <- res
					return
				}

				rowsAffected, err := result.RowsAffected()
				if err == nil {
					res.FilasAfectas = rowsAffected
					res.Detalle = fmt.Sprintf("Instrucción DML ejecutada. Filas impactadas: %d", rowsAffected)
				} else {
					res.Detalle = "Bloque PL/SQL o sentencia DDL procesada de manera correcta."
				}
			}

			resultsChan <- res
		}(target)
	}

	var finalResponse []ExecutionResult
	for i := 0; i < len(targetsToExecute); i++ {
		resultadoNode := <-resultsChan
		finalResponse = append(finalResponse, resultadoNode)
		despacharAlertasAsincronas(resultadoNode)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(finalResponse)
}

func despacharAlertasAsincronas(res ExecutionResult) {
	go func(r ExecutionResult) {
		mensaje := fmt.Sprintf("⚠️ Alerta BD [%s] (%s) - Estado: %s.\nDetalle: %s\nRegistros procesados: %d",
			r.Nombre, r.TargetID, r.Estado, r.Detalle, r.FilasAfectas)

		log.Printf("[EMAIL/WHATSAPP NOTIFICACIÓN]: %s", mensaje)
	}(res)
}
