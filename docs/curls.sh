#!/usr/bin/env bash
# =============================================================================
# Report Service — Documentação de Endpoints Internos
# Base URL: http://localhost:8083
# ATENÇÃO: estes endpoints são chamados pelo API Gateway, não pelo cliente final
# =============================================================================

BASE_URL="http://localhost:8083"
REPORT_ID="${REPORT_ID:-SEU_REPORT_ID_AQUI}"

# ─── Health Check ─────────────────────────────────────────────────────────────
curl -i "${BASE_URL}/ping"

# Resposta: 200 pong

# ─── Leitura de Relatório (chamado pelo Gateway) ──────────────────────────────
# GET /internal/reports/:reportId
# reportId deve ser um UUID válido
# O reportId é retornado pelo GET /api/process/:id/status quando status = ANALISADO
#
curl -i "${BASE_URL}/internal/reports/${REPORT_ID}" \
  -H "X-Request-ID: test-report-001"

# Resposta esperada (200 OK):
# {
#   "report_id": "660e8400-e29b-41d4-a716-446655440000",
#   "process_id": "550e8400-e29b-41d4-a716-446655440000",
#   "components": [
#     "API Gateway (porta 8080)",
#     "Upload Orchestrator Service",
#     "PostgreSQL (status tracking)",
#     "MinIO (object storage)",
#     "RabbitMQ (message broker)",
#     "Processing Service (AI worker)",
#     "DynamoDB Local (job history)",
#     "Report Service",
#     "MongoDB (relatórios)"
#   ],
#   "risks": [
#     "Comunicação entre serviços sem mTLS",
#     "DynamoDB sem backup configurado",
#     "Sem circuit breaker nas chamadas upstream"
#   ],
#   "recommendations": [
#     "Adicionar mTLS entre serviços internos",
#     "Configurar DynamoDB Point-in-Time Recovery",
#     "Implementar circuit breaker com retry exponencial"
#   ],
#   "created_at": "2026-05-16T22:05:00Z"
# }

# ─── Casos de Erro ────────────────────────────────────────────────────────────

# reportId com formato inválido → 400
curl -i "${BASE_URL}/internal/reports/nao-e-uuid" \
  -H "X-Request-ID: test-err-001"

# reportId UUID inexistente → 404
curl -i "${BASE_URL}/internal/reports/00000000-0000-0000-0000-000000000000" \
  -H "X-Request-ID: test-err-002"

# ─── Verificação no MongoDB ───────────────────────────────────────────────────
# Conecte ao MongoDB para inspecionar relatórios diretamente:
#
#   docker exec -it hacka-mongodb-1 mongosh \
#     "mongodb://report:dev_mongo_pass@localhost:27017/reports?authSource=admin" \
#     --eval "db.reports.find({}, {raw_response: 0}).pretty()"
#
# Contar total de relatórios:
#   docker exec -it hacka-mongodb-1 mongosh \
#     "mongodb://report:dev_mongo_pass@localhost:27017/reports?authSource=admin" \
#     --eval "db.reports.countDocuments()"
#
# Buscar por process_id:
#   docker exec -it hacka-mongodb-1 mongosh \
#     "mongodb://report:dev_mongo_pass@localhost:27017/reports?authSource=admin" \
#     --eval "db.reports.findOne({process_id: 'SEU_PROCESS_ID'})"

# ─── Estado do Consumer RabbitMQ ─────────────────────────────────────────────
# Verificar se o consumer está ativo na report.queue:
#
#   curl -s -u guest:dev_rabbitmq_pass \
#     http://localhost:15672/api/queues/%2F/report.queue | python3 -m json.tool
