// Ferramenta de validação empírica de OIDs em campo.
//
// Não faz parte do towercore em produção — é um utilitário standalone
// para apontar a um site (ex: KPALAN_S15) e fazer um walk completo ou a
// um branch específico da MIB, imprimindo OID, tipo e valor bruto tal
// como o dispositivo devolve. Serve para confirmar/descartar hipóteses
// de índice desalinhado após troca de hardware (bateria, módulo, etc.)
// antes de mexer em profiles/thresholds no código.
//
// Uso:
//
//	go run . -target 192.168.107.5 -community public -oid 1.3.6.1.4.1.15104
//	go run . -target 192.168.107.5:161 -community public -version v2c
//	go run . -target 192.168.107.5 -v3user admin -authproto sha -authpass ... -privproto aes -privpass ...
//
// Sem -oid, faz walk a partir de 1.3.6.1.2.1 (system) — para um walk
// completo do branch do vendor, passar -oid com o OID base do MIB
// (ex.: enterprise OID Eltek/Enetek/etc, já presente nos profiles do
// towercore).
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	g "github.com/gosnmp/gosnmp"
)

func main() {
	var (
		target     = flag.String("target", "", "IP ou host[:porta] do dispositivo (obrigatório)")
		community  = flag.String("community", "public", "SNMP community (v1/v2c)")
		version    = flag.String("version", "v2c", "Versão SNMP: v1, v2c ou v3")
		baseOID    = flag.String("oid", "1.3.6.1.2.1", "OID base para o walk (default: system MIB-2)")
		timeoutSec = flag.Int("timeout", 5, "Timeout por request, em segundos")
		retries    = flag.Int("retries", 1, "Número de retries")
		outCSV     = flag.String("csv", "", "Caminho opcional para gravar resultado também em CSV")

		// SNMPv3
		v3User     = flag.String("v3user", "", "SNMPv3 username")
		authProto  = flag.String("authproto", "", "SNMPv3 auth protocol: md5, sha, sha256, sha512")
		authPass   = flag.String("authpass", "", "SNMPv3 auth password")
		privProto  = flag.String("privproto", "", "SNMPv3 priv protocol: des, aes, aes256")
		privPass   = flag.String("privpass", "", "SNMPv3 priv password")
	)
	flag.Parse()

	if *target == "" {
		fmt.Println("erro: -target é obrigatório")
		flag.Usage()
		os.Exit(1)
	}

	host, port := splitHostPort(*target)

	params := &g.GoSNMP{
		Target:    host,
		Port:      port,
		Timeout:   time.Duration(*timeoutSec) * time.Second,
		Retries:   *retries,
		MaxOids:   60,
	}

	switch strings.ToLower(*version) {
	case "v1":
		params.Version = g.Version1
		params.Community = *community
	case "v2c":
		params.Version = g.Version2c
		params.Community = *community
	case "v3":
		params.Version = g.Version3
		params.SecurityModel = g.UserSecurityModel
		msgFlags := g.NoAuthNoPriv
		usmParams := &g.UsmSecurityParameters{UserName: *v3User}

		if *authPass != "" {
			usmParams.AuthenticationPassphrase = *authPass
			usmParams.AuthenticationProtocol = parseAuthProto(*authProto)
			msgFlags = g.AuthNoPriv
		}
		if *privPass != "" {
			usmParams.PrivacyPassphrase = *privPass
			usmParams.PrivacyProtocol = parsePrivProto(*privProto)
			msgFlags = g.AuthPriv
		}
		params.MsgFlags = msgFlags
		params.SecurityParameters = usmParams
	default:
		log.Fatalf("versão inválida: %s (use v1, v2c ou v3)", *version)
	}

	if err := params.Connect(); err != nil {
		log.Fatalf("falha a conectar a %s:%d — %v", host, port, err)
	}
	defer params.Conn.Close()

	fmt.Printf("# SNMP walk — target=%s:%d version=%s base_oid=%s\n", host, port, *version, *baseOID)
	fmt.Printf("# %s\n\n", time.Now().Format(time.RFC3339))
	fmt.Printf("%-45s %-12s %s\n", "OID", "TIPO", "VALOR")
	fmt.Println(strings.Repeat("-", 100))

	var csvWriter *csv.Writer
	var csvFile *os.File
	if *outCSV != "" {
		f, err := os.Create(*outCSV)
		if err != nil {
			log.Fatalf("não foi possível criar %s: %v", *outCSV, err)
		}
		csvFile = f
		defer csvFile.Close()
		csvWriter = csv.NewWriter(f)
		defer csvWriter.Flush()
		_ = csvWriter.Write([]string{"oid", "type", "value"})
	}

	count := 0
	walkFn := func(pdu g.SnmpPDU) error {
		count++
		typ := pdu.Type.String()
		val := formatValue(pdu)
		fmt.Printf("%-45s %-12s %s\n", pdu.Name, typ, val)
		if csvWriter != nil {
			_ = csvWriter.Write([]string{pdu.Name, typ, val})
		}
		return nil
	}

	err := params.BulkWalk(*baseOID, walkFn)
	if err != nil {
		// fallback para GetNext clássico (alguns dispositivos antigos
		// não suportam GetBulk correctamente)
		fmt.Printf("\n# BulkWalk falhou (%v), a tentar Walk clássico...\n\n", err)
		err = params.Walk(*baseOID, walkFn)
		if err != nil {
			log.Fatalf("walk falhou: %v", err)
		}
	}

	fmt.Printf("\n# total: %d OIDs\n", count)
	if *outCSV != "" {
		fmt.Printf("# gravado também em: %s\n", *outCSV)
	}
}

func splitHostPort(target string) (string, uint16) {
	if strings.Contains(target, ":") {
		parts := strings.SplitN(target, ":", 2)
		var port uint16 = 161
		fmt.Sscanf(parts[1], "%d", &port)
		return parts[0], port
	}
	return target, 161
}

func parseAuthProto(s string) g.SnmpV3AuthProtocol {
	switch strings.ToLower(s) {
	case "sha":
		return g.SHA
	case "sha256":
		return g.SHA256
	case "sha512":
		return g.SHA512
	case "md5":
		return g.MD5
	default:
		return g.SHA
	}
}

func parsePrivProto(s string) g.SnmpV3PrivProtocol {
	switch strings.ToLower(s) {
	case "aes":
		return g.AES
	case "aes256":
		return g.AES256
	case "des":
		return g.DES
	default:
		return g.AES
	}
}

func formatValue(pdu g.SnmpPDU) string {
	switch pdu.Type {
	case g.OctetString:
		b, ok := pdu.Value.([]byte)
		if ok {
			return fmt.Sprintf("%q", string(b))
		}
		return fmt.Sprintf("%v", pdu.Value)
	default:
		return fmt.Sprintf("%v", pdu.Value)
	}
}