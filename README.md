# Clima por CEP com Observabilidade

Sistema distribuído em Go com dois microsserviços que, juntos, consultam a temperatura atual de uma cidade a partir do CEP. Todas as requisições são rastreadas com OpenTelemetry, passam por um OTEL Collector e podem ser visualizadas no Zipkin.

## Como funciona

```
Usuário ──POST──▶ Serviço A ──POST──▶ Serviço B ──▶ ViaCEP
  (8080)          (validação)       (orquestração) ──▶ Open-Meteo
                       │                  │
                       └──── traces ──────┴──▶ OTEL Collector ──▶ Zipkin (9411)
```

- **Serviço A (input)**: recebe o CEP via `POST`, valida o formato e repassa para o Serviço B. A resposta do Serviço B volta para o usuário do jeito que veio.
- **Serviço B (orquestração)**: busca a cidade no ViaCEP, descobre as coordenadas dela na API de geocodificação do Open-Meteo, consulta a temperatura atual e devolve o valor em Celsius, Fahrenheit e Kelvin.
- **OTEL Collector**: recebe os spans dos dois serviços via OTLP gRPC (porta `4317`) e exporta para o Zipkin.
- **Zipkin**: interface para navegar pelos traces.

## Pré-requisitos

- Docker
- Docker Compose

Para rodar sem Docker você vai precisar do Go 1.27 ou mais recente.

## Subindo o projeto

Na raiz do repositório, rode:

```bash
docker compose up --build
```

Isso sobe quatro containers:

| Container | Porta | Descrição |
| --- | --- | --- |
| `servico-a` | 8080 | Porta de entrada do sistema |
| `servico-b` | 8181 | Lógica de negócio |
| `otel-collector` | 4317 | Recebe os traces dos serviços |
| `zipkin` | 9411 | Visualização dos traces |

Para derrubar tudo:

```bash
docker compose down
```

## Fazendo requisições no Serviço A

O Serviço A aceita `POST` na raiz com um JSON contendo o CEP como string de 8 dígitos.

```bash
curl -X POST http://localhost:8080 \
  -H "Content-Type: application/json" \
  -d '{"cep": "29902555"}'
```

### Respostas possíveis

**200 OK**, CEP válido e encontrado:

```json
{"city":"Linhares","temp_C":32.4,"temp_F":90.32,"temp_K":305.4}
```

**422 Unprocessable Entity**, CEP fora do formato (menos ou mais de 8 dígitos, letras, hífen ou enviado como número):

```bash
curl -X POST http://localhost:8080 -H "Content-Type: application/json" -d '{"cep": "123"}'
curl -X POST http://localhost:8080 -H "Content-Type: application/json" -d '{"cep": 29902555}'
```

```
invalid zipcode
```

**404 Not Found**, CEP no formato certo mas que não existe:

```bash
curl -X POST http://localhost:8080 -H "Content-Type: application/json" -d '{"cep": "99999999"}'
```

```
can not find zipcode
```

Se quiser chamar o Serviço B direto, ele aceita o mesmo payload em `http://localhost:8181`.

## Vendo os traces no Zipkin

1. Faça pelo menos uma requisição no Serviço A.
2. Abra [http://localhost:9411](http://localhost:9411) no navegador.
3. Clique em **Run Query** (ou filtre por `serviceName=servico-a`).
4. Abra um dos traces da lista.

Os spans podem levar alguns segundos para aparecer, porque o collector envia em lotes.

Dentro de cada trace dá para ver o caminho inteiro da requisição:

```
servico-a  post                  (span automático do HTTP server)
└─ servico-a  servico-a-request
   └─ servico-b  post            (span automático do HTTP server)
      └─ servico-b  servico-b-request
         ├─ servico-b  busca-cep          (tempo da consulta ao ViaCEP)
         └─ servico-b  busca-temperatura  (tempo da consulta ao Open-Meteo)
```

Os spans `busca-cep` e `busca-temperatura` são criados manualmente e medem o tempo gasto em cada API externa. Quando alguma delas falha, o erro fica registrado no próprio span.

## Variáveis de ambiente

Os valores abaixo já estão configurados no `docker-compose.yaml`. Os padrões servem para rodar localmente fora do Docker.

**Serviço A**

| Variável | Padrão |
| --- | --- |
| `HTTP_PORT` | `:8080` |
| `SERVICO_B_URL` | `http://localhost:8181` |
| `OTEL_SERVICE_NAME` | `servico-a` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` |
| `REQUEST_NAME_OTEL` | `servico-a-request` |

**Serviço B**

| Variável | Padrão |
| --- | --- |
| `HTTP_PORT` | `:8181` |
| `VIACEP_URL` | `https://viacep.com.br` |
| `GEOCODING_URL` | `https://geocoding-api.open-meteo.com` |
| `PREVISAO_URL` | `https://api.open-meteo.com` |
| `OTEL_SERVICE_NAME` | `servico-b` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` |
| `REQUEST_NAME_OTEL` | `servico-b-request` |

## Rodando sem Docker

Suba só o Zipkin e o collector com o Compose e rode os serviços pelo Go, cada um em um terminal.

```bash
docker compose up zipkin otel-collector
go run ./cmd/servico-b
go run ./cmd/servico-a
```

## Testes

```bash
go test ./...
```

Os testes cobrem a validação do CEP e o handler do Serviço B, usando um servidor HTTP falso no lugar do ViaCEP e do Open-Meteo.

## Estrutura

```
cmd/
  servico-a/          ponto de entrada do Serviço A
  servico-b/          ponto de entrada do Serviço B
internal/
  cep/                leitura do payload e validação do CEP
  servicoa/           servidor HTTP do Serviço A
  servicob/           servidor HTTP, clientes ViaCEP e Open-Meteo, conversão de temperatura
  telemetria/         configuração do TracerProvider e do exporter OTLP
.docker/
  otel-collector-config.yaml
docker-compose.yaml
Dockerfile
```

## Observações

- A conversão para Kelvin segue a fórmula do enunciado, `K = C + 273`.
- Os valores de temperatura são arredondados para duas casas decimais.
- Quando existem várias cidades com o mesmo nome, o Serviço B usa o estado retornado pelo ViaCEP para escolher a certa na geocodificação.
