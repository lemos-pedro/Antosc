package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gosnmp/gosnmp"
)

// snmpwalker: ferramenta standalone para varrer um dispositivo SNMP e
// listar TODOS os OIDs que ele expõe, com nome (se resolvido pelo MIB
// implícito da lib) e valor. Objetivo: descobrir se o Eltek/rectificador
// já expõe porta/temperatura/fumo em OIDs que ainda não estão mapeados
// no SNMPIngestService, antes de decidir comprar sensores novos.
//
// Uso:
//   go run main.go <target_ip> <version> [args...]
//
// v2c:
//   go run main.go 192.168.1.10 v2c <community>
//
// v3:
//   go run main.go 192.168.1.10 v3 <user> <authProto> <authPass> <privProto> <privPass>
//   authProto/privProto: MD5, SHA, SHA224, SHA256, SHA384, SHA512 | DES, AES, AES192, AES256

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Uso:")
		fmt.Println("  v2c: go run main.go <ip> v2c <community>")
		fmt.Println("  v3:  go run main.go <ip> v3 <user> <authProto> <authPass> <privProto> <privPass>")
		os.Exit(1)
	}

	target := os.Args[1]
	version := os.Args[2]

	params := &gosnmp.GoSNMP{
		Target:    target,
		Port:      161,
		Timeout:   time.Duration(10) * time.Second,
		Retries:   2,
		MaxOids:   gosnmp.MaxOids,
	}

	switch version {
	case "v2c":
		if len(os.Args) < 4 {
			log.Fatal("v2c requer <community>")
		}
		params.Version = gosnmp.Version2c
		params.Community = os.Args[3]

	case "v3":
		if len(os.Args) < 8 {
			log.Fatal("v3 requer <user> <authProto> <authPass> <privProto> <privPass>")
		}
		user := os.Args[3]
		authProto := parseAuthProtocol(os.Args[4])
		authPass := os.Args[5]
		privProto := parsePrivProtocol(os.Args[6])
		privPass := os.Args[7]

		params.Version = gosnmp.Version3
		params.SecurityModel = gosnmp.UserSecurityModel
		params.MsgFlags = gosnmp.AuthPriv
		params.SecurityParameters = &gosnmp.UsmSecurityParameters{
			UserName:                 user,
			AuthenticationProtocol:   authProto,
			AuthenticationPassphrase: authPass,
			PrivacyProtocol:          privProto,
			PrivacyPassphrase:        privPass,
		}

	default:
		log.Fatalf("versão desconhecida: %s (usa v2c ou v3)", version)
	}

	if err := params.Connect(); err != nil {
		log.Fatalf("erro a conectar: %v", err)
	}
	defer params.Conn.Close()

	// SNMPv3 precisa de um handshake de discovery (EngineID/boots/time)
	// antes de qualquer pedido autenticado. O Connect() por si só nem
	// sempre despoleta isto corretamente contra todos os agentes — um
	// Get inicial a sysDescr (OID universal, sem auth necessária para
	// discovery) força o gosnmp a aprender o EngineID certo antes do
	// Walk autenticado.
	if version == "v3" {
		_, err := params.Get([]string{".1.3.6.1.2.1.1.1.0"})
		if err != nil {
			log.Printf("aviso: discovery inicial falhou (%v) — a tentar walk na mesma", err)
		} else {
			fmt.Println("Discovery SNMPv3 OK (EngineID aprendido).")
		}
	}

	fmt.Printf("=== SNMP walk em %s (%s) ===\n\n", target, version)
	fmt.Println("Root OID: .1.3.6.1.4.1 (private enterprises — pode demorar)")
	fmt.Println("Ctrl+C para parar se demorar demasiado; corre depois num sub-ramo mais específico.")
	fmt.Println()

	count := 0
	err := params.Walk(".1.3.6.1.4.1", func(pdu gosnmp.SnmpPDU) error {
		count++
		fmt.Printf("%-45s = %v  [%s]\n", pdu.Name, formatValue(pdu), pdu.Type)
		return nil
	})
	if err != nil {
		log.Printf("walk interrompido: %v", err)
	}

	fmt.Printf("\n=== Total: %d OIDs encontrados ===\n", count)
	fmt.Println("\nProcura na saída acima por nomes/padrões como:")
	fmt.Println("  - door, porta, access, intrusion")
	fmt.Println("  - temp, temperature, ambient, room")
	fmt.Println("  - smoke, fire, fumo, incendio")
	fmt.Println("  - humidity, humidade")
}

func formatValue(pdu gosnmp.SnmpPDU) interface{} {
	switch pdu.Type {
	case gosnmp.OctetString:
		return string(pdu.Value.([]byte))
	default:
		return gosnmp.ToBigInt(pdu.Value)
	}
}

func parseAuthProtocol(s string) gosnmp.SnmpV3AuthProtocol {
	switch s {
	case "MD5":
		return gosnmp.MD5
	case "SHA":
		return gosnmp.SHA
	case "SHA224":
		return gosnmp.SHA224
	case "SHA256":
		return gosnmp.SHA256
	case "SHA384":
		return gosnmp.SHA384
	case "SHA512":
		return gosnmp.SHA512
	default:
		log.Fatalf("authProto desconhecido: %s", s)
		return gosnmp.NoAuth
	}
}

func parsePrivProtocol(s string) gosnmp.SnmpV3PrivProtocol {
	switch s {
	case "DES":
		return gosnmp.DES
	case "AES":
		return gosnmp.AES
	case "AES192":
		return gosnmp.AES192
	case "AES256":
		return gosnmp.AES256
	default:
		log.Fatalf("privProto desconhecido: %s", s)
		return gosnmp.NoPriv
	}
}