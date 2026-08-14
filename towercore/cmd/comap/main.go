// comapcheck v3 - mais robusto contra timeouts e conexão suja
package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	fnReadHolding = 0x03
	inactiveValue = 0x8000
)

type regInfo struct {
	addr   uint16
	name   string
	scale  float64
	unit   string
	signed bool
	min    float64 // para validação plausível
	max    float64
}

// Registos que já vimos funcionar nos AC03
var profileAC03 = []regInfo{
	{50, "Battery Voltage", 0.1, "V", false, 8, 36},
	{53, "Oil Pressure", 0.1, "bar", true, 0, 15},
	{54, "Engine Temp", 1.0, "°C", true, -20, 150},
	{55, "Fuel Level", 1.0, "%", false, 0, 120},
}

func main() {
	ip := flag.String("ip", "", "IP do equipamento")
	port := flag.Int("port", 502, "porta")
	unitList := flag.String("unit", "1", "unit ids (recomendado: 1)")
	timeout := flag.Duration("timeout", 3*time.Second, "timeout por pedido")
	flag.Parse()

	if *ip == "" {
		fmt.Fprintln(os.Stderr, "uso: go run main.go -ip <IP> [-unit 1] [-timeout 3s]")
		os.Exit(2)
	}

	units, _ := parseIntList(*unitList)
	addr := net.JoinHostPort(*ip, strconv.Itoa(*port))

	fmt.Printf("=== comapcheck v3 ===\n")
	fmt.Printf("alvo:    %s\n", addr)
	fmt.Printf("units:   %v\n\n", units)

	// Teste TCP inicial
	fmt.Print("[1] TCP ... ")
	conn, err := net.DialTimeout("tcp", addr, *timeout)
	if err != nil {
		fmt.Printf("FALHOU: %v\n", err)
		diagnoseDial(err)
		os.Exit(1)
	}
	fmt.Println("OK")
	conn.Close() // vamos abrir sob demanda

	var bestUnit int = -1
	var bestHits int
	var report []string

	for _, unit := range units {
		fmt.Printf("\n--- Unit ID = %d ---\n", unit)
		hits := 0
		var lines []string

		for _, r := range profileAC03 {
			raw, err := readOne(addr, *timeout, byte(unit), r.addr)
			if err != nil {
				lines = append(lines, fmt.Sprintf("  %-18s ERRO: %v", r.name, err))
				continue
			}

			if raw == inactiveValue {
				lines = append(lines, fmt.Sprintf("  %-18s = 0x8000 (inativo)", r.name))
				hits++ // ainda conta como "respondeu"
				continue
			}

			fval := float64(raw)
			if r.signed {
				fval = float64(int16(raw))
			}
			fval *= r.scale

			plausible := fval >= r.min && fval <= r.max
			mark := ""
			if plausible {
				hits += 2
				mark = " ✓"
			} else {
				hits++
				mark = " (valor estranho)"
			}

			lines = append(lines, fmt.Sprintf("  %-18s = %.1f %s (raw %d)%s", r.name, fval, r.unit, raw, mark))
		}

		for _, l := range lines {
			fmt.Println(l)
		}

		if hits > bestHits {
			bestHits = hits
			bestUnit = unit
			report = lines
		}
	}

	fmt.Println("\n========== RESUMO ==========")
	if bestUnit < 0 || bestHits == 0 {
		fmt.Println("Nenhum registo útil respondeu.")
		fmt.Println("Possíveis causas: Modbus desativado, unit id errado, ou mapa diferente.")
		os.Exit(1)
	}

	fmt.Printf("Melhor Unit ID : %d\n", bestUnit)
	fmt.Printf("Hits           : %d\n", bestHits)

	// Heurística simples
	ac03Good := 0
	for _, l := range report {
		if strings.Contains(l, "✓") || strings.Contains(l, "0x8000") {
			ac03Good++
		}
	}

	if ac03Good >= 2 {
		fmt.Println("Modelo estimado: InteliLite NT / AC03 (alta confiança)")
	} else {
		fmt.Println("Modelo estimado: Desconhecido / parcial")
	}
	fmt.Println("============================")
}

// Abre conexão nova a cada pedido → evita conexão "suja" depois de timeout
func readOne(addr string, timeout time.Duration, unit byte, reg uint16) (uint16, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return 0, fmt.Errorf("dial: %w", err)
	}
	defer conn.Close()

	txID := uint16(time.Now().UnixNano() & 0xFFFF)
	req := make([]byte, 12)
	binary.BigEndian.PutUint16(req[0:2], txID)
	binary.BigEndian.PutUint16(req[2:4], 0)
	binary.BigEndian.PutUint16(req[4:6], 6)
	req[6] = unit
	req[7] = fnReadHolding
	binary.BigEndian.PutUint16(req[8:10], reg)
	binary.BigEndian.PutUint16(req[10:12], 1)

	conn.SetDeadline(time.Now().Add(timeout))
	if _, err := conn.Write(req); err != nil {
		return 0, err
	}

	header := make([]byte, 7)
	if _, err := readFull(conn, header); err != nil {
		return 0, fmt.Errorf("header: %w", err)
	}

	if binary.BigEndian.Uint16(header[0:2]) != txID {
		return 0, errors.New("tx id mismatch")
	}
	length := binary.BigEndian.Uint16(header[4:6])
	if length < 2 {
		return 0, fmt.Errorf("length %d", length)
	}

	pdu := make([]byte, length-1)
	if _, err := readFull(conn, pdu); err != nil {
		return 0, err
	}

	if pdu[0]&0x80 != 0 {
		exc := byte(0)
		if len(pdu) > 1 {
			exc = pdu[1]
		}
		return 0, fmt.Errorf("exceção 0x%02X", exc)
	}

	if len(pdu) < 4 {
		return 0, errors.New("pdu curto")
	}
	return binary.BigEndian.Uint16(pdu[2:4]), nil
}

func readFull(conn net.Conn, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := conn.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func diagnoseDial(err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "i/o timeout"):
		fmt.Println("→ Timeout: IP errado / equipamento off / sem rota")
	case strings.Contains(msg, "connection refused"):
		fmt.Println("→ Connection refused: porta 502 fechada (Modbus desativado?)")
	case strings.Contains(msg, "no route to host"):
		fmt.Println("→ Sem rota de rede")
	}
}

func parseIntList(s string) ([]int, error) {
	var out []int
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := strconv.Atoi(p)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}