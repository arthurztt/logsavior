# LogSavior - Gerador de Eventos Sintéticos

Módulo inicial do projeto LogSavior. Simula eventos de monitoramento de rede
(no formato que o Zabbix enviaria via webhook) para permitir o desenvolvimento
e teste da lógica do LogSavior sem depender do ambiente de produção.

## Estrutura

```
cmd/generator/      -> CLI que roda um cenário e envia os eventos
cmd/mockreceiver/    -> servidor HTTP que simula o futuro endpoint do LogSavior
internal/domain/     -> modelo de dados Event/Severity, compartilhado por tudo
internal/generator/  -> motor de simulação (ruído de fundo + bursts)
internal/sender/     -> destinos possíveis: stdout, arquivo (JSONL) ou webhook HTTP
scenarios/           -> cenários de exemplo em JSON
```

## Como rodar

Compilar os dois binários:

```bash
go build -o bin/generator ./cmd/generator
go build -o bin/mockreceiver ./cmd/mockreceiver
```

### Teste rápido (sem receptor, só olhando no terminal)

```bash
./bin/generator -scenario scenarios/network_outage.json -mode stdout
```

### Teste de ponta a ponta (simulando o webhook real)

Em um terminal, suba o receptor:

```bash
./bin/mockreceiver -addr :9090 -out received_events.jsonl
```

Em outro terminal, rode o gerador apontando para ele:

```bash
./bin/generator -scenario scenarios/network_outage.json -mode webhook -target http://localhost:9090/webhook
```

Os eventos recebidos ficam salvos em `received_events.jsonl`, um JSON por linha.

### Salvando direto em arquivo (sem precisar do receptor)

```bash
./bin/generator -scenario scenarios/network_outage.json -mode file -target eventos.jsonl
```

## Criando seus próprios cenários

Copie `scenarios/network_outage.json` e ajuste:

- `hosts`: dispositivos simulados e seus triggers possíveis.
- `background_noise`: ruído de fundo (severidades leves, intervalo em ms).
- `bursts`: sequência de rajadas de eventos, cada uma com severidade, quantidade,
  intervalo entre eventos e quantos hosts distintos são afetados. Use
  `delay_before_ms` para controlar quando cada burst começa em relação ao anterior,
  simulando uma escalada de problema ao longo do tempo.

## Próximo passo

O pacote `internal/domain` (struct `Event`) e a interface `generator.EventSender`
foram desenhados para serem reaproveitados diretamente pelo LogSavior real:
basta implementar um novo `EventSource` que converse com a API do Zabbix (ou
receba o mesmo webhook) e a lógica de contador/janela/flush pode consumir os
mesmos `Event` gerados aqui.
