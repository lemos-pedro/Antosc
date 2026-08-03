# Estado do AIP vs. plano original

## Feito (produção-ready nativo, sem Docker)

| Área | Estado |
|------|--------|
| Ingestão TowerCore + features + alertas Teams | ✅ original |
| Predição por torre (Python) + persistência | ✅ original |
| Assistente Ollama + tools + grounding + audit | ✅ original + portfolio tools |
| Normas de consumo + desvios | ✅ original |
| Causas de incidente + confirmação O&M | ✅ original |
| Relatórios semanais/mensais **por persona** + Resend | ✅ |
| Resumo executivo Ollama (CEO/Conselho) | ✅ |
| Ranking de risco IA nos relatórios | ✅ |
| Power BI (5 datasets REST) | ✅ |
| predict_batch (cron) | ✅ |
| Auth JWT + users + bootstrap | ✅ |
| RBAC por role | ✅ |
| Seed admin + personas | ✅ |
| Makefile, migrate, smoke, runbook | ✅ |

## Parcial / operacional (depende do ambiente)

| Item | Notas |
|------|--------|
| Dados reais nas previsões | Requer TowerCore + features ≥30 pts + Python inference up |
| Emails a chegar | Resend API key + REPORT_RECIPIENTS_* |
| Power BI dashboards | Endpoints prontos; **modelo .pbix** ainda não desenhado na UI |
| Ollama online | Assistente e resumos CEO dependem disto |

## Ainda não feito (roadmap “super modelo” / TowerCore-level)

### ML avançado (Fase 2–3 do plano inicial)
- [x] Feature engineering temporal (lags, rolling, slope, EWMA)
- [x] Sazonalidade Fourier (tod_sin/cos por série)
- [x] Cross-tower embeddings (PCA + /towers/{id}/similar)
- [x] Forecast multi-sinal v2 (EWMA + aceleradores)
- [x] XGBoost health opcional (`training/train_xgb_health.py`) quando há labels
- [ ] TFT / N-BEATS / Prophet com falhas rotuladas em massa
- [x] Explicabilidade (contributions + SHAP opcional) nos predicts
- [x] Re-treino mínimo via `make retrain` / scripts/retrain_models.sh
- [x] Anomaly v2 (features temporais) + sidecar `.meta.json` de versão
- [x] Export real `ai_features` → wide CSV (`scripts/export_ai_features.sh`)
- [x] Drift monitoring (PSI + mean shift) via `make drift`
- [x] MLflow opcional (log no retrain se MLFLOW_TRACKING_URI)
- [ ] Model Registry Staging/Production formal (promover modelos)
- [ ] Modelos prescritivos (“substituir bateria em X dias”)
- [ ] Risk score agregado CEO (Monte Carlo / sobrevivência)

### Produto / integração
- [x] Modelo Power BI completo (docs + M + DAX + export CSV) — .pbix montado no Desktop
- [ ] Ficheiro .pbix binário versionado (opcional, feito localmente)
- [x] Audit de logins (`auth_audit_log` + GET /api/v1/auth/audit)
- [x] Refresh tokens / logout / lockout após N falhas
- [ ] Docker Compose (adiado — restrições de PC)
- [ ] Notificações proativas Teams/Slack por role (além do alerta genérico)
- [ ] What-if no assistente (simulações)

### Qualidade
- [ ] Testes unitários/integração Go automatizados (além do smoke HTTP)
- [ ] CI (GitHub Actions) — build + vet + smoke contra Postgres de teste

## Prioridade recomendada a seguir

1. **Operacionalizar o que já existe** — migrate, seed, scheduler, predict_batch, 1 relatório de teste Resend, ligar 1 dataset no Power BI Desktop  
2. ~~Modelo Power BI~~ (docs/powerbi_modelo.md)  
3. **SHAP + melhorar forecast** — primeiro ganho ML visível  
4. **MLOps mínimo** (re-treino semanal + versionar modelo)  
5. Docker quando o ambiente permitir  

O núcleo “relatórios + BI + auth + assistente + batch” está **completo**. O salto para “super modelo preditivo enterprise” é sobretudo a **camada ML** (itens em falta acima).
