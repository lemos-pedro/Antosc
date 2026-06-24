package services

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
	"towercore/internal/infrastructure/logger"
)

// VendorResolver traduz um sysObjectID no nome de vendor conhecido
// pelo sistema. Existe como interface (em vez de chamar diretamente
// adapters/snmp.VendorFromSysObjectID) para o core/services não
// depender de adapters/snmp — mantém a regra "core nunca depende de
// adapters" do projeto.
type VendorResolver interface {
	VendorFromSysObjectID(sysObjectID string) string
}

// DiscoveryService varre um intervalo de IPs (CIDR) à procura de
// agentes SNMP a responder, usando uma única community candidata
// (decisão tomada para este sistema: só 'Antosc-noc' é tentada).
//
// Dispositivos que respondem são guardados em discovered_devices como
// 'pending' — nunca como Tower diretamente, porque faltam dados que o
// SNMP não fornece (nome amigável, operador, região).
type DiscoveryService struct {
	prober      interfaces.DeviceProber
	repo        interfaces.DiscoveredDeviceRepository
	vendor      VendorResolver
	community   string
	concurrency int
	log         *logger.Logger
	now         func() time.Time
}

// NewDiscoveryService cria o serviço. concurrency limita quantas
// sondagens correm em paralelo (um scan de /24 com concurrency=1 seria
// lento demais; com concurrency alto demais pode saturar a rede ou o
// próprio processo Go com sockets UDP simultâneos).
func NewDiscoveryService(
	prober interfaces.DeviceProber,
	repo interfaces.DiscoveredDeviceRepository,
	vendor VendorResolver,
	community string,
	concurrency int,
	log *logger.Logger,
) *DiscoveryService {
	if concurrency <= 0 {
		concurrency = 32
	}
	return &DiscoveryService{
		prober:      prober,
		repo:        repo,
		vendor:      vendor,
		community:   community,
		concurrency: concurrency,
		log:         log,
		now:         time.Now,
	}
}

// ScanResult resume o resultado de uma passagem de discovery.
type ScanResult struct {
	Scanned   int
	Responded int
	Errors    int
}

// ScanCIDR varre todos os IPs utilizáveis dentro do CIDR indicado
// (ex: "10.0.0.0/24") e regista em discovered_devices os que
// responderem à sondagem SNMP.
func (s *DiscoveryService) ScanCIDR(ctx context.Context, cidr string) (ScanResult, error) {
	ips, err := hostsInCIDR(cidr)
	if err != nil {
		return ScanResult{}, fmt.Errorf("expandir CIDR %s: %w", cidr, err)
	}

	var (
		mu     sync.Mutex
		result ScanResult
		wg     sync.WaitGroup
	)

	sem := make(chan struct{}, s.concurrency)

	for _, ip := range ips {
		select {
		case <-ctx.Done():
			wg.Wait()
			return result, ctx.Err()
		default:
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(ip string) {
			defer wg.Done()
			defer func() { <-sem }()

			probeResult, err := s.prober.Probe(ctx, ip, s.community)

			mu.Lock()
			defer mu.Unlock()
			result.Scanned++

			if err != nil {
				result.Errors++
				return
			}
			if !probeResult.Responded {
				return
			}

			result.Responded++
			s.recordDevice(ctx, ip, probeResult.SysObjectID)
		}(ip)
	}

	wg.Wait()
	return result, nil
}

// recordDevice grava (ou atualiza) a entrada de discovered_devices
// para um IP que respondeu. Erros de persistência são registados via
// logger mas não interrompem o scan dos outros IPs.
func (s *DiscoveryService) recordDevice(ctx context.Context, ip string, sysObjectID string) {
	now := s.now().UTC()
	device := &domain.DiscoveredDevice{
		IPAddress:      ip,
		SysObjectID:    sysObjectID,
		DetectedVendor: s.vendor.VendorFromSysObjectID(sysObjectID),
		SNMPVersion:    "v2c",
		SNMPCommunity:  s.community,
		Status:         domain.DiscoveredDeviceStatusPending,
		FirstSeenAt:    now,
		LastSeenAt:     now,
	}
	if err := s.repo.UpsertSeen(ctx, device); err != nil && s.log != nil {
		s.log.Errorf("discovery upsert failed ip=%s err=%v", ip, err)
	}
}

// hostsInCIDR expande um CIDR para a lista de IPs utilizáveis,
// excluindo o endereço de rede e o de broadcast em blocos /30 ou
// maiores (para blocos /31 e /32, devolve os endereços tal como estão,
// já que não têm broadcast distinto).
func hostsInCIDR(cidr string) ([]string, error) {
	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var ips []string
	for current := cloneIP(ip.Mask(ipNet.Mask)); ipNet.Contains(current); incIP(current) {
		ips = append(ips, current.String())
	}

	ones, bits := ipNet.Mask.Size()
	if bits-ones >= 2 && len(ips) > 2 {
		// Remove endereço de rede (primeiro) e broadcast (último).
		ips = ips[1 : len(ips)-1]
	}

	return ips, nil
}

func cloneIP(ip net.IP) net.IP {
	dup := make(net.IP, len(ip))
	copy(dup, ip)
	return dup
}

func incIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}
