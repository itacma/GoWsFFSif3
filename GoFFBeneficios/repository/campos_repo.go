package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ffleming/api-beneficios/domain"
)

func (r *oracleRepo) ObtenerCamposCampana(ctx context.Context, reqCampoID int) ([]domain.CampoCampanaResponse, error) {
	campos := make([]domain.CampoCampanaResponse, 0)

	// Contexto con timeout explícito
	ctxChild, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	query := `
		SELECT LISTA_REQ_CAMPO_ID, 
		       NVL(CAMPO_NOMBRE, ''), 
		       NVL(CAMPO_TIPO, 'STRING'), 
		       NVL(REQUERIDO, 'N'), 
		       NVL(VALOR, '')
		FROM FF_CONT.LISTA_REQ_CAMPO
		WHERE LISTA_REQ_CAMPO_ID = :1`

	rows, err := r.db.QueryContext(ctxChild, query, reqCampoID)
	if err != nil {
		return nil, fmt.Errorf("error al consultar LISTA_REQ_CAMPO: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var c domain.CampoCampanaResponse
		var valorNull sql.NullString

		err := rows.Scan(
			&c.ListaReqCampoID,
			&c.CampoNombre,
			&c.CampoTipo,
			&c.Requerido,
			&valorNull,
		)
		if err != nil {
			return nil, fmt.Errorf("error al escanear campo de campaña: %w", err)
		}

		if valorNull.Valid {
			c.Valor = valorNull.String
		}

		campos = append(campos, c)
	}

	return campos, nil
}
