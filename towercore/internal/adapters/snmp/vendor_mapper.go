package snmp

import "strings"

// enterpriseVendors mapeia o número de enterprise SNMP (sob
// 1.3.6.1.4.1.<enterprise>) para o nome de vendor usado em todo o
// towercore (eltek/huawei/enetek), o mesmo valor esperado em
// domain.Tower.Vendor e usado para escolher o Profile certo.
//
// Fonte dos números: ficheiros MIB em mibs/<VENDOR>/.
//   - Huawei:  enterprises 2011   (IMAP_NORTHBOUND_MIB-V2.mib)
//   - Eltek:   enterprises 12148  (eltek/profile.go)
//   - Enetek:  enterprises 53318  (enetek_J.mib)
var enterpriseVendors = map[string]string{
	"1.3.6.1.4.1.2011":  "huawei",
	"1.3.6.1.4.1.12148": "eltek",
	"1.3.6.1.4.1.53318": "enetek",
}

// VendorFromSysObjectID tenta identificar o vendor a partir do valor
// de sysObjectID devolvido por um dispositivo. Devolve "" se o prefixo
// não corresponder a nenhum vendor conhecido — nesse caso o
// dispositivo continua a entrar em discovered_devices, só sem vendor
// preenchido, para revisão manual.
func VendorFromSysObjectID(sysObjectID string) string {
	normalized := strings.TrimPrefix(strings.TrimSpace(sysObjectID), ".")
	for prefix, vendor := range enterpriseVendors {
		if strings.HasPrefix(normalized, prefix) {
			return vendor
		}
	}
	return ""
}
