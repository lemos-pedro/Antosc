package main

import (
	"context"
	"fmt"
	"time"

	"towercore/internal/adapters/enetek"
	"towercore/internal/adapters/snmp"
	"towercore/internal/core/domain"
)

func main() {
	towerID := "24b86ee5-3052-4139-8764-b30fb41a6317"
	target := "192.168.125.1"

	// Monta um domain.Tower mínimo com os campos que o coletor confere.
	// SNMPEnabled tem de ser true, senão o Collect recusa logo com
	// "snmp is disabled for tower". SNMPCommunity é obrigatório em v2c.
	tower := domain.Tower{
		ID:            towerID,
		SNMPEnabled:   true,
		SNMPTarget:    target,
		SNMPVersion:   "v2c",
		SNMPCommunity: "public", // <-- troca pela community real da torre CAVUBA
		Vendor:        "enetek",
	}

	collector := snmp.NewGoSNMPCollector(5*time.Second, 2)
	profile := enetek.Profile()

	raw, err := collector.Collect(context.Background(), tower, profile)
	if err != nil {
		fmt.Println("erro na coleta:", err)
		return
	}

	fmt.Println("=== RAW (OID -> valor) ===")
	for oid, val := range raw {
		fmt.Printf("%s = %v\n", oid, val)
	}

	// NormalizeSamples(profile, raw) — ordem corrigida conforme o compilador indicou
	normalized := snmp.NormalizeSamples(profile, raw)

	fmt.Println("\n=== NORMALIZADO (key -> valor) ===")
	for k, v := range normalized {
		fmt.Printf("%s = %v\n", k, v)
	}
}