// cmd/linkwalk valida empiricamente o IF-MIB contra um router real antes de
// qualquer mapeamento definitivo em domain.NetworkLink. Uso:
//
//	go run ./cmd/linkwalk -host 10.0.0.1 -community public
//
// Lista todas as interfaces descobertas (ifIndex, ifDescr, ifAlias, ifType)
// para confirmares quais correspondem aos links de trânsito nomeados no
// Zabbix (ex: "CONNECTED-BEN-RT", "CONNECT-TO-AFRICELL-LAD00395") e qual o
// ifIndex correto de cada um antes de os registares em network_links.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"towercore/internal/adapters/snmp/router_transit"
	"towercore/internal/core/interfaces"
	"towercore/internal/infrastructure/logger"
)

func main() {
	host := flag.String("host", "", "IP de gestão SNMP do router (obrigatório)")
	community := flag.String("community", "public", "SNMP community")
	port := flag.Uint("port", 161, "porta SNMP")
	timeout := flag.Int("timeout", 5, "timeout em segundos")
	ifIndex := flag.Int("ifindex", 0, "se definido (>0), faz Get pontual de métricas deste ifIndex em vez de discovery completo")
	flag.Parse()

	if *host == "" {
		fmt.Println("uso: linkwalk -host <ip> [-community public] [-ifindex N]")
		os.Exit(1)
	}

	log := logger.New()
	client := router_transit.NewClient(*community, uint16(*port), *timeout, 1, log)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if *ifIndex > 0 {
		fmt.Printf("A recolher métricas pontuais de %s ifIndex=%d...\n\n", *host, *ifIndex)
		snapshot, err := client.GetInterfaceMetrics(ctx, *host, *ifIndex)
		if err != nil {
			fmt.Printf("ERRO: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("OperStatus:   %s\n", snapshot.OperStatus)
		fmt.Printf("AdminStatus:  %s\n", snapshot.AdminStatus)
		fmt.Printf("InOctets:     %d\n", snapshot.InOctets)
		fmt.Printf("OutOctets:    %d\n", snapshot.OutOctets)
		fmt.Printf("InErrors:     %d\n", snapshot.InErrors)
		fmt.Printf("OutErrors:    %d\n", snapshot.OutErrors)
		fmt.Printf("InDiscards:   %d\n", snapshot.InDiscards)
		fmt.Printf("OutDiscards:  %d\n", snapshot.OutDiscards)
		fmt.Printf("SpeedMb:      %d\n", snapshot.SpeedMb)
		return
	}

	fmt.Printf("A descobrir interfaces em %s...\n\n", *host)
	descriptors, err := client.DiscoverInterfaces(ctx, *host)
	if err != nil {
		fmt.Printf("ERRO: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%-8s %-40s %-45s %-6s\n", "IfIndex", "IfDescr", "IfAlias", "IfType")
	fmt.Println("--------------------------------------------------------------------------------------------------")
	for _, d := range descriptors {
		if d.IfAlias == "" {
			// interfaces sem alias configurado normalmente não são links de
			// trânsito relevantes (loopbacks, interfaces internas, etc.) —
			// mas mostramos na mesma para decisão manual.
			continue
		}
		fmt.Printf("%-8d %-40s %-45s %-6d\n", d.IfIndex, d.IfDescr, d.IfAlias, d.IfType)
	}

	fmt.Printf("\nTotal de interfaces com alias configurado: %d (de %d descobertas)\n", countWithAlias(descriptors), len(descriptors))
	fmt.Println("\nPróximo passo: confirmar cada IfIndex+IfAlias contra o nome do link no Zabbix e registar em network_links via POST /api/v1/links (ou seed direto na BD).")
}

func countWithAlias(descriptors []interfaces.InterfaceDescriptor) int {
	count := 0
	for _, d := range descriptors {
		if d.IfAlias != "" {
			count++
		}
	}
	return count
}
