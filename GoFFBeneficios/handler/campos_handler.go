package handler

import (
	"encoding/json"
	"net/http"

	"github.com/ffleming/api-beneficios/domain"
)

func (h *PromotionHandler) CamposCampanaHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req domain.CamposCampanaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "JSON de entrada inválido: " + err.Error()})
		return
	}

	if req.ListaReqCampoID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "LISTA_REQ_CAMPO_ID debe ser mayor a 0"})
		return
	}

	campos, err := h.repo.ObtenerCamposCampana(r.Context(), req.ListaReqCampoID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Error en base de datos al obtener campos",
			"detalle": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(campos)
}
