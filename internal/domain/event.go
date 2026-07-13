// Package domain contém o modelo de dados central do projeto: o Event.
// Essa struct é o "contrato" compartilhado entre qualquer fonte de eventos
// (gerador sintético, webhook do Zabbix, Windows Event Log, etc.) e qualquer
// consumidor (o mockreceiver hoje, o LogSavior de verdade amanhã).
package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Severity espelha os níveis de severidade de trigger do Zabbix.
// Mantemos os mesmos nomes para que, quando o LogSavior passar a receber
// eventos reais do Zabbix, nenhuma lógica de negócio precise mudar.
type Severity int

const (
	SeverityNotClassified Severity = iota
	SeverityInformation
	SeverityWarning
	SeverityAverage
	SeverityHigh
	SeverityDisaster
)

var severityNames = map[Severity]string{
	SeverityNotClassified: "NotClassified",
	SeverityInformation:   "Information",
	SeverityWarning:       "Warning",
	SeverityAverage:       "Average",
	SeverityHigh:          "High",
	SeverityDisaster:      "Disaster",
}

var severityFromName = buildSeverityLookup()

func buildSeverityLookup() map[string]Severity {
	m := make(map[string]Severity, len(severityNames))
	for k, v := range severityNames {
		m[strings.ToLower(v)] = k
	}
	return m
}

func (s Severity) String() string {
	if name, ok := severityNames[s]; ok {
		return name
	}
	return "Unknown"
}

// ParseSeverity converte uma string (ex: "Average", "disaster", "HIGH")
// no valor de Severity correspondente.
func ParseSeverity(raw string) (Severity, error) {
	if sev, ok := severityFromName[strings.ToLower(strings.TrimSpace(raw))]; ok {
		return sev, nil
	}
	return SeverityNotClassified, fmt.Errorf("severidade desconhecida: %q", raw)
}

// IsAtLeast retorna true se a severidade for igual ou mais crítica que 'min'.
// Útil para a lógica de flush do LogSavior (ex: sev.IsAtLeast(domain.SeverityAverage)).
func (s Severity) IsAtLeast(min Severity) bool {
	return s >= min
}

// Event representa um evento de monitoramento normalizado.
// Os nomes de campo JSON (eventid, host, trigger_name, severity...) foram
// escolhidos para espelhar o que normalmente se extrai das macros do Zabbix
// em uma Action com media type Webhook (ex: {EVENT.ID}, {HOST.NAME},
// {TRIGGER.NAME}, {TRIGGER.SEVERITY}).
type Event struct {
	ID        string
	Host      string
	Trigger   string
	Severity  Severity
	Timestamp time.Time
	Message   string
}

// eventJSON é a representação "on-the-wire" do Event, usada para
// serializar/desserializar Severity como texto legível (ex: "Disaster")
// em vez de um número inteiro interno.
type eventJSON struct {
	ID        string    `json:"eventid"`
	Host      string    `json:"host"`
	Trigger   string    `json:"trigger_name"`
	Severity  string    `json:"severity"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

// MarshalJSON serializa o Event no formato que trafega entre o gerador,
// o webhook e o receptor.
func (e Event) MarshalJSON() ([]byte, error) {
	return json.Marshal(eventJSON{
		ID:        e.ID,
		Host:      e.Host,
		Trigger:   e.Trigger,
		Severity:  e.Severity.String(),
		Timestamp: e.Timestamp,
		Message:   e.Message,
	})
}

// UnmarshalJSON desserializa um payload recebido (ex: no mockreceiver),
// convertendo a severidade textual de volta para o tipo Severity.
func (e *Event) UnmarshalJSON(data []byte) error {
	var raw eventJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	sev, err := ParseSeverity(raw.Severity)
	if err != nil {
		return err
	}

	e.ID = raw.ID
	e.Host = raw.Host
	e.Trigger = raw.Trigger
	e.Severity = sev
	e.Timestamp = raw.Timestamp
	e.Message = raw.Message
	return nil
}
