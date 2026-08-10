package snmp

import "strings"

// enterpriseVendors mapeia o número de enterprise SNMP (sob
// 1.3.6.1.4.1.<enterprise>) para o nome de vendor usado em todo o
// towercore (eltek/huawei/enetek/vertiv), o mesmo valor esperado em
// domain.Tower.Vendor e usado para escolher o Profile certo.
//
// Fonte dos números: ficheiros MIB em mibs/<VENDOR>/, exceto Vertiv que
// ainda não tem MIB oficial em mibs/ — número confirmado por SNMP walk
// real (192.168.204.133, 2026-08-10): sysObjectID devolveu
// .1.3.6.1.4.1.6302.2.1 e sysDescr identificou "Vertiv Tech Co.,Ltd.".
//   - Huawei:  enterprises 2011   (IMAP_NORTHBOUND_MIB-V2.mib)
//   - Eltek:   enterprises 12148  (eltek/profile.go)
//   - Enetek:  enterprises 53318  (enetek_J.mib)
//   - Vertiv:  enterprises 6302   (confirmado via walk real, sem MIB
//     ainda em mibs/ — adicionar o ficheiro MIB oficial aqui se/quando
//     obtido, para documentar a par dos outros vendors)
var enterpriseVendors = map[string]string{
	"1.3.6.1.4.1.2011":  "huawei",
	"1.3.6.1.4.1.12148": "eltek",
	"1.3.6.1.4.1.53318": "enetek",
	"1.3.6.1.4.1.6302":  "vertiv",
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