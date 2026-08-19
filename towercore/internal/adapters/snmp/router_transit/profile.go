package router_transit

// OIDs base do IF-MIB (RFC 2863) e da extensão de alta capacidade (RFC 2863
// / IF-MIB HC counters), usados para qualquer router genérico (Cisco, etc.).
// Estes OIDs são padrão — não são vendor-specific como Eltek/Vertiv/Enetek.
//
// Todos requerem sufixo ".<ifIndex>" no momento do Get, ex:
// ifOperStatusOID + ".10101" para a interface de ifIndex 10101.
const (
	// ifDescr — descrição da interface, ex: "TenGigabitEthernet2/3"
	OIDIfDescr = ".1.3.6.1.2.1.2.2.1.2"

	// ifOperStatus — 1=up, 2=down, 3=testing, 4=unknown, 5=dormant,
	// 6=notPresent, 7=lowerLayerDown
	OIDIfOperStatus = ".1.3.6.1.2.1.2.2.1.8"

	// ifAdminStatus — mesmos valores possíveis que ifOperStatus (1..3)
	OIDIfAdminStatus = ".1.3.6.1.2.1.2.2.1.7"

	// ifInErrors / ifOutErrors — contadores de pacotes com erro
	OIDIfInErrors  = ".1.3.6.1.2.1.2.2.1.14"
	OIDIfOutErrors = ".1.3.6.1.2.1.2.2.1.20"

	// ifInDiscards / ifOutDiscards — pacotes descartados (congestionamento,
	// falta de buffer — sintoma indireto de degradação de qualidade)
	OIDIfInDiscards  = ".1.3.6.1.2.1.2.2.1.13"
	OIDIfOutDiscards = ".1.3.6.1.2.1.2.2.1.19"

	// ifHCInOctets / ifHCOutOctets — contadores de 64-bit (High Capacity).
	// OBRIGATÓRIO usar estes em vez de ifInOctets/ifOutOctets (32-bit,
	// .1.3.6.1.2.1.2.2.1.10/.16) em interfaces >=1Gbps: um link de 10Gbps
	// satura um counter de 32-bit em poucos segundos sob tráfego alto,
	// invalidando qualquer cálculo de bps entre polls.
	OIDIfHCInOctets  = ".1.3.6.1.2.1.31.1.1.1.6"
	OIDIfHCOutOctets = ".1.3.6.1.2.1.31.1.1.1.10"

	// ifHighSpeed — velocidade nominal em Mbps (suporta >4.3Gbps,
	// ao contrário de ifSpeed que estoura em interfaces >=4.3Gbps)
	OIDIfHighSpeed = ".1.3.6.1.2.1.31.1.1.1.15"

	// ifAlias — nome descritivo configurado manualmente no router,
	// ex: "CONNECT-TO-AFRICELL-LAD00395". É esta string que identifica
	// o link para efeitos de reconciliação com domain.NetworkLink.IfAlias.
	OIDIfAlias = ".1.3.6.1.2.1.31.1.1.1.18"

	// ifType — tipo de interface (ethernetCsmacd=6, etc.), usado sobretudo
	// no discovery para filtrar interfaces relevantes.
	OIDIfType = ".1.3.6.1.2.1.2.2.1.3"
)

// operStatusMap traduz o valor inteiro devolvido por SNMP para
// domain.LinkOperStatus. Mantido aqui (não em domain) porque é um detalhe
// de codificação específico do protocolo IF-MIB.
var operStatusIntToString = map[int]string{
	1: "up",
	2: "down",
	3: "testing",
	4: "unknown",
	5: "dormant",
	6: "not_present",
	7: "lower_layer_down",
}
