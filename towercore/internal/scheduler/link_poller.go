package scheduler

import (
	"context"
	"time"

	"towercore/internal/core/interfaces"
	"towercore/internal/core/services"
	"towercore/internal/infrastructure/logger"
)

// LinkPoller corre em intervalo próprio (tipicamente mais curto que o
// scheduler de energia, ex: 30s) e avalia todos os links registados.
// Reutiliza LinkStatusService — a lógica de negócio não muda consoante
// quem a invoca, seguindo a regra "scheduler reutiliza casos de uso do
// core/services" já definida em estrutura.md.
type LinkPoller struct {
	linkRepo      interfaces.NetworkLinkRepository
	statusService *services.LinkStatusService
	intervalSecs  int
	log           *logger.Logger
}

func NewLinkPoller(
	linkRepo interfaces.NetworkLinkRepository,
	statusService *services.LinkStatusService,
	intervalSecs int,
	log *logger.Logger,
) *LinkPoller {
	if intervalSecs <= 0 {
		intervalSecs = 30
	}
	return &LinkPoller{
		linkRepo:      linkRepo,
		statusService: statusService,
		intervalSecs:  intervalSecs,
		log:           log,
	}
}

// Run bloqueia a correr o ciclo de polling até o contexto ser cancelado.
// Chamar como goroutine a partir de cmd/scheduler/main.go, tal como os
// outros pollers (SNMP, Nagios, ComAp, NetEco) já existentes.
func (p *LinkPoller) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(p.intervalSecs) * time.Second)
	defer ticker.Stop()

	p.log.Infof("link_poller: iniciado com intervalo de %ds", p.intervalSecs)

	for {
		select {
		case <-ctx.Done():
			p.log.Infof("link_poller: a terminar")
			return
		case <-ticker.C:
			p.pollAll(ctx)
		}
	}
}

func (p *LinkPoller) pollAll(ctx context.Context) {
	// limit/offset generosos — número de links de trânsito é pequeno
	// (dezenas, não centenas), ao contrário do parque de torres.
	links, _, err := p.linkRepo.List(ctx, 500, 0)
	if err != nil {
		p.log.Errorf("link_poller: falha ao listar links: %v", err)
		return
	}

	for _, link := range links {
		if err := p.statusService.EvaluateLink(ctx, link); err != nil {
			p.log.Errorf("link_poller: falha ao avaliar link %s (%s): %v", link.LinkID, link.Name, err)
			continue
		}
	}
}
