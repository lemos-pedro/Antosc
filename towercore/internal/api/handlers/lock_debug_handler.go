// internal/api/handlers/lock_debug_handler.go
//
// TEMPORÁRIO — usar só para descobrir os valores reais de "sno" por site,
// depois de preenchido o HIZIMA_STATION_MAP com os pares corretos.
// Remover este ficheiro e a rota correspondente depois de completares o mapeamento.
package handlers

import (
	"net/http"

	"towercore/internal/core/interfaces"
	"towercore/pkg/apierror"
)

// GetAllStations atende GET /api/v1/hizima/debug/stations
// Lista TODOS os sites/locks que a conta Hizima consegue ver, sem filtro,
// para conseguires ler o "sno" e "stationName" reais de cada um.
func (h *LockHandler) GetAllStations(w http.ResponseWriter, r *http.Request) {
	result, err := h.Client.GetLockStatus(r.Context(),
		interfaces.LockStatusFilter{}, // sem filtro — devolve tudo o que a conta vê
		interfaces.PageRequest{Current: 1, Size: 200},
	)
	if err != nil {
		apierror.Write(w, http.StatusBadGateway, "internal_error", "failed to fetch stations from Hizima: "+err.Error())
		return
	}

	// Resposta simplificada só com o essencial para montares o HIZIMA_STATION_MAP
	type stationInfo struct {
		StationNo      string `json:"stationNo"`
		StationName    string `json:"stationName"`
		CustomerSiteID string `json:"customerSiteId"`
		LockName       string `json:"lockName"`
		Region         string `json:"region"`
	}

	out := make([]stationInfo, 0, len(result.Records))
	for _, lock := range result.Records {
		out = append(out, stationInfo{
			StationNo:      lock.StationNo,
			StationName:    lock.StationName,
			CustomerSiteID: lock.CustomerSiteID,
			LockName:       lock.LockName,
			Region:         lock.RegionPath[len(lock.RegionPath)-1],
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"total_found": result.Total,
		"stations":    out,
	})
}