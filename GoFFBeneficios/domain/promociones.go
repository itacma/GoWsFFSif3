package domain

import "github.com/golang-jwt/jwt/v5"

type Credenciales struct {
	Usuario  string `json:"usuario"`
	Password string `json:"password"`
}

type Claims struct {
	Usuario string `json:"usuario"`
	jwt.RegisteredClaims
}

type GenericResponse struct {
	Code    int    `json:"code"`
	Mensaje string `json:"mensaje"`
}

type CuponVigenteRequest struct {
	Cupon string `json:"cupon"`
}

type CuponVigenteResponse struct {
	CuponVigente bool   `json:"cupon_vigente"`
	Mensaje      string `json:"mensaje"`
}

type CambiaActivoCuponRequest struct {
	Cupon      string `json:"cupon"`
	Activo     string `json:"activo"`
	Referencia string `json:"referencia"`
}

type CambiaActivoCuponResponse struct {
	CambioEstado bool   `json:"cambio-estado"`
	Mensaje      string `json:"mensaje"`
}

type ArticuloVenta struct {
	ArticuloID string `json:"articulo_id"`
	Cantidad   int    `json:"cantidad"`
}

type PromocionesVentaRequest struct {
	SucursalID int             `json:"sucursal_id"`
	Articulos  []ArticuloVenta `json:"articulos"`
}

// CampoRequeridoFront define las propiedades solicitadas por LISTA_REQ_CAMPO
type CampoRequeridoFront struct {
	CampoNombre string `json:"campo_nombre"` // NOMBRE, VIGENCIA, TIPO
	CampoTipo   string `json:"campo_tipo"`   // STRING, DATETIME, COMBOBOX
	Requerido   string `json:"requerido"`    // S o N
}

// ArticuloBeneficio mapea los SKUs de regalo
type ArticuloBeneficio struct {
	ArticuloID string `json:"articulo_id"`
	Cantidad   int    `json:"cantidad"`
}

// BeneficioPaquete estructura la metadata en caso de LISTA_OFT_PKG_ID > 0
type BeneficioPaquete struct {
	PaqueteID                int                 `json:"paquete_id"`
	NombrePaquete            string              `json:"nombre_paquete"`
	LogicaNegocioBnf         string              `json:"logica_negocio_bnf"`        // '&' o '|'
	LogicaNegocioBnfPrecio   string              `json:"logica_negocio_bnf_precio"` // '0' o '1'
	ArticulosBeneficioRegalo []ArticuloBeneficio `json:"articulos_beneficio_regalo,omitempty"`
}

// PromocionesResponseWrapper encapsula la meta-información de la oferta y el array de artículos
type PromocionesResponseWrapper struct {
	NombreOferta       string                     `json:"nombre"`
	RequiereCampos     int                        `json:"requiere_Campos"` // Usamos int para representar el ID de lista_req_campo_id
	ArticulosCotizados []PromocionesVentaResponse `json:"articulos"`
}

// PromocionesVentaResponse es el modelo corregido con TODOS los campos de la vista
type PromocionesVentaResponse struct {
	ArticuloID          string  `json:"articulo_id"`
	SucursalID          int     `json:"sucursal_id"`
	ListaOftID          int     `json:"lista_oft_id"`                 // Campo añadido
	NombreOferta        string  `json:"nombre_oferta,omitempty"`      // Mapeado desde cab.NOMBRE
	IDListaReqCampo     int     `json:"id_lista_req_campo,omitempty"` // Mapeado desde cab.LISTA_REQ_CAMPO_ID
	PrecioVenta         float64 `json:"precio_venta"`
	PrecioVentaMax      float64 `json:"precio_venta_max"`  // <-- FALTABA
	PrecioVentaFarm     float64 `json:"precio_venta_farm"` // <-- FALTABA
	PrecioCosto         float64 `json:"precio_costo"`      // <-- FALTABA
	CostoProm           float64 `json:"costo_prom"`        // <-- FALTABA
	CostoUltCompra      float64 `json:"costo_ult_compra"`  // <-- FALTABA
	FechaAlta           string  `json:"fecha_alta"`        // <-- FALTABA
	FechaBaja           string  `json:"fecha_baja"`        // <-- FALTABA
	ImpCompra           int     `json:"imp_compra"`        // <-- FALTABA
	ImpVenta            int     `json:"imp_venta"`         // <-- FALTABA
	Activo              string  `json:"activo"`            // <-- FALTABA
	PrecioOrgVenta      float64 `json:"precio_org_venta"`
	PorcentajeDescuento float64 `json:"porcentaje_descuento"`
	ImporteDescuento    float64 `json:"importe_descuento"`
	PrecioConDescuento  float64 `json:"precio_con_descuento"`
	IvaConDescuento     float64 `json:"ivacondescuento"`
	IvaSinDescuento     float64 `json:"ivasindescuento"`

	// --- NUEVOS CAMPOS POR PARTIDA MASIVA ---
	ImporteDescuentoTotp float64 `json:"importe_descuento_totp"`
	IvaSinDescuentoTotp  float64 `json:"ivasindescuento_totp"`
	ImporteFinalTotp     float64 `json:"importefinal_totp"`

	// CONTROL DE FLUJO FRONT-END
	RequiereCapturaCampos bool                  `json:"requiere_captura_campos"`
	CamposFaltantes       []CampoRequeridoFront `json:"campos_faltantes,omitempty"`
	TienePaqueteRegalo    bool                  `json:"tiene_paquete_regalo"`
	DetallePaquete        *BeneficioPaquete     `json:"detalle_paquete,omitempty"`
}

// Request para obtener campos de campaña
type CamposCampanaRequest struct {
	ListaOftID      int `json:"lista_oft_id"`
	ListaReqCampoID int `json:"LISTA_REQ_CAMPO_ID"`
}

// Response del detalle del campo
type CampoCampanaResponse struct {
	ListaReqCampoID int    `json:"lista_req_campo_id"`
	CampoNombre     string `json:"campo_nombre"`
	CampoTipo       string `json:"campo_tipo"`
	Requerido       string `json:"requerido"` // CHAR(1) -> "S" o "N"
	Valor           string `json:"valor"`     // Opciones delimitadas por '|' si es COMBOBOX
}
