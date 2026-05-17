# Report Service

Serviço responsável por consolidar os resultados da análise de IA em relatórios técnicos persistentes, notificar o orquestrador sobre a conclusão do fluxo e expor os relatórios via REST.

---

## Descrição do Problema

Após a análise do diagrama pela IA, o resultado precisa ser persistido de forma estruturada e consultável, com identificador próprio que permita ao cliente recuperá-lo independentemente do processo de upload. O relatório também precisa fechar o ciclo de rastreamento, notificando o orquestrador para que o status final do processo seja atualizado.

**Desafios específicos endereçados:**

- Persistir relatórios com identificador UUID próprio (desacoplado do `process_id`)
- Garantir que o orquestrador seja notificado mesmo em caso de falha na persistência
- Validar o `report_id` antes de qualquer consulta ao banco (prevenção de IDOR)
- Fechar o ciclo do saga coreografado publicando no `report.topic`

---

## Arquitetura Proposta

```mermaid
graph TD
    RQ["RabbitMQ: report.queue"]
    GW["API Gateway"]

    subgraph RS["Report Service :8083"]
        CON["consumer/report_queue.go"]
        UC["usecase/create_report.go"]
        MG[("repository/report_mongo.go\n(MongoDB)")]
        Q["queue/rabbitmq.go"]
        H["handler/report.go\nGET /internal/reports/:reportId\nuuid.Parse → report_mongo.go"]
        CON --> UC
        UC --> MG
        UC --> Q
    end

    RT["report.topic (fanout)\n→ upload-orchestrator: ANALISADO"]

    RQ -->|consume| CON
    Q --> RT
    GW --> H
```

### Camadas internas (Clean Architecture)

```
internal/
├── domain/
│   └── report.go            ← Report, Analysis, ErrReportNotFound, ErrInvalidID
├── usecase/
│   ├── ports.go             ← ReportRepository, EventPublisher (interfaces)
│   ├── create_report.go     ← gera UUID → MongoDB → publica report.topic
│   └── get_report.go        ← uuid.Parse + FindByID
├── repository/
│   └── report_mongo.go      ← InsertOne (_id=UUID), FindByID, índice em process_id
├── queue/
│   └── rabbitmq.go          ← DeclareExchange (fanout), PublishToExchange, Consume
├── consumer/
│   └── report_queue.go      ← New Relic transaction por mensagem, Nack sem requeue em erro
└── handler/
    └── report.go            ← GET /internal/reports/:reportId com erros tipados
```

### Esquema MongoDB (collection `reports`)

| Campo | Tipo | Descrição |
|---|---|---|
| `_id` | String (UUID) | Identificador único do relatório |
| `process_id` | String | UUID do processo de upload |
| `components` | Array\<String\> | Componentes identificados no diagrama |
| `risks` | Array\<String\> | Riscos e vulnerabilidades detectados |
| `recommendations` | Array\<String\> | Recomendações técnicas |
| `raw_response` | String | Resposta bruta do modelo de IA |
| `created_at` | Date | Timestamp de criação |

> O campo `_id` usa UUID string (não ObjectId), permitindo lookup direto sem índice secundário.

---

## Fluxo da Solução

```mermaid
flowchart TD
    MSG["Mensagem: report.queue\n{process_id, analysis, raw_response}"]
    R1["Gera report_id = uuid.New()"]
    R2["MongoDB.InsertOne\n{_id: report_id, process_id, analysis, raw_response, created_at}"]
    OK{persistiu?}
    R3["Publica report.topic\n{report_id, event: report_created}\n→ upload-orchestrator: ANALISADO + report_id"]
    ACK["Ack()"]
    ERR["Publica report.topic {event: report_failed}\nNack(false, false)"]

    MSG --> R1 --> R2 --> OK
    OK -->|sim| R3 --> ACK
    OK -->|não| ERR

    subgraph GET["GET /internal/reports/:reportId"]
        G1["uuid.Parse(reportId)"]
        G2{válido?}
        G3["400 Bad Request"]
        G4["MongoDB.FindOne\n{_id: reportId}"]
        G5{encontrado?}
        G6["404 Not Found"]
        G7["200 {report_id, process_id,\ncomponents, risks,\nrecommendations, created_at}"]
        G1 --> G2
        G2 -->|não| G3
        G2 -->|sim| G4
        G4 --> G5
        G5 -->|não| G6
        G5 -->|sim| G7
    end
```

---

## Instruções de Execução

### Variáveis de ambiente

| Variável | Obrigatório | Padrão | Descrição |
|---|---|---|---|
| `MONGO_URI` | Sim | — | Ex: `mongodb://user:pass@mongodb:27017/reports?authSource=admin` |
| `RABBITMQ_URL` | Sim | — | Ex: `amqp://guest:pass@rabbitmq:5672/` |
| `PORT` | Não | `8083` | Porta HTTP |
| `MONGO_DB` | Não | `reports` | Nome do banco MongoDB |
| `REPORT_QUEUE` | Não | `report.queue` | Fila de entrada |
| `REPORT_TOPIC` | Não | `report.topic` | Exchange de notificação de conclusão |
| `NEW_RELIC_LICENSE_KEY` | Não | — | Chave New Relic |
| `NEW_RELIC_APP_NAME` | Não | — | Nome da app no New Relic |

### Executar com Docker Compose (recomendado)

```bash
# A partir da raiz do projeto
docker compose up --build -d report-service

# Verificar saúde
curl http://localhost:8083/ping

# Acompanhar logs
docker logs -f hacka-report-service-1
```

### Executar localmente (desenvolvimento)

```bash
cd report-service
go mod download

export MONGO_URI="mongodb://report:dev_mongo_pass@localhost:27017/reports?authSource=admin"
export RABBITMQ_URL="amqp://guest:dev_rabbitmq_pass@localhost:5672/"

go run main.go
```

### Inspecionar relatórios no MongoDB

```bash
# Listar últimos relatórios (sem campo raw_response)
docker exec -it hacka-mongodb-1 mongosh \
  "mongodb://report:dev_mongo_pass@localhost:27017/reports?authSource=admin" \
  --eval "db.reports.find({}, {raw_response: 0}).sort({created_at: -1}).limit(5).pretty()"

# Contar total de relatórios
docker exec -it hacka-mongodb-1 mongosh \
  "mongodb://report:dev_mongo_pass@localhost:27017/reports?authSource=admin" \
  --eval "db.reports.countDocuments()"

# Buscar relatório por process_id
docker exec -it hacka-mongodb-1 mongosh \
  "mongodb://report:dev_mongo_pass@localhost:27017/reports?authSource=admin" \
  --eval "db.reports.findOne({process_id: 'SEU-PROCESS-ID'}, {raw_response: 0})"
```

### Endpoints

| Método | Rota | Descrição |
|---|---|---|
| `GET` | `/ping` | Healthcheck |
| `GET` | `/internal/reports/:reportId` | Busca relatório por ID |

---

## Segurança

### Requisitos básicos adotados

| Controle | Implementação |
|---|---|
| Validação de UUID | `uuid.Parse(reportId)` antes de qualquer consulta MongoDB — previne IDOR e buscas mal-formadas |
| Consulta por `_id` | `FindOne({_id: reportId})` — sem query construída com entrada do usuário |
| `_id` como UUID string | Lookup direto sem índice secundário — elimina risco de enumeração por ObjectId sequencial |
| Nack sem requeue | Mensagens inválidas descartadas definitivamente — sem loop infinito |
| Credenciais via env | `MONGO_URI` com usuário/senha, `RABBITMQ_URL` — sem hardcode |
| Índice em `process_id` | Criado na startup — sem full collection scan em buscas por processo |

### Validação de entradas não confiáveis

- **`reportId` (URL):** `uuid.Parse(reportId)` retorna `ErrInvalidID` → 400 antes de tocar o MongoDB. Sem `reportId` controlado pelo usuário chegando à query.
- **Mensagens RabbitMQ (`report.queue`):** `json.Unmarshal` + validação de `ProcessID` não vazio; `Nack(false, false)` em payload inválido ou `process_id` ausente.
- **Dados da análise (`analysis`, `raw_response`):** recebidos do processing-service (confiável internamente); armazenados como-estão no MongoDB — não são renderizados em HTML, não há risco de XSS na persistência.
- **MongoDB query:** `bson.M{"_id": id}` com valor string tipado — sem injeção de operadores MongoDB por entrada do usuário.

### Comunicação entre serviços

- Publicação no `report.topic` (fanout) após persistência no MongoDB — o orquestrador recebe `report_id` para completar o ciclo.
- Publicação de `report_failed` mesmo em caso de erro — garante que o orquestrador nunca fique em estado indeterminado.
- Todas as credenciais de conexão MongoDB e RabbitMQ via variável de ambiente.
- Timeout de conexão MongoDB: 10s no `Connect`; operações de leitura/escrita respeitam o `context` do request Gin (timeout do servidor HTTP).

### Principais riscos e limitações

| Risco | Severidade | Mitigação atual | Recomendação para produção |
|---|---|---|---|
| MongoDB sem TLS | Média | Rede Docker isolada | Habilitar TLS e autenticação por certificado |
| `raw_response` armazenado sem sanitização | Baixa | Não é renderizado em HTML | Sanitizar se futuramente exposto em interface web |
| Sem autenticação no endpoint `/internal/reports/:id` | Média | Endpoint não exposto ao cliente (apenas gateway acessa) | Adicionar validação de token interno entre serviços |
| Sem paginação na collection `reports` | Baixa | MVP — volume pequeno | Implementar paginação e índice TTL para relatórios antigos |
| Evento `report_failed` não tem retry | Média | Mensagem descartada; orquestrador marca como ERRO | Implementar dead letter queue com alerting |

---

### Build

```bash
cd report-service
go build ./...
go vet ./...
```
