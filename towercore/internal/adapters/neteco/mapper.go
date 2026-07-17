package neteco

import (
	"fmt"

	"towercore/internal/core/interfaces"
)

func mapBatteryStatus(raw map[string]interface{}) (*interfaces.BatteryStatus, error) {
	data, ok := raw["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("campo 'data' ausente ou inválido na resposta")
	}

	siteName, _ := data["siteName"].(string)
	siteDn, _ := data["siteDn"].(string)
	connectStatus, _ := data["siteConnectStatus"].(string)

	status := &interfaces.BatteryStatus{
		SiteDn:        siteDn,
		SiteName:      siteName,
		ConnectStatus: connectStatus,
		SOC:           -1,
		SOH:           -1,
	}

	groups, ok := data["batGroupSohDataList"].([]interface{})
	if !ok || len(groups) == 0 {
		// Site sem grupo de bateria reportado — devolve status base sem valores
		return status, nil
	}

	// Usa o primeiro grupo de bateria (comportamento a validar/ajustar
	// se um site tiver múltiplos grupos relevantes no futuro)
	group, ok := groups[0].(map[string]interface{})
	if !ok {
		return status, nil
	}

	if soc, ok := group["soc"].(float64); ok {
		status.SOC = soc
	}
	if soh, ok := group["soh"].(float64); ok {
		status.SOH = soh
	}
	// backupTime vem como string (ex: "--" ou "3.20"), tratar com cautela
	if bt, ok := group["backupTime"].(string); ok && bt != "--" {
		fmt.Sscanf(bt, "%f", &status.BackupTimeH)
	}

	return status, nil
}