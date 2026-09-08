// internal/adapters/neteco/mapper.go — nova função para energia

package neteco

import (
	"fmt"

	"towercore/internal/core/interfaces"
)

const (
	mocIdDCPower        = 69999
	mocIdDCDistribution = 60009
	mocIdRectifierGroup = 60039
	mocIdBatteryGroup   = 60016
	mocIdGenset         = 60003
)

type SiteCounterInfo struct {
	DCOutputVoltage  *float64
	DCLoadCurrent    *float64
	RectifierCurrent *float64
	BatteryVoltage   *float64
	BatteryCurrent   *float64
	GensetL1Voltage  *float64
	GensetL2Voltage  *float64
	GensetL3Voltage  *float64
}

func mapSiteCounterInfo(raw map[string]interface{}) (*SiteCounterInfo, error) {
	root, ok := raw["siteCounterInfo"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("campo siteCounterInfo ausente")
	}

	info := &SiteCounterInfo{}

	for _, v := range root {
		device, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		status, _ := device["status"].(map[string]interface{})
		mocId := int64(0)
		if status != nil {
			if m, ok := status["mocId"].(float64); ok {
				mocId = int64(m)
			}
		}

		counters, _ := device["counters"].([]interface{})
		realtime, _ := device["realTimeCounters"].([]interface{})
		allCounters := append(counters, realtime...)

		switch mocId {
		case mocIdDCPower, mocIdDCDistribution:
			info.DCOutputVoltage = extractCounterValue(allCounters, "DC Output Voltage")
			info.DCLoadCurrent = extractCounterValue(allCounters, "Total DC Load Current")
		case mocIdRectifierGroup:
			info.RectifierCurrent = extractCounterValue(allCounters, "Total DC Output Current")
		case mocIdBatteryGroup:
			info.BatteryVoltage = extractCounterValue(allCounters, "Voltage")
			info.BatteryCurrent = extractCounterValue(allCounters, "Current")
		case mocIdGenset:
			info.GensetL1Voltage = extractCounterValue(allCounters, "Phase L1 Voltage")
			info.GensetL2Voltage = extractCounterValue(allCounters, "Phase L2 Voltage")
			info.GensetL3Voltage = extractCounterValue(allCounters, "Phase L3 Voltage")
		}
	}

	return info, nil
}

func extractCounterValue(counters []interface{}, name string) *float64 {
	for _, c := range counters {
		counter, ok := c.(map[string]interface{})
		if !ok {
			continue
		}
		if counter["name"] != name {
			continue
		}
		valStr, _ := counter["value"].(string)
		if valStr == "" {
			return nil // sentinela "sem dado" — nunca fabricar
		}
		var f float64
		if _, err := fmt.Sscanf(valStr, "%f", &f); err != nil {
			return nil
		}
		return &f
	}
	return nil
}

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
		return status, nil
	}

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
	if bt, ok := group["backupTime"].(string); ok && bt != "--" {
		fmt.Sscanf(bt, "%f", &status.BackupTimeH)
	}

	return status, nil
}
