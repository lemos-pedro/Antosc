package main

import (
	"fmt"
	"time"

	"github.com/gosnmp/gosnmp"
)

func main() {
	// Vamos focar-nos apenas no .5 que sabemos que responde
	target := "192.168.115.5"
	community := "public" // Tenta também "private" ou "monitor" se "public" não trouxer nada

	fmt.Printf("\n--- A explorar a árvore completa de: %s ---\n", target)
	
	params := &gosnmp.GoSNMP{
		Target:    target,
		Port:      161,
		Transport: "udp",
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(5) * time.Second, // Aumentámos o tempo de espera
	}

	err := params.Connect()
	if err != nil {
		fmt.Printf("[-] Erro ao conectar: %v\n", err)
		return
	}
	defer params.Conn.Close()

	// Alteração: De "1.3.6.1.2.1" para "1.3.6.1"
	// Isto força o script a ler a raiz da árvore SNMP
	err = params.BulkWalk("1.3.6.1", func(pdu gosnmp.SnmpPDU) error {
		// Imprimimos apenas OIDs que tenham valores interessantes
		// Isso vai ajudar a encontrar os dados "escondidos"
		fmt.Printf("OID: %s | Tipo: %s | Valor: %v\n", pdu.Name, pdu.Type, pdu.Value)
		return nil
	})

	if err != nil {
		fmt.Printf("[-] Erro durante o Walk: %v\n", err)
	}
}