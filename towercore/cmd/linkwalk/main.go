// eltekwalk é uma tool standalone de diagnóstico para validar empiricamente
// os OIDs e scale factors usados no profile Eltek (internal/adapters/eltek).
//
// Segue o mesmo padrão de comapcheck / vertivwalk: consulta o equipamento
// real via SNMP, mostra o raw value devolvido, e aplica vários candidatos
// de Scale lado a lado para comparar com a leitura física do equipamento
// (display do controlador, multímetro, etc.) antes de mudar código.
//
// Uso:
//
//	go run ./eltekwalk -ip 192.168.203.5 -community public
//	go run ./eltekwalk -ip 192.168.203.5 -community public -oid .1.3.6.1.4.1.12148.10.9.2.5.0
//	go run ./eltekwalk -ip 192.168.203.5 -community public -walk .1.3.6.1.4.1.12148.10.9
//
// Sem -oid nem -walk, corre GET a todos os OIDs conhecidos do profile Eltek
// (metrics.go local abaixo) e imprime uma tabela: key, OID, raw, e o valor
// resultante aplicando Scale 1, 0.1, 0.01 e 10 lado a lado.
package main

import (
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/gosnmp/gosnmp"
)

// knownMetric espelha as entradas relevantes de internal/adapters/eltek/profile.go
// Mantido aqui (duplicado, não importado) para a tool poder correr isolada
// sem depender do resto do módulo towercore.
type knownMetric struct {
	Key         string
	OID         string
	DeclaredSca float64 // scale atualmente no código, para referência
	Note        string
}

var knownMetrics = []knownMetric{
	{"battery_voltage_v", ".1.3.6.1.4.1.12148.10.10.5.5.0", 0.01, ""},
	{"battery_current_a", ".1.3.6.1.4.1.12148.10.10.6.5.0", 0.1, ""},
	{"battery_temperature_c", ".1.3.6.1.4.1.12148.10.10.7.5.0", 1, "IgnoreValues: -100 (slot vazio)"},
	{"battery_quality", ".1.3.6.1.4.1.12148.10.10.12.5.0", 1, "ZeroMeansNotTested"},
	{"battery_remaining_pct?", ".1.3.6.1.4.1.12148.10.10.9.5.0", 1, "key sem sufixo no código-fonte, confirmar unidade"},
	{"battery_total_pct?", ".1.3.6.1.4.1.12148.10.10.11.5.0", 1, "key sem sufixo no código-fonte, confirmar unidade"},
	{"battery_status", ".1.3.6.1.4.1.12148.10.10.1.0", 1, ""},

	{"load_current_a", ".1.3.6.1.4.1.12148.10.9.2.5.0", 1, "SUSPEITO: comentário diz MULTIPLIER 0.1 mas Scale=1 no código"},
	{"load_voltage_v", ".1.3.6.1.4.1.12148.10.9.9.1.6.1.1", 0.01, ""},
	{"load_status", ".1.3.6.1.4.1.12148.10.9.1.0", 1, ""},

	{"mains_status", ".1.3.6.1.4.1.12148.10.3.1.0", 1, ""},
	{"mains_voltage_l1_v", ".1.3.6.1.4.1.12148.10.3.4.1.6.1", 1, "corrigido de 10 para 1 em 2026-07-09"},
	{"mains_voltage_l2_v", ".1.3.6.1.4.1.12148.10.3.4.1.6.2", 1, "corrigido de 10 para 1 em 2026-07-09"},
	{"mains_voltage_l3_v", ".1.3.6.1.4.1.12148.10.3.4.1.6.3", 1, "reintegrado em 2026-07-09, só existe em hardware 3 fases"},

	{"rectifier_1_input_v", ".1.3.6.1.4.1.12148.10.5.6.1.4.1", 1, "corrigido de 0.1 para 1"},
	{"rectifier_2_input_v", ".1.3.6.1.4.1.12148.10.5.6.1.4.2", 1, "corrigido de 0.1 para 1"},
	{"rectifier_3_input_v", ".1.3.6.1.4.1.12148.10.5.6.1.4.3", 1, "adicionado 2026-07-09, só existe em hardware 3 retificadores"},
	{"rectifier_1_output_a", ".1.3.6.1.4.1.12148.10.5.6.1.3.1", 0.1, ""},
	{"rectifier_2_output_a", ".1.3.6.1.4.1.12148.10.5.6.1.3.2", 0.1, ""},
	{"rectifier_3_output_a", ".1.3.6.1.4.1.12148.10.5.6.1.3.3", 0.1, ""},
	{"rectifier_1_status", ".1.3.6.1.4.1.12148.10.5.6.1.2.1", 1, ""},
	{"rectifier_2_status", ".1.3.6.1.4.1.12148.10.5.6.1.2.2", 1, ""},
	{"rectifier_3_status", ".1.3.6.1.4.1.12148.10.5.6.1.2.3", 1, ""},
	{"rectifiers_temperature_c", ".1.3.6.1.4.1.12148.10.5.18.5.0", 1, ""},

	{"controller_temperature_c", ".1.3.6.1.4.1.12148.10.13.11.2.1.6.1.1", 1, ""},
	{"system_status", ".1.3.6.1.4.1.12148.10.2.1.0", 1, ""},
}

func main() {
	ip := flag.String("ip", "", "IP do equipamento Eltek (obrigatório)")
	port := flag.Uint("port", 161, "porta SNMP")
	community := flag.String("community", "public", "community string SNMP")
	version := flag.String("version", "2c", "versão SNMP: 1 ou 2c")
	timeout := flag.Duration("timeout", 5*time.Second, "timeout por request")
	retries := flag.Int("retries", 1, "número de retries")
	singleOID := flag.String("oid", "", "se definido, faz GET só a este OID (ignora tabela completa)")
	walkOID := flag.String("walk", "", "se definido, faz WALK a partir deste OID base (descoberta, ignora tabela conhecida)")
	flag.Parse()

	if *ip == "" {
		fmt.Fprintln(os.Stderr, "erro: -ip é obrigatório")
		flag.Usage()
		os.Exit(1)
	}

	snmpVersion := gosnmp.Version2c
	if *version == "1" {
		snmpVersion = gosnmp.Version1
	}

	params := &gosnmp.GoSNMP{
		Target:    *ip,
		Port:      uint16(*port),
		Community: *community,
		Version:   snmpVersion,
		Timeout:   *timeout,
		Retries:   *retries,
	}

	if err := params.Connect(); err != nil {
		log.Fatalf("falha ao abrir ligação SNMP a %s:%d: %v", *ip, *port, err)
	}
	defer params.Conn.Close()

	switch {
	case *walkOID != "":
		runWalk(params, *walkOID)
	case *singleOID != "":
		runSingleGet(params, *singleOID)
	default:
		runKnownTable(params)
	}
}

// toFloat converte o resultado devolvido pelo gosnmp (Integer, Counter32,
// Counter64/big.Int, Gauge32, OctetString, etc.) para float64, tal como o
// SNMPIngestService faria antes de aplicar Scale.
//
// Nota de lição já registada: valores gosnmp.Counter64 vêm como *big.Int e
// precisam de int(...Int64()), NÃO existe método .IntPart() nesse tipo.
func toFloat(pdu gosnmp.SnmpPDU) (float64, string, error) {
	switch pdu.Type {
	case gosnmp.Integer:
		v, ok := pdu.Value.(int)
		if !ok {
			return 0, "", fmt.Errorf("tipo inesperado para Integer: %T", pdu.Value)
		}
		return float64(v), "Integer", nil
	case gosnmp.Counter32, gosnmp.Gauge32, gosnmp.TimeTicks, gosnmp.Uinteger32:
		v := gosnmp.ToBigInt(pdu.Value)
		return float64(v.Int64()), fmt.Sprintf("%v", pdu.Type), nil
	case gosnmp.Counter64:
		bi, ok := pdu.Value.(*big.Int)
		if !ok {
			bi = gosnmp.ToBigInt(pdu.Value)
		}
		return float64(bi.Int64()), "Counter64", nil
	case gosnmp.OctetString:
		b, ok := pdu.Value.([]byte)
		if !ok {
			return 0, "", fmt.Errorf("tipo inesperado para OctetString: %T", pdu.Value)
		}
		return 0, fmt.Sprintf("OctetString(%q)", string(b)), fmt.Errorf("valor não numérico")
	default:
		v := gosnmp.ToBigInt(pdu.Value)
		return float64(v.Int64()), fmt.Sprintf("%v", pdu.Type), nil
	}
}

func runKnownTable(params *gosnmp.GoSNMP) {
	oids := make([]string, 0, len(knownMetrics))
	byOID := make(map[string]knownMetric, len(knownMetrics))
	for _, m := range knownMetrics {
		oids = append(oids, m.OID)
		byOID[m.OID] = m
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tOID\tRAW\tSNMP_TYPE\tSCALE_ATUAL\tx1\tx0.1\tx0.01\tx10\tNOTA")

	// GET em lotes (gosnmp aceita múltiplos OIDs por request; mantemos
	// lotes pequenos para não estourar o limite de PDU de equipamento
	// mais antigo).
	const batchSize = 10
	for i := 0; i < len(oids); i += batchSize {
		end := i + batchSize
		if end > len(oids) {
			end = len(oids)
		}
		batch := oids[i:end]

		result, err := params.Get(batch)
		if err != nil {
			for _, oid := range batch {
				m := byOID[oid]
				fmt.Fprintf(w, "%s\t%s\tERRO\t-\t%v\t-\t-\t-\t-\t%s | erro: %v\n",
					m.Key, oid, m.DeclaredSca, m.Note, err)
			}
			continue
		}

		for _, pdu := range result.Variables {
			m, ok := byOID[pdu.Name]
			if !ok {
				// gosnmp às vezes devolve o OID sem o ponto inicial
				m, ok = byOID["."+pdu.Name]
				if !ok {
					continue
				}
			}

			if pdu.Type == gosnmp.NoSuchObject || pdu.Type == gosnmp.NoSuchInstance || pdu.Type == gosnmp.EndOfMibView {
				fmt.Fprintf(w, "%s\t%s\tN/A\t%v\t%v\t-\t-\t-\t-\t%s | OID não existe neste equipamento\n",
					m.Key, m.OID, pdu.Type, m.DeclaredSca, m.Note)
				continue
			}

			raw, typ, err := toFloat(pdu)
			if err != nil {
				fmt.Fprintf(w, "%s\t%s\t%v\t%s\t%v\t-\t-\t-\t-\t%s | %v\n",
					m.Key, m.OID, pdu.Value, typ, m.DeclaredSca, m.Note, err)
				continue
			}

			fmt.Fprintf(w, "%s\t%s\t%.4f\t%s\t%v\t%.4f\t%.4f\t%.4f\t%.4f\t%s\n",
				m.Key, m.OID, raw, typ, m.DeclaredSca,
				raw*1, raw*0.1, raw*0.01, raw*10,
				m.Note)
		}
	}
	w.Flush()

	fmt.Println("\nComparação: anota a leitura REAL do equipamento (display/multímetro) e vê qual coluna (x1/x0.1/x0.01/x10) bate certo.")
	fmt.Println("Atenção especial a load_current_a — comentário no código diz MULTIPLIER 0.1 mas Scale atual é 1.")
}

func runSingleGet(params *gosnmp.GoSNMP, oid string) {
	result, err := params.Get([]string{oid})
	if err != nil {
		log.Fatalf("GET falhou para %s: %v", oid, err)
	}
	for _, pdu := range result.Variables {
		if pdu.Type == gosnmp.NoSuchObject || pdu.Type == gosnmp.NoSuchInstance {
			fmt.Printf("%s => OID não existe neste equipamento\n", oid)
			continue
		}
		raw, typ, err := toFloat(pdu)
		if err != nil {
			fmt.Printf("%s => valor=%v tipo=%s (não numérico: %v)\n", oid, pdu.Value, typ, err)
			continue
		}
		fmt.Printf("%s => raw=%.4f tipo=%s | x1=%.4f x0.1=%.4f x0.01=%.4f x10=%.4f\n",
			oid, raw, typ, raw*1, raw*0.1, raw*0.01, raw*10)
	}
}

func runWalk(params *gosnmp.GoSNMP, baseOID string) {
	var rows []string
	err := params.Walk(baseOID, func(pdu gosnmp.SnmpPDU) error {
		raw, typ, ferr := toFloat(pdu)
		if ferr != nil {
			rows = append(rows, fmt.Sprintf("%s = %v (tipo=%s, não numérico)", pdu.Name, pdu.Value, typ))
			return nil
		}
		rows = append(rows, fmt.Sprintf("%s = %.4f (tipo=%s)", pdu.Name, raw, typ))
		return nil
	})
	if err != nil {
		log.Fatalf("walk falhou a partir de %s: %v", baseOID, err)
	}
	sort.Strings(rows)
	for _, r := range rows {
		fmt.Println(r)
	}
	fmt.Printf("\n%d OIDs encontrados sob %s\n", len(rows), baseOID)
}