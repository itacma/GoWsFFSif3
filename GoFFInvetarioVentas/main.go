package main

import (
	"context" // Se usa ahora para el timeout de la consulta
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/sijms/go-ora/v2"
	"golang.org/x/crypto/bcrypt"
)

// --- ESTRUCTURAS ---
type CatSucursalFiscalResponse struct {
	SucursalesID   int     `json:"sucursales_id"`
	NombreFiscal   string  `json:"nombre_fiscal"`
	SucursalesIDSS *string `json:"sucursales_id_ss"` // Puntero por si es nulo
	CveFel         *string `json:"cve_fel"`          // Puntero por si es nulo
	NombreCto      string  `json:"nombre_cto"`
	TipoSucursal   *string `json:"tipo_sucursal"`
	StatusSendCdis *string `json:"status_send_cdis"`
	FechaIniVigSuc *string `json:"fecha_ini_vig_suc"`
	FechaFinVigSuc *string `json:"fecha_fin_vig_suc"`
}

type CatSucursalRequest struct {
	SucursalesID int `json:"sucursales_id"`
}

// Esta estructura recibirá el arreglo desde el JSON
type InventarioBulkRequest struct {
	Articulos []InventarioRequest `json:"articulos"`
}

// --- ESTRUCTURAS NUEVAS PARA INVENTARIO ---

type InventarioRequest struct {
	ArticuloID      string  `json:"articulo_id"`
	SucursalID      int     `json:"sucursal_id"`
	Existencias     int     `json:"existencias"`
	ExistenciaMin   int     `json:"existencia_min"`
	ExistenciaMax   int     `json:"existencia_max"`
	PuntoDeReorden  int     `json:"punto_de_reorden"`
	InvFisico       int     `json:"inv_fisico"`
	FechaUltMov     string  `json:"fecha_ult_mov"` // Se recibe como string para procesar formato
	FechaUltEnt     string  `json:"fecha_ult_ent"`
	FechaUltSal     string  `json:"fecha_ult_sal"`
	UltCosto        float64 `json:"ult_costo"`
	CostoProm       float64 `json:"costo_prom"`
	Peps            float64 `json:"peps"`
	Ueps            float64 `json:"ueps"`
	FechaInvFisico  string  `json:"fecha_inv_fisico"`
	Ubicacion       string  `json:"ubicacion"`
	ApartadoVtas    int     `json:"apartado_vtas"`
	UltUsuario      string  `json:"ult_usuario"`
	UltDocumento    string  `json:"ult_documento"`
	UltMovInvID     int     `json:"ult_mov_inv_id"`
	UltPartida      int     `json:"ult_partida"`
	UltFechaRefresh string  `json:"ult_fecha_refresh"`
}

type VentasDetalle struct {
	Partida           int     `json:"partida"`
	ArticuloID        string  `json:"articulo_id"`
	Existencia        int     `json:"existencia"`
	Fecha             string  `json:"fecha"`
	Entrada           int     `json:"entrada"`
	Salida            int     `json:"salida"`
	CostoProm         float64 `json:"costo_prom"`
	Costo             float64 `json:"costo"`
	UsuarioID         string  `json:"usuario_id"`
	KardexID          int     `json:"kardex_id"`
	PrecioCosto       float64 `json:"precio_costo"`
	PrecioVenta       float64 `json:"precio_venta"`
	Iva               float64 `json:"iva"`
	Importe_descuento float64 `json:"importe_descuento"`
}

type VentasRequest struct {
	TipoMovInvID     int             `json:"tipo_mov_inv_id"`
	SucursalID       int             `json:"sucursal_id"`
	Documento        string          `json:"documento"`
	FechaGenera      string          `json:"fecha_genera"`
	FechaCierre      string          `json:"fecha_cierre"`
	UsuarioGenera    int             `json:"usuario_genera"`
	Cliente          int             `json:"cliente"`
	TipoCliente      string          `json:"tipo_cliente"`
	Condiciones      string          `json:"condiciones"`
	Pcredito         string          `json:"pcredito"`
	Sif              string          `json:"sif"`
	Descuento        string          `json:"descuento"`
	ImporteDescuento string          `json:"importe_descuento"`
	ImporteTotal     float64         `json:"importe_total"`
	CajaID           int             `json:"caja_id"`
	CajeroID         string          `json:"cajero_id"`
	Detalle          []VentasDetalle `json:"detalle"`
}
type PedidoRequest struct {
	SucursalID int `json:"sucursal_id"`
}

type Pedido15dResponse struct {
	SucursalID           int     `json:"sucursal_id"`
	ArticuloID           string  `json:"articulo_id"`
	Nombre1              string  `json:"nombre1"`
	Nombre2              string  `json:"nombre2"`
	Tipo3                string  `json:"tipo3"`
	PrecioVenta          float64 `json:"precio_venta"`
	ApartadoVtas         int     `json:"apartado_vtas"`
	Existencias          int     `json:"existencias"`
	Faltante             float64 `json:"faltante"`
	ExistenciasCdis      float64 `json:"existencias_cdis"`
	ExistenciasTotales   float64 `json:"existencias_totales"`
	EoqSucursal          int     `json:"eoq_sucursal"`
	Ventas30d            float64 `json:"ventas_30d"`
	Ventas30dPromedioOrg float64 `json:"ventas_30d_promedio_org"`
	Pedido15d            float64 `json:"pedido_15d"`
	SugeridoOrigen       string  `json:"sugerido_origen"`
	DetalleExistencias   string  `json:"detalle_existencias"`
}

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

type FaltanteRequest struct {
	ArticuloID    string `json:"articulo_id"`
	Cantidad      int    `json:"cantidad"`
	SucursalID    int    `json:"sucursal_id"`
	UsuarioID     string `json:"usuario_id"`
	FolioFaltante string `json:"folio_faltante"`
	NombreUsuario string `json:"nombre_usuario"`
}

type RemisionFacturaRequest struct {
	SucursalID    int    `json:"sucursal_id"`
	UsuarioID     int    `json:"usuario_id"`
	Referencia    string `json:"referencia"`
	FolioFel      string `json:"folio_fel"`
	UUID          string `json:"uuid"`
	Tipo          string `json:"tipo"`
	UsoCfdi       string `json:"uso_cfdi"`
	RegimenFiscal int    `json:"regimen_fiscal"`
	TipoPago      string `json:"tipo_pago"`
	Refeacturar   string `json:"refeacturar"` // Nuevo campo agregado
}

type RemisionFacturaResponse struct {
	SucursalID    int    `json:"sucursal_id"`
	UsuarioID     int    `json:"usuario_id"`
	Referencia    string `json:"referencia"`
	Fecha         string `json:"fecha"`
	FolioFel      string `json:"folio_fel"`
	UUID          string `json:"uuid"`
	Tipo          string `json:"tipo"`
	UsoCfdi       string `json:"uso_cfdi"`
	RegimenFiscal int    `json:"regimen_fiscal"`
	TipoPago      string `json:"tipo_pago"`
	Refeacturar   string `json:"refeacturar"`
}

type RemisionConsultaRequest struct {
	SucursalID int     `json:"sucursal_id"`
	FolioFel   *string `json:"folio_fel"` // Puntero para manejar la opcionalidad
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

type CatPreExisResponse struct {
	ArticuloID     string   `json:"articulo_id"`
	Nombre1        string   `json:"nombre1"`
	Tipo1          *string  `json:"tipo1"`
	Tipo3          *string  `json:"tipo3"`
	Tipo6          *string  `json:"tipo6"`
	Tipo7          *string  `json:"tipo7"`
	Existencias    int      `json:"existencias"`
	PuntoDeReorden int      `json:"punto_de_reorden"`
	PrecioCosto    *float64 `json:"precio_costo"`     // Cambiado a puntero
	PrecioVenta    *float64 `json:"precio_venta"`     // Cambiado a puntero
	Iva            *float64 `json:"iva"`              // <--- ESTE CAUSABA EL ERROR
	PrecioVentaMax *float64 `json:"precio_venta_max"` // Cambiado a puntero
	CostoProm      *float64 `json:"costo_prom"`       // Cambiado a puntero
	TipoProv       *string  `json:"tipoprov"`
	ClienteID      *int     `json:"cliente_id"`
	NombreCliente  *string  `json:"nombre_cliente"`
	FechaUltEnt    *string  `json:"fecha_ult_ent"`
	// Existencias por sucursal
	Cap int `json:"cap"`
	Mat int `json:"mat"`
	Pop int `json:"pop"`
	Tec int `json:"tec"`
	Cho int `json:"cho"`
	Cen int `json:"cen"`
	Ben int `json:"ben"`
	Cu  int `json:"cu"`
	Gpe int `json:"gpe"`
	Ama int `json:"ama"`
	Aca int `json:"aca"`
	A50 int `json:"a50"`
	Smt int `json:"smt"`
	Ccu int `json:"ccu"`
	Gob int `json:"gob"`
	Sta int `json:"sta"`
	Izu int `json:"izu"`
	Uam int `json:"uam"`
	Tep int `json:"tep"`
	Hue int `json:"hue"`
	Alf int `json:"alf"`
	Atx int `json:"atx"`
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

// --- ESTRUCTURAS RENOMBRADAS ---

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

var (
	db     *sql.DB
	jwtKey = []byte("f7f1a3e5c9b2d8a4f6e0b1c2d3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2")
	puerto = ":54322"
)

func main() {
	//connStr := "oracle://FARMACIA:FAFL01@172.19.24.159:8087/f1s3"
	//connStr := "oracle://FARMACIA:cdispruebas@172.19.24.159:8087/cvpruebas"
	connStr := "oracle://FF_CONT:0PFA.robusto!23@201.165.55.62:8087/DATA2026"
	//connStr := "oracle://FF_CONT:Cl0ud.S1stemas$$1026@129.213.47.228:1521/DATA2021_PDB1.sub11092254140.vcnofaf.oraclevcn.com"
	//connStr := "oracle:// "
	GenerarHashPrueba()

	var err error
	db, err = sql.Open("oracle", connStr)

	db.SetMaxOpenConns(50) // Máximo de conexiones abiertas
	db.SetMaxIdleConns(10) // Conexiones en espera
	db.SetConnMaxLifetime(5 * time.Minute)

	if err != nil {
		log.Fatal("Error conexión:", err)
	}
	defer db.Close()

	http.HandleFunc("/api/health", healthCheckHandler) // NUEVO ENDPOINT
	http.HandleFunc("/api/inventarioSucursales", auditoriaMiddleware(validarJWT(inventarioSucursalesHandler)))
	http.HandleFunc("/api/ventasSucursalesDocumento", auditoriaMiddleware(validarJWT(ventasSucursalesDocumentoHandler)))
	http.HandleFunc("/api/reportes/Pedido15d", auditoriaMiddleware(validarJWT(pedido15dHandler)))
	http.HandleFunc("/api/cfaltante", auditoriaMiddleware(validarJWT(cfaltanteHandler)))
	http.HandleFunc("/api/cRemisionFacturas", auditoriaMiddleware(validarJWT(cRemisionFacturasHandler)))
	http.HandleFunc("/api/rRemisionFacturas", auditoriaMiddleware(validarJWT(rRemisionFacturasHandler)))
	http.HandleFunc("/api/rCatSucursalesFsical", auditoriaMiddleware(validarJWT(rCatSucursalesFiscalHandler)))
	http.HandleFunc("/api/gcatpreexis", auditoriaMiddleware(validarJWT(gcatpreexisHandler)))
	http.HandleFunc("/api/PermisosSIF3", PermisosSIF3Handler(db))
	http.HandleFunc("/api/LoginSIF3", LoginSIF3Handler(db))

	http.HandleFunc("/api/login", auditoriaMiddleware(loginHandler))

	log.Printf("Iniciando servicio en el puerto %s", puerto)
	if err := http.ListenAndServe(puerto, nil); err != nil {
		log.Fatalf("Error crítico al iniciar servidor: %v", err)
	}
}

// --- SEGURIDAD ---

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

	var dbPassword string
	query := "SELECT PASSWORD FROM FF_CONT.API_USUARIOS WHERE USUARIO = :1 AND ACTIVO = 'S'"
	err := db.QueryRow(query, creds.Usuario).Scan(&dbPassword)

	if err == sql.ErrNoRows || dbPassword != creds.Password {
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
	tokenString, _ := token.SignedString(jwtKey)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": tokenString})

}

func validarJWT(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenString := r.Header.Get("Authorization")
		//fmt.Println("Token recibido:", tokenString) // 🔍 Agregamos esto para depurar
		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}

		if tokenString == "" {
			http.Error(w, "Acceso denegado: Token faltante", http.StatusUnauthorized)
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Acceso denegado: Token inválido o expirado", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func ventasSucursalesDocumentoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		enviarRespuesta(w, 405, "Método no permitido")
		return
	}

	var req VentasRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		enviarRespuesta(w, 400, "JSON inválido: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	// Iniciamos la Transacción para asegurar integridad entre Cabecera y Detalle
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		enviarRespuesta(w, 500, "Error al iniciar transacción: "+err.Error())
		return
	}

	// 1. Insertar Cabecera
	queryCab := `INSERT INTO FF_CONT.CONS_VENTAS (
		TIPO_MOV_INV_ID, SUCURSAL_ID, DOCUMENTO, FECHA_GENERA, FECHA_CIERRE,
		USUARIO_GENERA, CLIENTE, TIPO_CLIENTE, CONDICIONES, PCREDITO,
		SIF, DESCUENTO, IMPORTE_DESCUENTO, IMPORTE_TOTAL, CAJA_ID, CAJERO_ID
	) VALUES (
		:1, :2, :3, TO_DATE(:4, 'YYYY-MM-DD HH24:MI:SS'), TO_DATE(:5, 'YYYY-MM-DD HH24:MI:SS'),
		:6, :7, :8, :9, :10, :11, :12, :13, :14, :15, :16
	)`

	_, err = tx.ExecContext(ctx, queryCab,
		req.TipoMovInvID, req.SucursalID, req.Documento, req.FechaGenera, req.FechaCierre,
		req.UsuarioGenera, req.Cliente, req.TipoCliente, req.Condiciones, req.Pcredito,
		req.Sif, req.Descuento, req.ImporteDescuento, req.ImporteTotal, req.CajaID, req.CajeroID,
	)

	if err != nil {
		tx.Rollback()
		log.Printf("Error Cabecera: %v", err)
		enviarRespuesta(w, 500, "Error en cabecera de venta")
		return
	}

	// 2. Insertar Detalle
	queryDet := `INSERT INTO FF_CONT.CONS_VENTAS_DET (
		TIPO_MOV_INV_ID, SUCURSAL_ID, DOCUMENTO, PARTIDA, ARTICULO_ID,
		EXISTENCIA, FECHA, ENTRADA, SALIDA, COSTO_PROM,
		COSTO, USUARIO_ID, KARDEX_ID, PRECIO_COSTO, PRECIO_VENTA, IVA, IMPORTE_DESCUENTO
	) VALUES (
		:1, :2, :3, :4, :5, :6, TO_DATE(:7, 'YYYY-MM-DD HH24:MI:SS'), :8, :9, :10, :11, :12, :13, :14, :15, :16, :17
	)`

	for _, d := range req.Detalle {
		_, err = tx.ExecContext(ctx, queryDet,
			req.TipoMovInvID, req.SucursalID, req.Documento, d.Partida, d.ArticuloID,
			d.Existencia, d.Fecha, d.Entrada, d.Salida, d.CostoProm,
			d.Costo, d.UsuarioID, d.KardexID, d.PrecioCosto, d.PrecioVenta, d.Iva,
			d.Importe_descuento,
		)
		if err != nil {
			tx.Rollback() // Si falla una partida, cancelamos todo el documento
			log.Printf("Error Partida %d: %v", d.Partida, err)
			enviarRespuesta(w, 500, "Error al insertar detalle de venta")
			return
		}
	}

	// Confirmar transacción
	if err := tx.Commit(); err != nil {
		enviarRespuesta(w, 500, "Error final al confirmar documento")
		return
	}

	enviarRespuesta(w, 1, "OK")
}

func inventarioSucursalesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		enviarRespuesta(w, 405, "Método no permitido")
		return
	}

	// Ahora decodificamos el BulkRequest (el arreglo)
	var bulkReq InventarioBulkRequest
	if err := json.NewDecoder(r.Body).Decode(&bulkReq); err != nil {
		enviarRespuesta(w, 400, "JSON inválido: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second) // Más tiempo para múltiples registros
	defer cancel()

	// Iniciamos transacción
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		enviarRespuesta(w, 500, "Error al iniciar transacción: "+err.Error())
		return
	}

	query := `
        MERGE INTO FF_CONT.INVENTARIO t
        USING (SELECT :1 as art, :2 as suc FROM dual) s
        ON (t.ARTICULO_ID = s.art AND t.SUCURSAL_ID = s.suc)
        WHEN MATCHED THEN
            UPDATE SET 
                EXISTENCIAS = :3, EXISTENCIA_MIN = :4, EXISTENCIA_MAX = :5,
                PUNTO_DE_REORDEN = :6, INV_FISICO = :7, ULT_COSTO = :8,
                COSTO_PROM = :9, PEPS = :10, UEPS = :11, UBICACION = :12,
                APARTADO_VTAS = :13, ULT_USUARIO = :14, ULT_DOCUMENTO = :15,
                ULT_MOV_INV_ID = :16, ULT_PARTIDA = :17, 
                FECHA_ULT_MOV = TO_DATE(:18, 'YYYY-MM-DD HH24:MI:SS'),
                FECHA_ULT_ENT = TO_DATE(:19, 'YYYY-MM-DD HH24:MI:SS'),
                FECHA_ULT_SAL = TO_DATE(:20, 'YYYY-MM-DD HH24:MI:SS'),
                FECHA_INV_FISICO = TO_DATE(:21, 'YYYY-MM-DD HH24:MI:SS'),
                ULT_FECHA_REFRESH = TO_DATE(:22, 'YYYY-MM-DD HH24:MI:SS')
        WHEN NOT MATCHED THEN
            INSERT (ARTICULO_ID, SUCURSAL_ID, EXISTENCIAS, EXISTENCIA_MIN, EXISTENCIA_MAX, 
                    PUNTO_DE_REORDEN, INV_FISICO, ULT_COSTO, COSTO_PROM, PEPS, UEPS, 
                    UBICACION, APARTADO_VTAS, ULT_USUARIO, ULT_DOCUMENTO, ULT_MOV_INV_ID, 
                    ULT_PARTIDA, FECHA_ULT_MOV, FECHA_ULT_ENT, FECHA_ULT_SAL, 
                    FECHA_INV_FISICO, ULT_FECHA_REFRESH)
            VALUES (:1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11, :12, :13, :14, :15, :16, :17, 
                    TO_DATE(:18, 'YYYY-MM-DD HH24:MI:SS'), TO_DATE(:19, 'YYYY-MM-DD HH24:MI:SS'), 
                    TO_DATE(:20, 'YYYY-MM-DD HH24:MI:SS'), TO_DATE(:21, 'YYYY-MM-DD HH24:MI:SS'), 
                    TO_DATE(:22, 'YYYY-MM-DD HH24:MI:SS'))`

	// Iteramos sobre el arreglo de artículos
	for _, req := range bulkReq.Articulos {
		_, err := tx.ExecContext(ctx, query,
			req.ArticuloID, req.SucursalID, req.Existencias, req.ExistenciaMin, req.ExistenciaMax,
			req.PuntoDeReorden, req.InvFisico, req.UltCosto, req.CostoProm, req.Peps, req.Ueps,
			req.Ubicacion, req.ApartadoVtas, req.UltUsuario, req.UltDocumento, req.UltMovInvID,
			req.UltPartida,
			req.FechaUltMov, req.FechaUltEnt, req.FechaUltSal, req.FechaInvFisico, req.UltFechaRefresh,
		)

		if err != nil {
			tx.Rollback() // Cancelamos todo si uno falla
			log.Printf("Error en Inventario (Articulo %s): %v", req.ArticuloID, err)
			enviarRespuesta(w, 500, "Error en artículo "+req.ArticuloID+": "+err.Error())
			return
		}
	}

	// Si todo salió bien, confirmamos
	if err := tx.Commit(); err != nil {
		enviarRespuesta(w, 500, "Error al confirmar la transacción")
		return
	}

	enviarRespuesta(w, 1, "OK")
}

// Función auxiliar para estandarizar la respuesta
func enviarRespuesta(w http.ResponseWriter, code int, mensaje string) {
	w.Header().Set("Content-Type", "application/json")
	// Si el código no es 1 (éxito), podrías querer cambiar el Status Code de HTTP
	if code != 1 && code < 600 {
		w.WriteHeader(http.StatusOK) // Mantenemos 200 pero enviamos el error en el body según tu requerimiento
	}
	json.NewEncoder(w).Encode(GenericResponse{
		Code:    code,
		Mensaje: mensaje,
	})
}

// Latido (Ping)
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "online",
		"server": "Go-Oracle-Ventas",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func pedido15dHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		enviarRespuesta(w, 405, "Método no permitido")
		return
	}

	var req PedidoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		enviarRespuesta(w, 400, "JSON inválido")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `SELECT 
                SUCURSAL_ID, ARTICULO_ID, NOMBRE1, NOMBRE2, TIPO3, 
                PRECIO_VENTA, APARTADO_VTAS, EXISTENCIAS, FALTANTE, 
                EXISTENCIAS_CDIS, EXISTENCIAS_TOTALES, EOQ_SUCURSAL, 
                VENTAS_30D, VENTAS_30D_PROMEDIO_ORG, PEDIDO_15D, 
                SUGERIDO_ORIGEN, DETALLE_EXISTENCIAS 
              FROM cs_inventario_pedido 
              WHERE sucursal_id = :1`

	rows, err := db.QueryContext(ctx, query, req.SucursalID)
	if err != nil {
		log.Printf("Error en consulta reporte: %v", err)
		enviarRespuesta(w, 500, "Error al consultar el reporte")
		return
	}
	defer rows.Close()

	var lista []Pedido15dResponse
	for rows.Next() {
		var p Pedido15dResponse
		// Usamos Scan para mapear las columnas a la estructura
		err := rows.Scan(
			&p.SucursalID, &p.ArticuloID, &p.Nombre1, &p.Nombre2, &p.Tipo3,
			&p.PrecioVenta, &p.ApartadoVtas, &p.Existencias, &p.Faltante,
			&p.ExistenciasCdis, &p.ExistenciasTotales, &p.EoqSucursal,
			&p.Ventas30d, &p.Ventas30dPromedioOrg, &p.Pedido15d,
			&p.SugeridoOrigen, &p.DetalleExistencias,
		)
		if err != nil {
			log.Printf("Error al escanear fila: %v", err)
			continue
		}
		lista = append(lista, p)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lista)
}

func cfaltanteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		enviarRespuesta(w, 405, "Método no permitido")
		return
	}

	var req FaltanteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		enviarRespuesta(w, 400, "JSON inválido: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// La consulta usa SYSDATE directamente en el VALUES
	query := `INSERT INTO FF_CONT.CONS_FALTANTE (
                FECHA, ARTICULO_ID, CANTIDAD, SUCURSAL_ID, 
                USUARIO_ID, FOLIO_FALTANTE, NOMBRE_USUARIO
              ) VALUES (
                SYSDATE, :1, :2, :3, :4, :5, :6
              )`

	_, err := db.ExecContext(ctx, query,
		req.ArticuloID, req.Cantidad, req.SucursalID,
		req.UsuarioID, req.FolioFaltante, req.NombreUsuario,
	)

	if err != nil {
		log.Printf("Error al insertar faltante: %v", err)
		enviarRespuesta(w, 500, "Error al registrar el faltante en la base de datos")
		return
	}

	enviarRespuesta(w, 1, "Faltante registrado correctamente")
}

func cRemisionFacturasHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		enviarRespuesta(w, 405, "Método no permitido")
		return
	}

	var req RemisionFacturaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		enviarRespuesta(w, 400, "JSON inválido: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Consulta actualizada con el campo REFEACTURAR
	query := `INSERT INTO FF_CONT.REMISION_FACTURAS (
                SUCURSAL_ID, USUARIO_ID, REFERENCIA, FECHA, 
                FOLIO_FEL, UUID, TIPO, USO_CFDI, 
                REGIMEN_FISCAL, TIPO_PAGO, REFEACTURAR
              ) VALUES (
                :1, :2, :3, SYSDATE, :4, :5, :6, :7, :8, :9, :10
              )`

	_, err := db.ExecContext(ctx, query,
		req.SucursalID, req.UsuarioID, req.Referencia,
		req.FolioFel, req.UUID, req.Tipo, req.UsoCfdi,
		req.RegimenFiscal, req.TipoPago, req.Refeacturar, // Parámetro 10
	)

	if err != nil {
		log.Printf("Error al insertar remisión factura: %v", err)
		enviarRespuesta(w, 500, "Error al registrar la remisión/factura")
		return
	}

	enviarRespuesta(w, 1, "Factura/Remisión registrada correctamente")
}

func rRemisionFacturasHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		enviarRespuesta(w, 405, "Método no permitido")
		return
	}

	var req RemisionConsultaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		enviarRespuesta(w, 400, "JSON inválido")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Lógica para el Folio opcional:
	// Si req.FolioFel es nil o cadena vacía, usamos un comodín o ignoramos el filtro
	var folioVal string
	if req.FolioFel != nil {
		folioVal = *req.FolioFel
	}

	// SQL Flexible:
	// Si folioVal está vacío, la segunda parte del WHERE se vuelve verdadera para todos
	query := `SELECT 
                SUCURSAL_ID, USUARIO_ID, REFERENCIA, TO_CHAR(FECHA, 'YYYY-MM-DD HH24:MI:SS'), 
                FOLIO_FEL, UUID, TIPO, USO_CFDI, 
                REGIMEN_FISCAL, TIPO_PAGO, REFEACTURAR 
              FROM FF_CONT.REMISION_FACTURAS 
              WHERE SUCURSAL_ID = :1 
              AND (:2 IS NULL OR FOLIO_FEL = :2)`

	// Si el string está vacío, pasamos nil al driver de Oracle para que se active el "IS NULL"
	var p2 interface{}
	if folioVal != "" {
		p2 = folioVal
	} else {
		p2 = nil
	}

	rows, err := db.QueryContext(ctx, query, req.SucursalID, p2)
	if err != nil {
		log.Printf("Error al consultar remisiones: %v", err)
		enviarRespuesta(w, 500, "Error en la consulta de base de datos")
		return
	}
	defer rows.Close()

	var lista []RemisionFacturaResponse
	for rows.Next() {
		var f RemisionFacturaResponse
		err := rows.Scan(
			&f.SucursalID, &f.UsuarioID, &f.Referencia, &f.Fecha,
			&f.FolioFel, &f.UUID, &f.Tipo, &f.UsoCfdi,
			&f.RegimenFiscal, &f.TipoPago, &f.Refeacturar,
		)
		if err != nil {
			log.Printf("Error escaner fila remisión: %v", err)
			continue
		}
		lista = append(lista, f)
	}

	if lista == nil {
		lista = []RemisionFacturaResponse{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lista)
}

func rCatSucursalesFiscalHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		enviarRespuesta(w, 405, "Método no permitido")
		return
	}

	var req CatSucursalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		enviarRespuesta(w, 400, "JSON inválido")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Consulta con las columnas exactas de tu tabla
	query := `SELECT 
                SUCURSALES_ID, 
                NOMBRE_FISCAL, 
                SUCURSALES_ID_SS, 
                CVE_FEL, 
                NOMBRE_CTO, 
                TIPO_SUCURSAL, 
                STATUS_SEND_CDIS,
                TO_CHAR(FECHA_INI_VIG_SUC, 'DD/MM/YYYY HH24:MI:SS'),
                TO_CHAR(FECHA_FIN_VIG_SUC, 'DD/MM/YYYY HH24:MI:SS')
              FROM FF_CONT.CAT_SUCURSALES_FISCAL
              WHERE (:1 = 1000 OR SUCURSALES_ID = :1)`

	rows, err := db.QueryContext(ctx, query, req.SucursalesID)
	if err != nil {
		log.Printf("Error al consultar CAT_SUCURSALES_FISCAL: %v", err)
		enviarRespuesta(w, 500, "Error en la consulta de base de datos")
		return
	}
	defer rows.Close()

	var lista []CatSucursalFiscalResponse
	for rows.Next() {
		var s CatSucursalFiscalResponse
		// Escaneamos las 9 columnas solicitadas
		err := rows.Scan(
			&s.SucursalesID,
			&s.NombreFiscal,
			&s.SucursalesIDSS,
			&s.CveFel,
			&s.NombreCto,
			&s.TipoSucursal,
			&s.StatusSendCdis,
			&s.FechaIniVigSuc,
			&s.FechaFinVigSuc,
		)
		if err != nil {
			log.Printf("Error escaner fila fiscal: %v", err)
			continue
		}
		lista = append(lista, s)
	}

	if lista == nil {
		lista = []CatSucursalFiscalResponse{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lista)
}

// Función interna para insertar el log
func guardarAuditoria(entry AuditEntry) {
	// El trigger en Oracle 11g llenará el ID automáticamente
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

		// Usamos la estructura global. Inicializamos con 200 por defecto.
		rw := &responseWriterWrapper{w, http.StatusOK}

		// Ejecutar el siguiente handler usando nuestro wrapper
		next(rw, r)

		// Extraer usuario del JWT...
		usuario := "ANONIMO"
		tokenString := r.Header.Get("Authorization")
		if len(tokenString) > 7 {
			tokenString = tokenString[7:]
			claims := &Claims{}
			// ParseWithClaims... (tu lógica actual)
			token, _ := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
				return jwtKey, nil
			})
			if token != nil && claims.Usuario != "" {
				usuario = claims.Usuario
			}
		}

		// Preparar entrada de auditoría
		entry := AuditEntry{
			Usuario:    usuario,
			Endpoint:   r.URL.Path,
			Metodo:     r.Method,
			IP:         r.RemoteAddr,
			StatusCode: rw.statusCode, // Aquí ya tendrá el código real (401, 500, etc.)
			Duracion:   time.Since(start).Milliseconds(),
		}

		go guardarAuditoria(entry)
	}
}

// Definición global del wrapper para capturar el Status Code
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

// Implementación del método WriteHeader para que intercepte el código
func (rw *responseWriterWrapper) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func gcatpreexisHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		enviarRespuesta(w, 405, "Método no permitido")
		return
	}

	// Timeout de 60 segundos porque la consulta involucra un DBLINK y muchos datos
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	query := `SELECT 
            ARTICULO_ID, NOMBRE1, TIPO1, TIPO3, TIPO6, TIPO7,
            EXISTENCIAS, PUNTO_DE_REORDEN, 
            NVL(PRECIO_COSTO, 0), NVL(PRECIO_VENTA, 0), NVL(IVA, 0), 
            NVL(PRECIO_VENTA_MAX, 0), NVL(COSTO_PROM, 0), 
            TIPOPROV, CIENTE_ID, NOMBRE_CLIENTE, FECHA_ULT_ENT,
            CAP, MAT, POP, TEC, CHO, CEN, BEN, CU,
            GPE, AMA, ACA, A50, SMT, CCU, GOB, STA,
            IZU, UAM, TEP, HUE, ALF, ATX
          FROM cs_cat_pre_api_go`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("Error al consultar gcatpreexis: %v", err)
		enviarRespuesta(w, 500, "Error al consultar vista de existencias")
		return
	}
	defer rows.Close()

	var lista []CatPreExisResponse = []CatPreExisResponse{}

	for rows.Next() {
		var c CatPreExisResponse
		err := rows.Scan(
			&c.ArticuloID, &c.Nombre1, &c.Tipo1, &c.Tipo3, &c.Tipo6, &c.Tipo7,
			&c.Existencias, &c.PuntoDeReorden, &c.PrecioCosto, &c.PrecioVenta,
			&c.Iva, &c.PrecioVentaMax, &c.CostoProm, &c.TipoProv, &c.ClienteID,
			&c.NombreCliente, &c.FechaUltEnt,
			&c.Cap, &c.Mat, &c.Pop, &c.Tec, &c.Cho, &c.Cen, &c.Ben, &c.Cu,
			&c.Gpe, &c.Ama, &c.Aca, &c.A50, &c.Smt, &c.Ccu, &c.Gob, &c.Sta,
			&c.Izu, &c.Uam, &c.Tep, &c.Hue, &c.Alf, &c.Atx,
		)
		if err != nil {
			log.Printf("Error escaner gcatpreexis: %v", err)
			continue
		}
		lista = append(lista, c)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lista)
}

// PermisosSIF3Handler procesa la validación activa de permisos bajo demanda utilizando el Número de Empleado
func PermisosSIF3Handler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Validar que sea un método POST
		if r.Method != http.MethodPost {
			http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
			return
		}

		var req PermisoRequest
		// Decodificar el body del request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON inválido en el request", http.StatusBadRequest)
			return
		}

		// Inicializar mapa de acciones con valores por defecto en falso
		acciones := map[string]bool{}

		// Query actualizado con el esquema "FARMACIA" y filtrado por numero_empleado
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

		// Escanear los resultados y mapear al diccionario booleano
		for rows.Next() {
			var accion string
			var permitido string
			if err := rows.Scan(&accion, &permitido); err == nil {
				acciones[accion] = (permitido == "S")
			}
		}

		// Construir la respuesta final estructurada con el número de empleado
		res := PermisoResponse{
			NumeroEmpleado:     req.NumeroEmpleado,
			Modulo:             req.Modulo,
			SubModulo:          req.SubModulo,
			AccionesPermitidas: acciones,
		}

		// Responder al Frontend Web
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(res)
	}
}

// --- HANDLER ACTUALIZADO ---

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

		// Query limpio para Oracle (Esquema FF_CONT y marcadores :1)
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

		// VALIDACIÓN DE BLOQUEO TEMPORAL
		if u.FecDispIngreso.Valid && time.Now().Before(u.FecDispIngreso.Time) {
			minutosRestantes := int(time.Until(u.FecDispIngreso.Time).Minutes()) + 1
			w.WriteHeader(http.StatusForbidden)
			msg := fmt.Sprintf("Tu cuenta está bloqueada temporalmente por exceso de intentos erróneos. Intenta de nuevo en %d minutos.", minutosRestantes)
			json.NewEncoder(w).Encode(LoginResponseSif3{Success: false, Message: msg})
			return
		}

		// VALIDACIÓN DE CONTRASEÑA BCRYPT
		errPass := bcrypt.CompareHashAndPassword([]byte(u.ContrasenaHash), []byte(req.Contrasena))
		if errPass != nil {
			nuevosErrores := u.IngresosErroneos + 1
			var queryUpdateError string
			var errUpd error

			if nuevosErrores >= 3 {
				// Bloqueo por 15 minutos en Oracle (15/1440)
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

		// VALIDACIÓN VIGENCIA CONTRASEÑA
		if time.Now().After(u.FechaVigenciaPassw) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(LoginResponseSif3{Success: false, Message: "Tu contraseña ha expirado. Es necesario actualizarla."})
			return
		}

		// LOGIN EXITOSO -> RESET
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

// Llama a esto dentro de tu func main() solo una vez para ver el resultado en la terminal
func GenerarHashPrueba() {
	//passwordPlano := "default"
	passwordPlano := "4L3XAND3R"
	hash, err := bcrypt.GenerateFromPassword([]byte(passwordPlano), bcrypt.DefaultCost)
	if err != nil {
		log.Println("Error al generar hash:", err)
		return
	}
	fmt.Println("--------------------------------------------------")
	fmt.Printf("El hash Bcrypt para '%s' es:\n%s\n", passwordPlano, string(hash))
	fmt.Println("--------------------------------------------------")
}
