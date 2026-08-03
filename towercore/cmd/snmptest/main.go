// snmp_test_pfsense.go
//
// Teste standalone e descartável para confirmar se um pfSense responde a
// SNMP antes de investirmos tempo a construir o adapter completo.
//
// Uso:
//   1. Copia este ficheiro para dentro da pasta do towercore (para reutilizar
//      o go.mod/go.sum já existente, que já tem gosnmp como dependência):
//        C:\Users\joaquim.pedro\Documents\Antosc\towercore\cmd\snmptest\main.go
//   2. Corre:
//        go run ./cmd/snmptest -target 192.168.139.1 -community public
//
// Testa primeiro com a community "public" (default mais comum). Se não
// responder, tenta outras communities conhecidas do teu ambiente, ou
// confirma que SNMP nem está ativo nesse pfSense.
package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/gosnmp/gosnmp"
)

func main() {
	target := flag.String("target", "", "IP do pfSense a testar (ex: 192.168.139.1)")
	community := flag.String("community", "public", "SNMP community string")
	timeout := flag.Int("timeout", 3, "timeout em segundos")
	flag.Parse()

	if *target == "" {
		log.Fatal("uso: go run . -target <ip> [-community <string>]")
	}

	params := &gosnmp.GoSNMP{
		Target:    *target,
		Port:      161,
		Community: *community,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(*timeout) * time.Second,
		Retries:   1,
	}

	if err := params.Connect(); err != nil {
		log.Fatalf("FALHA ao conectar a %s: %v", *target, err)
	}
	defer params.Conn.Close()

	// sysDescr.0 - OID padrão que qualquer dispositivo SNMP responde se
	// estiver mesmo ativo. Se isto não vier, não há SNMP nesse host.
	oids := []string{"1.3.6.1.2.1.1.1.0", "1.3.6.1.2.1.1.5.0"}

	result, err := params.Get(oids)
	if err != nil {
		log.Fatalf("FALHA no SNMP GET a %s (community=%s): %v", *target, *community, err)
	}

	fmt.Printf("SUCESSO — %s respondeu com community=%q:\n", *target, *community)
	for _, variable := range result.Variables {
		switch variable.Type {
		case gosnmp.OctetString:
			fmt.Printf("  %s = %s\n", variable.Name, string(variable.Value.([]byte)))
		default:
			fmt.Printf("  %s = %v\n", variable.Name, gosnmp.ToBigInt(variable.Value))
		}
	}
}