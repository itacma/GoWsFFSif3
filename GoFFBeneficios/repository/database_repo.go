package repository

import (
	"context"
	"database/sql"
	"fmt"
	"math"

	"strings"
	"time"

	"github.com/ffleming/api-beneficios/domain" // Importa tus structs globales
)

// PromotionRepository define los contratos que los handlers mandarán a llamar
type PromotionRepository interface {
	GetCupon(ctx context.Context, folio string) (string, int, error)
	UpdateCuponStatus(ctx context.Context, folio, activo, ref string) (bool, error)
	EvaluateArticuloPromotions(ctx context.Context, sucursalID int, artID string, cantidad int) (domain.PromocionesVentaResponse, error)

	GetPasswordHash(ctx context.Context, usuario string) (string, error)
	ObtenerCamposCampana(ctx context.Context, reqCampoID int) ([]domain.CampoCampanaResponse, error)
}

type oracleRepo struct {
	db *sql.DB
}

// NewOraclePromotionRepository inyecta la conexión a la base de datos
func NewOraclePromotionRepository(db *sql.DB) PromotionRepository {
	return &oracleRepo{db: db}
}

// GetCupon ejecuta la consulta del cupón en la capa de datos basándose en el folio de tipo string.
// Retorna el estado activo del cupón ('S', 'G', 'P'), el ID de la promoción asociada y un error en caso de fallo.
// GetCupon consulta la vigencia de un cupón forzando el tratamiento estricto de strings en Oracle 19c
func (r *oracleRepo) GetCupon(ctx context.Context, folio string) (string, int, error) {
	// Explicación técnica: TO_CHAR(:1) bloquea la conversión implícita del driver a número,
	// obligando a Oracle a comparar cadenas de texto idénticas con sus ceros a la izquierda.
	query := `SELECT activo, LISTA_OFT_ID 
	          FROM FF_CONT.LISTA_OFERTAS_CUPONES 
	          WHERE folio_cupon = TO_CHAR(:1) AND ROWNUM = 1`

	var activo string
	var listOftID int

	// Ejecución segura controlada bajo el ciclo de vida del contexto
	err := r.db.QueryRowContext(ctx, query, folio).Scan(&activo, &listOftID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Retornamos error controlado nativo si no hay registros coincidentes
			return "", 0, err
		}
		return "", 0, fmt.Errorf("error al consultar el folio cupon %s en la base de datos cloud: %w", folio, err)
	}

	return activo, listOftID, nil
}

// UpdateCuponStatus cambia el estado (S, G, P) del folio
// UpdateCuponStatus ejecuta el UPDATE en Oracle sobre la tabla de cupones.
// Retorna true si la fila fue actualizada con éxito o false si el folio no existe.
func (r *oracleRepo) UpdateCuponStatus(ctx context.Context, folio, activo, ref string) (bool, error) {
	query := `UPDATE FF_CONT.LISTA_OFERTAS_CUPONES 
	          SET ACTIVO = :1, 
	              REFERENCIA = :2, 
	              ULT_ACTUALIZACION = SYSDATE 
	          WHERE TRIM(FOLIO_CUPON) = TRIM(:3)`

	res, err := r.db.ExecContext(ctx, query, activo, ref, folio)
	if err != nil {
		return false, fmt.Errorf("error al actualizar estado del cupon %s en Oracle: %w", folio, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("error al verificar filas afectadas para el cupon %s: %w", folio, err)
	}

	// Si rowsAffected == 0, significa que el folio no existe en la base de datos
	return rowsAffected > 0, nil
}

// redondear2Decimale evita errores de flotantes de IEEE 754 en precios de Oracle
func redondear2Decimales(valor float64) float64 {
	return math.Round(valor*100) / 100
}

func (r *oracleRepo) EvaluateArticuloPromotions(ctx context.Context, sucursalID int, artID string, cantidad int) (domain.PromocionesVentaResponse, error) {
	var resp domain.PromocionesVentaResponse
	resp.ArticuloID = strings.TrimSpace(artID)
	resp.SucursalID = sucursalID
	resp.CamposFaltantes = []domain.CampoRequeridoFront{}

	// 1. Consulta de la oferta activa en la tabla LISTA_OFERTAS_VENTA_DET
	var viewPrecios string
	var listOftID, cantidadReq, beneficioCant, listaOftPkgID int
	var descuento, importeBeneficio, importePacto float64

	var nombreOfertaNull sql.NullString
	var reqCampoIDNull sql.NullInt64

	queryPromo := `
		SELECT det.LISTA_OFT_ID, NVL(det.CANTIDAD, 1), NVL(det.BENEFICIO_CANTIDAD, 0), det.DESCUENTO, 
		       det.IMPORTE_BENEFICIO, det.IMPORTE_PACTO, det.LISTA_OFT_PKG_ID, 
		       TRIM(cab.view_precios), cab.NOMBRE, cab.LISTA_REQ_CAMPO_ID
		FROM FF_CONT.LISTA_OFERTAS_VENTA_DET det
		INNER JOIN FF_CONT.LISTA_OFERTAS_VENTA cab ON det.LISTA_OFT_ID = cab.LISTA_OFT_ID
		INNER JOIN FF_CONT.LISTA_OFERTAS_VENTA_SUCURSAL suc ON cab.LISTA_OFT_ID = suc.LISTA_OFT_ID
		WHERE TRIM(det.ARTICULO_ID) = TRIM(:1) 
		  AND suc.SUCURSAL_ID = :2 
		  AND det.ACTIVO = 'S'
		  AND TRUNC(SYSDATE) BETWEEN TRUNC(det.FECHA_INICIO) AND TRUNC(det.FECHA_FIN)
		  AND ROWNUM = 1`

	err := r.db.QueryRowContext(ctx, queryPromo, resp.ArticuloID, sucursalID).Scan(
		&listOftID, &cantidadReq, &beneficioCant, &descuento,
		&importeBeneficio, &importePacto, &listaOftPkgID,
		&viewPrecios, &nombreOfertaNull, &reqCampoIDNull,
	)

	if err == sql.ErrNoRows {
		viewPrecios = "CS_LP_0"
		listOftID = 0
		cantidadReq = 1
	} else if err != nil {
		return resp, fmt.Errorf("error al buscar promociones para SKU %s: %w", resp.ArticuloID, err)
	} else {
		if nombreOfertaNull.Valid {
			resp.NombreOferta = nombreOfertaNull.String
		}
		if reqCampoIDNull.Valid {
			resp.IDListaReqCampo = int(reqCampoIDNull.Int64)
			if resp.IDListaReqCampo > 0 {
				resp.RequiereCapturaCampos = true
			}
		}
	}

	resp.ListaOftID = listOftID

	// 2. Consulta de lista de precios base en Oracle[cite: 1]
	var fechaBajaNull sql.NullString
	var precioOrgNull sql.NullFloat64

	queryPrecios := fmt.Sprintf(`
		SELECT PRECIO_VENTA, PRECIO_VENTA_MAX, PRECIO_VENTA_FARM, PRECIO_COSTO, 
		       COSTO_PROM, COSTO_ULT_COMPRA, TO_CHAR(FECHA_ALTA, 'YYYY-MM-DD HH24:MI:SS'), 
		       TO_CHAR(FECHA_BAJA, 'YYYY-MM-DD HH24:MI:SS'),
		       ACTIVO, PRECIO_ORG_VENTA, IMP_COMPRA, IMP_VENTA
		FROM FF_CONT.%s 
		WHERE TRIM(ARTICULO_ID) = TRIM(:1) AND SUCURSAL_ID = :2 AND ROWNUM = 1`, viewPrecios)

	err = r.db.QueryRowContext(ctx, queryPrecios, resp.ArticuloID, sucursalID).Scan(
		&resp.PrecioVenta, &resp.PrecioVentaMax, &resp.PrecioVentaFarm, &resp.PrecioCosto,
		&resp.CostoProm, &resp.CostoUltCompra, &resp.FechaAlta, &fechaBajaNull,
		&resp.Activo, &precioOrgNull, &resp.ImpCompra, &resp.ImpVenta,
	)

	// Fallback a la tabla general LISTA_PRECIOS[cite: 1]
	if err == sql.ErrNoRows {
		queryRespaldo := `
			SELECT PRECIO_VENTA, PRECIO_VENTA_MAX, PRECIO_VENTA_MAX as farm, PRECIO_COSTO, 
			       COSTO_PROM, COSTO_PROM as ult_c, TO_CHAR(FECHA_ALTA, 'YYYY-MM-DD HH24:MI:SS'), 
			       TO_CHAR(FECHA_BAJA, 'YYYY-MM-DD HH24:MI:SS'),
			       ACTIVO, PRECIO_ORG_VENTA, IMP_COMPRA, IMP_VENTA
			FROM FF_CONT.LISTA_PRECIOS 
			WHERE TRIM(ARTICULO_ID) = TRIM(:1) AND SUCURSAL_ID = :2 AND ROWNUM = 1`

		err = r.db.QueryRowContext(ctx, queryRespaldo, resp.ArticuloID, sucursalID).Scan(
			&resp.PrecioVenta, &resp.PrecioVentaMax, &resp.PrecioVentaFarm, &resp.PrecioCosto,
			&resp.CostoProm, &resp.CostoUltCompra, &resp.FechaAlta, &fechaBajaNull,
			&resp.Activo, &precioOrgNull, &resp.ImpCompra, &resp.ImpVenta,
		)
		if err != nil {
			return resp, fmt.Errorf("el articulo %s no existe en catalogos para sucursal %d: %w", resp.ArticuloID, sucursalID, err)
		}
	} else if err != nil {
		return resp, fmt.Errorf("error de lectura en la vista de precios %s: %w", viewPrecios, err)
	}

	if precioOrgNull.Valid {
		resp.PrecioOrgVenta = precioOrgNull.Float64
	} else {
		resp.PrecioOrgVenta = resp.PrecioVenta
	}

	if fechaBajaNull.Valid {
		resp.FechaBaja = fechaBajaNull.String
	} else {
		resp.FechaBaja = "null"
	}

	// =========================================================================
	// 3. MOTOR ALGORTÍMICO CON CANDADO DE SEGURIDAD PARA IMPORTE PACTADO
	// =========================================================================
	precioBaseUnitario := resp.PrecioVenta
	piezasTotales := cantidad
	if piezasTotales <= 0 {
		piezasTotales = 1
	}

	var importeFinalPartida float64
	var importeDescuentoPartida float64

	// 🔒 CANDADO DE SEGURIDAD:
	// Para aplicar IMPORTE_PACTO, este DEBE ser estrictamente menor al PRECIO_VENTA unitario.
	// Si importePacto >= precioBaseUnitario, se ignora la oferta por ser perjudicial/inválida.
	esPactoValido := listOftID > 0 && importePacto > 0 && importePacto < precioBaseUnitario

	// CASO A: Oferta activa por IMPORTE_PACTO VÁLIDO y cantidad suficiente
	if esPactoValido && cantidadReq > 0 && piezasTotales >= cantidadReq {
		numPaquetes := piezasTotales / cantidadReq
		piezasExcedentes := piezasTotales % cantidadReq

		// Paquetes a precio pactado + excedentes a precio normal base
		costoPaquetes := float64(numPaquetes) * (importePacto * float64(cantidadReq))
		costoExcedentes := float64(piezasExcedentes) * precioBaseUnitario

		importeFinalPartida = costoPaquetes + costoExcedentes
		montoSinDescuento := float64(piezasTotales) * precioBaseUnitario
		importeDescuentoPartida = montoSinDescuento - importeFinalPartida

		// Promedio unitario efectivo
		resp.PrecioConDescuento = redondear2Decimales(importeFinalPartida / float64(piezasTotales))
		resp.ImporteDescuento = redondear2Decimales(importeDescuentoPartida / float64(piezasTotales))

		// CASO B: Descuento directo en porcentaje (%)
	} else if listOftID > 0 && descuento > 0 {
		resp.PorcentajeDescuento = descuento
		resp.ImporteDescuento = redondear2Decimales(precioBaseUnitario * (descuento / 100.0))
		resp.PrecioConDescuento = precioBaseUnitario - resp.ImporteDescuento

		importeDescuentoPartida = resp.ImporteDescuento * float64(piezasTotales)
		importeFinalPartida = (precioBaseUnitario * float64(piezasTotales)) - importeDescuentoPartida

		// CASO C: Descuento directo por importe fijo unitario (IMPORTE_BENEFICIO)
	} else if listOftID > 0 && importeBeneficio > 0 && importeBeneficio < precioBaseUnitario {
		resp.ImporteDescuento = importeBeneficio
		resp.PrecioConDescuento = precioBaseUnitario - importeBeneficio

		importeDescuentoPartida = resp.ImporteDescuento * float64(piezasTotales)
		importeFinalPartida = (precioBaseUnitario * float64(piezasTotales)) - importeDescuentoPartida

		// CASO D: Sin oferta, importe pactado inválido (candado activado) o cantidad insuficiente
	} else {
		resp.PrecioConDescuento = precioBaseUnitario
		resp.ImporteDescuento = 0.0
		resp.PorcentajeDescuento = 0.0
		importeDescuentoPartida = 0.0
		importeFinalPartida = precioBaseUnitario * float64(piezasTotales)
	}

	// 4. Cálculo de IVA Unitario y Total de Partida
	if resp.ImpVenta == 2 {
		resp.IvaConDescuento = redondear2Decimales((resp.PrecioConDescuento / 1.16) * 0.16)
		resp.IvaSinDescuento = redondear2Decimales((precioBaseUnitario / 1.16) * 0.16)
	} else {
		resp.IvaConDescuento = 0.0
		resp.IvaSinDescuento = 0.0
	}

	// 5. Asignación de Valores Finales en la Respuesta
	resp.ImporteDescuentoTotp = redondear2Decimales(importeDescuentoPartida)
	resp.IvaSinDescuentoTotp = redondear2Decimales(resp.IvaSinDescuento * float64(piezasTotales))
	resp.ImporteFinalTotp = redondear2Decimales(importeFinalPartida)

	return resp, nil
}

// GetPasswordHash consulta la credencial con logs de depuración para diagnóstico
func (r *oracleRepo) GetPasswordHash(ctx context.Context, usuario string) (string, error) {
	ctxChild, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	var dbPassword string

	// Forzamos TRIM para limpiar padding (tipo CHAR) y UPPER para evitar problemas de casing
	query := `
		SELECT TRIM(PASSWORD) 
		FROM FF_CONT.API_USUARIOS 
		WHERE UPPER(USUARIO) = UPPER(:1) AND ACTIVO = 'S'`

	usuarioLimpio := strings.TrimSpace(usuario)
	err := r.db.QueryRowContext(ctxChild, query, usuarioLimpio).Scan(&dbPassword)

	if err != nil {
		if err == sql.ErrNoRows {
			//		log.Printf("[DEBUG DB] Usuario NO encontrado en FF_CONT.API_USUARIOS: '%s' (o ACTIVO != 'S')", usuarioLimpio)
			return "", sql.ErrNoRows
		}
		//	log.Printf("[DEBUG DB ERROR]: Fallo al ejecutar QueryRowContext: %v", err)
		return "", fmt.Errorf("error al consultar credenciales en Oracle: %w", err)
	}

	//log.Printf("[DEBUG DB SUCCESS] Usuario encontrado: '%s'. Clave recuperada de DB (longitud %d)", usuarioLimpio, len(dbPassword))
	return dbPassword, nil
}
