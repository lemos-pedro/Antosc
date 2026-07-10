package snmp

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
	"towercore/internal/core/domain"
)

type GoSNMPCollector struct {
	timeout time.Duration
	retries int
	port    uint16
}

func NewGoSNMPCollector(timeout time.Duration, retries int) *GoSNMPCollector {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	if retries < 0 {
		retries = 0
	}
	return &GoSNMPCollector{
		timeout: timeout,
		retries: retries,
		port:    161,
	}
}

func (c *GoSNMPCollector) Collect(ctx context.Context, tower domain.Tower, profile Profile) (map[string]float64, error) {
	if !tower.SNMPEnabled {
		return nil, errors.New("snmp is disabled for tower")
	}
	if strings.TrimSpace(tower.SNMPTarget) == "" {
		return nil, errors.New("snmp_target is required")
	}
	if len(profile.Metrics) == 0 {
		return nil, errors.New("empty profile metrics")
	}

	client := &gosnmp.GoSNMP{
		Target:  tower.SNMPTarget,
		Port:    c.port,
		Timeout: c.timeout,
		Retries: c.retries,
		MaxOids: gosnmp.MaxOids,
	}
	version := strings.ToLower(strings.TrimSpace(tower.SNMPVersion))
	switch version {
	case "", "v2c":
		if strings.TrimSpace(tower.SNMPCommunity) == "" {
			return nil, errors.New("snmp_community is required for v2c")
		}
		client.Community = tower.SNMPCommunity
		client.Version = gosnmp.Version2c
	case "v3":
		if strings.TrimSpace(tower.SNMPV3User) == "" {
			return nil, errors.New("snmp_v3_user is required for v3")
		}
		authProto, err := mapAuthProtocol(tower.SNMPAuthProto)
		if err != nil {
			return nil, err
		}
		security := &gosnmp.UsmSecurityParameters{
			UserName:                 tower.SNMPV3User,
			AuthenticationProtocol:   authProto,
			AuthenticationPassphrase: tower.SNMPAuthPass,
		}
		if strings.TrimSpace(tower.SNMPPrivProto) != "" || strings.TrimSpace(tower.SNMPPrivPass) != "" {
			privProto, err := mapPrivProtocol(tower.SNMPPrivProto)
			if err != nil {
				return nil, err
			}
			security.PrivacyProtocol = privProto
			security.PrivacyPassphrase = tower.SNMPPrivPass
			client.MsgFlags = gosnmp.AuthPriv
		} else {
			client.MsgFlags = gosnmp.AuthNoPriv
		}

		client.Version = gosnmp.Version3
		client.SecurityModel = gosnmp.UserSecurityModel
		client.SecurityParameters = security
	default:
		return nil, errors.New("unsupported snmp_version")
	}

	if err := client.Connect(); err != nil {
		return nil, err
	}
	defer client.Conn.Close()

	oids := make([]string, 0, len(profile.Metrics))
	for _, md := range profile.Metrics {
		oids = append(oids, md.OID)
	}

	type result struct {
		values map[string]float64
		err    error
	}
	ch := make(chan result, 1)

	go func() {
		packet, err := client.Get(oids)
		if err != nil {
			ch <- result{err: err}
			return
		}
		values := make(map[string]float64)
		for _, v := range packet.Variables {
			// DEBUG: loga o tipo/valor devolvido para cada OID pedido, incluindo
			// respostas de erro SNMP (NoSuchInstance/NoSuchObject/Null), que antes
			// eram descartadas em silêncio por toFloat() sem deixar rasto.
			log.Printf("DEBUG snmp target=%s oid=%s type=%v value=%v", tower.SNMPTarget, v.Name, v.Type, v.Value)

			n, err := toFloat(v)
			if err != nil {
				log.Printf("DEBUG snmp target=%s oid=%s skipped: %v", tower.SNMPTarget, v.Name, err)
				continue
			}
			values[v.Name] = n
		}
		ch <- result{values: values}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-ch:
		return r.values, r.err
	}
}

func mapAuthProtocol(v string) (gosnmp.SnmpV3AuthProtocol, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "md5":
		return gosnmp.MD5, nil
	case "sha":
		return gosnmp.SHA, nil
	case "sha224":
		return gosnmp.SHA224, nil
	case "sha256":
		return gosnmp.SHA256, nil
	case "sha384":
		return gosnmp.SHA384, nil
	case "sha512":
		return gosnmp.SHA512, nil
	default:
		return gosnmp.NoAuth, fmt.Errorf("unsupported snmp_auth_protocol: %s", v)
	}
}

func mapPrivProtocol(v string) (gosnmp.SnmpV3PrivProtocol, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "des":
		return gosnmp.DES, nil
	case "aes":
		return gosnmp.AES, nil
	case "aes192":
		return gosnmp.AES192, nil
	case "aes256":
		return gosnmp.AES256, nil
	default:
		return gosnmp.NoPriv, fmt.Errorf("unsupported snmp_priv_protocol: %s", v)
	}
}

func toFloat(pdu gosnmp.SnmpPDU) (float64, error) {
	switch v := pdu.Value.(type) {
	case int:
		return float64(v), nil
	case int8:
		return float64(v), nil
	case int16:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case uint:
		return float64(v), nil
	case uint8:
		return float64(v), nil
	case uint16:
		return float64(v), nil
	case uint32:
		return float64(v), nil
	case uint64:
		return float64(v), nil
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case []byte:
		if len(v) == 0 {
			return 0, errors.New("empty byte value")
		}
		if n, err := strconv.ParseFloat(string(v), 64); err == nil {
			return n, nil
		}
		switch len(v) {
		case 1:
			return float64(v[0]), nil
		case 2:
			return float64(binary.BigEndian.Uint16(v)), nil
		case 4:
			return float64(binary.BigEndian.Uint32(v)), nil
		case 8:
			return math.Float64frombits(binary.BigEndian.Uint64(v)), nil
		default:
			return 0, fmt.Errorf("unsupported byte length: %d", len(v))
		}
	default:
		return 0, fmt.Errorf("unsupported pdu type: %T", pdu.Value)
	}
}