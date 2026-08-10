package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
)

// OIDs do perfil Eltek usados pelo TowerCore (GET pontual).
var eltekProfileOIDs = []struct {
	OID string
	Key string
}{
	{".1.3.6.1.4.1.12148.10.10.5.5.0", "battery_voltage_v"},
	{".1.3.6.1.4.1.12148.10.10.6.5.0", "battery_current_a"},
	{".1.3.6.1.4.1.12148.10.10.7.5.0", "battery_temperature_c"},
	{".1.3.6.1.4.1.12148.10.10.12.5.0", "battery_quality"},
	{".1.3.6.1.4.1.12148.10.10.9.5.0", "battery_remaining_ah"},
	{".1.3.6.1.4.1.12148.10.10.11.5.0", "battery_total_ah"},
	{".1.3.6.1.4.1.12148.10.10.1.0", "battery_status"},
	{".1.3.6.1.4.1.12148.10.9.2.5.0", "load_current_a"},
	{".1.3.6.1.4.1.12148.10.9.9.1.6.1.1", "load_voltage_v"},
	{".1.3.6.1.4.1.12148.10.9.1.0", "load_status"},
	{".1.3.6.1.4.1.12148.10.3.1.0", "mains_status"},
	{".1.3.6.1.4.1.12148.10.3.4.1.6.1", "mains_voltage_l1_v"},
	{".1.3.6.1.4.1.12148.10.3.4.1.6.2", "mains_voltage_l2_v"},
	{".1.3.6.1.4.1.12148.10.3.4.1.6.3", "mains_voltage_l3_v"},
	{".1.3.6.1.4.1.12148.10.5.6.1.4.1", "rectifier_1_input_v"},
	{".1.3.6.1.4.1.12148.10.5.6.1.4.2", "rectifier_2_input_v"},
	{".1.3.6.1.4.1.12148.10.5.6.1.4.3", "rectifier_3_input_v"},
	{".1.3.6.1.4.1.12148.10.5.6.1.3.1", "rectifier_1_output_a"},
	{".1.3.6.1.4.1.12148.10.5.6.1.3.2", "rectifier_2_output_a"},
	{".1.3.6.1.4.1.12148.10.5.6.1.3.3", "rectifier_3_output_a"},
	{".1.3.6.1.4.1.12148.10.5.6.1.2.1", "rectifier_1_status"},
	{".1.3.6.1.4.1.12148.10.5.6.1.2.2", "rectifier_2_status"},
	{".1.3.6.1.4.1.12148.10.5.6.1.2.3", "rectifier_3_status"},
	{".1.3.6.1.4.1.12148.10.5.18.5.0", "rectifiers_temperature_c"},
	{".1.3.6.1.4.1.12148.10.13.11.2.1.6.1.1", "controller_temperature_c"},
	{".1.3.6.1.4.1.12148.10.2.1.0", "system_status"},
}

func main() {
	target := flag.String("target", "", "IP único (ex: 192.168.165.5)")
	list := flag.String("list", "", "Ficheiro texto: uma linha por site → name,ip[,community]")
	community := flag.String("community", "public", "Community SNMP v2c default")
	timeout := flag.Int("timeout", 4, "Timeout segundos")
	retries := flag.Int("retries", 1, "Retries")
	mode := flag.String("mode", "profile", "profile | walk | both")
	walkRoot := flag.String("walk-root", ".1.3.6.1.4.1.12148.10", "Raiz do WALK (Eltek enterprise)")
	csvOut := flag.String("csv", "", "Opcional: escrever resumo CSV neste path")
	flag.Parse()

	sites := loadSites(*target, *list, *community)
	if len(sites) == 0 {
		log.Fatal(`uso:
  go run ./cmd/snmpwalk -target 192.168.165.5 -community public -mode both
  go run ./cmd/snmpwalk -list sites.txt -mode profile -csv report.csv

sites.txt (uma por linha):
  SIMIONE_S,192.168.165.5,public
  200_CASAS,192.168.118.5,public
  BR_TOMAS,192.168.182.5`)
	}

	var csvW *csv.Writer
	if *csvOut != "" {
		f, err := os.Create(*csvOut)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		csvW = csv.NewWriter(f)
		defer csvW.Flush()
		_ = csvW.Write([]string{"name", "ip", "community", "ok", "profile_ok", "profile_fail", "walk_count", "error"})
	}

	for _, s := range sites {
		fmt.Printf("\n========== %s (%s) community=%q ==========\n", s.Name, s.IP, s.Community)
		errMsg := ""
		profileOK, profileFail, walkCount := 0, 0, 0

		client, err := connect(s.IP, s.Community, *timeout, *retries)
		if err != nil {
			fmt.Printf("CONNECT FAIL: %v\n", err)
			errMsg = err.Error()
			if csvW != nil {
				_ = csvW.Write([]string{s.Name, s.IP, s.Community, "false", "0", "0", "0", errMsg})
			}
			continue
		}

		if *mode == "profile" || *mode == "both" {
			profileOK, profileFail = runProfile(client)
		}
		if *mode == "walk" || *mode == "both" {
			walkCount = runWalk(client, *walkRoot)
		}
		_ = client.Conn.Close()

		ok := profileOK > 0 || walkCount > 0
		fmt.Printf("--- resumo: profile_ok=%d profile_fail=%d walk_oids=%d ---\n", profileOK, profileFail, walkCount)
		if csvW != nil {
			_ = csvW.Write([]string{
				s.Name, s.IP, s.Community,
				fmt.Sprintf("%v", ok),
				fmt.Sprintf("%d", profileOK),
				fmt.Sprintf("%d", profileFail),
				fmt.Sprintf("%d", walkCount),
				errMsg,
			})
		}
	}
}

type site struct {
	Name, IP, Community string
}

func loadSites(target, listPath, defaultCommunity string) []site {
	var out []site
	if target != "" {
		out = append(out, site{Name: target, IP: target, Community: defaultCommunity})
	}
	if listPath == "" {
		return out
	}
	f, err := os.Open(listPath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ",")
		s := site{Community: defaultCommunity}
		switch len(parts) {
		case 1:
			s.Name, s.IP = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[0])
		case 2:
			s.Name, s.IP = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		default:
			s.Name = strings.TrimSpace(parts[0])
			s.IP = strings.TrimSpace(parts[1])
			if c := strings.TrimSpace(parts[2]); c != "" {
				s.Community = c
			}
		}
		out = append(out, s)
	}
	return out
}

func connect(ip, community string, timeoutSec, retries int) (*gosnmp.GoSNMP, error) {
	c := &gosnmp.GoSNMP{
		Target:    ip,
		Port:      161,
		Community: community,
		Version:   gosnmp.Version2c,
		Timeout:   time.Duration(timeoutSec) * time.Second,
		Retries:   retries,
		MaxOids:   gosnmp.MaxOids,
	}
	if err := c.Connect(); err != nil {
		return nil, err
	}
	return c, nil
}

func runProfile(c *gosnmp.GoSNMP) (ok, fail int) {
	fmt.Println("-- PROFILE GET (OIDs TowerCore/Eltek) --")
	for _, m := range eltekProfileOIDs {
		pkt, err := c.Get([]string{m.OID})
		if err != nil {
			fmt.Printf("  FAIL  %-28s %s  err=%v\n", m.Key, m.OID, err)
			fail++
			continue
		}
		if len(pkt.Variables) == 0 {
			fmt.Printf("  EMPTY %-28s %s\n", m.Key, m.OID)
			fail++
			continue
		}
		v := pkt.Variables[0]
		val, readable := formatPDU(v)
		if !readable {
			fmt.Printf("  MISS  %-28s %s  type=%s\n", m.Key, m.OID, v.Type)
			fail++
			continue
		}
		fmt.Printf("  OK    %-28s %s  = %s\n", m.Key, m.OID, val)
		ok++
	}
	return ok, fail
}

func runWalk(c *gosnmp.GoSNMP, root string) int {
	fmt.Printf("-- WALK %s --\n", root)
	count := 0
	err := c.Walk(root, func(pdu gosnmp.SnmpPDU) error {
		val, ok := formatPDU(pdu)
		if !ok {
			fmt.Printf("  %s type=%s (no value)\n", pdu.Name, pdu.Type)
			return nil
		}
		fmt.Printf("  %s = %s\n", pdu.Name, val)
		count++
		return nil
	})
	if err != nil {
		fmt.Printf("  WALK err: %v\n", err)
	}
	return count
}

func formatPDU(pdu gosnmp.SnmpPDU) (string, bool) {
	switch pdu.Type {
	case gosnmp.NoSuchObject, gosnmp.NoSuchInstance, gosnmp.Null, gosnmp.EndOfContents, gosnmp.EndOfMibView:
		return "", false
	case gosnmp.OctetString:
		b, _ := pdu.Value.([]byte)
		// tenta string; se binário, hex curto
		s := string(b)
		if isPrintable(s) {
			return s, true
		}
		return fmt.Sprintf("0x%x", b), true
	case gosnmp.Integer, gosnmp.Counter32, gosnmp.Gauge32, gosnmp.TimeTicks, gosnmp.Counter64, gosnmp.Uinteger32:
		return fmt.Sprintf("%v", gosnmp.ToBigInt(pdu.Value)), true
	case gosnmp.IPAddress:
		return fmt.Sprintf("%v", pdu.Value), true
	case gosnmp.ObjectIdentifier:
		return fmt.Sprintf("%v", pdu.Value), true
	default:
		return fmt.Sprintf("%v", pdu.Value), true
	}
}

func isPrintable(s string) bool {
	for _, r := range s {
		if r < 32 || r > 126 {
			return false
		}
	}
	return true
}