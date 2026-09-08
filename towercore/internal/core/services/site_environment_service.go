// Package services contém implementações de lógica de negócio para varias entidades do domínio.
package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"towercore/internal/core/domain"
	"towercore/internal/core/interfaces"
)

// SiteEnvironmentService gerencia a coleta e consulta de medições de ambiente e físico do site.
type SiteEnvironmentService struct {
	repo interfaces.SiteEnvironmentRepository
}

// NewSiteEnvironmentService cria um novo serviço de medições de ambiente do site.
func NewSiteEnvironmentService(repo interfaces.SiteEnvironmentRepository) *SiteEnvironmentService {
	return &SiteEnvironmentService{repo: repo}
}

// CollectEnvironment aceita uma medição coletada de ambiente do site.
// Esta é a porta de entrada para dados vindos de polling SNMP, scripts locais, ou agentes.
func (s *SiteEnvironmentService) CollectEnvironment(ctx context.Context, env *domain.SiteEnvironment) error {
	if env == nil {
		return errors.New("environment cannot be nil")
	}

	// Validar campos obrigatórios
	if env.SiteID == uuid.Nil {
		return errors.New("site_id é obrigatório")
	}
	if env.MeasuredAt.IsZero() {
		return errors.New("measured_at é obrigatório")
	}
	if env.SourcePoller == "" {
		return errors.New("source_poller é obrigatório")
	}
	if env.CollectionIntervalSec <= 0 {
		return errors.New("collection_interval_sec deve ser positivo")
	}
	if env.MeasuredAt.After(time.Now().Add(5 * time.Minute)) {
		return errors.New("measured_at não pode estar no futuro")
	}

	// Salvar no repositório
	if err := s.repo.Create(ctx, env); err != nil {
		return fmt.Errorf("falha ao coletar medição de ambiente: %w", err)
	}

	return nil
}

// CollectMultipleEnvironment aceita múltiplas medições de uma vez (ótimo para batch de polling).
func (s *SiteEnvironmentService) CollectMultipleEnvironment(ctx context.Context, envs []*domain.SiteEnvironment) error {
	if len(envs) == 0 {
		return nil
	}

	// Validar cada medição
	now := time.Now().UTC()
	for _, env := range envs {
		if env == nil {
			continue // Pular nils silenciosamente
		}
		if env.SiteID == uuid.Nil {
			return errors.New("site_id é obrigatório em uma das medições")
		}
		if env.MeasuredAt.IsZero() {
			env.MeasuredAt = now // Definir timestamp se não fornecido
		}
		if env.SourcePoller == "" {
			return errors.New("source_poller é obrigatório em uma das medições")
		}
		if env.CollectionIntervalSec <= 0 {
			return errors.New("collection_interval_sec deve ser positivo em uma das medições")
		}
		if env.MeasuredAt.After(now.Add(5 * time.Minute)) {
			return errors.New("measured_at não pode estar no futuro em uma das medições")
		}
	}

	// Salvar todas de uma vez
	if err := s.repo.CreateMany(ctx, envs); err != nil {
		return fmt.Errorf("falha ao coletar medições múltiplas de ambiente: %w", err)
	}

	return nil
}

// GetLatestEnvironment retorna a medição mais recente de ambiente para um site.
func (s *SiteEnvironmentService) GetLatestEnvironment(ctx context.Context, siteID uuid.UUID) (*domain.SiteEnvironment, error) {
	if siteID == uuid.Nil {
		return nil, errors.New("site_id cannot be nil")
	}
	env, err := s.repo.GetLatest(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("falha ao buscar medição mais recente de ambiente: %w", err)
	}
	return env, nil
}

// GetEnvironmentHistory retorna o histórico de medições de ambiente para um site.
func (s *SiteEnvironmentService) GetEnvironmentHistory(ctx context.Context, siteID uuid.UUID, limit int, offset int, measuredAfter *time.Time, measuredBefore *time.Time) ([]*domain.SiteEnvironment, int, error) {
	filter := &domain.SiteEnvironmentFilter{
		SiteID:           siteID,
		Limit:            limit,
		Offset:           offset,
		OrderBy:          []string{"measured_at DESC"},
		MeasuredAtAfter:  measuredAfter,
		MeasuredAtBefore: measuredBefore,
	}

	envs, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("falha ao buscar histórico de ambiente: %w", err)
	}
	return envs, total, nil
}
