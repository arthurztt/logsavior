package generator

import (
	"encoding/json"
	"fmt"
	"os"
)

// HostProfile descreve um dispositivo simulado (switch, roteador, link, AP...)
// e os nomes de trigger plausíveis para ele. Cada tipo de dispositivo tem
// problemas diferentes, então cada host tem sua própria lista de triggers.
type HostProfile struct {
	Name     string   `json:"name"`
	Triggers []string `json:"triggers"`
}

// NoiseConfig configura o "ruído de fundo": eventos de baixa severidade
// emitidos periodicamente, simulando o funcionamento normal do ambiente
// entre os incidentes (é assim que uma rede real se comporta: nem tudo
// é silêncio ou crise, existe um ruído constante de baixa severidade).
type NoiseConfig struct {
	Enabled    bool     `json:"enabled"`
	EveryMs    int      `json:"every_ms"`
	Severities []string `json:"severities"`
}

// BurstConfig configura uma rajada de eventos de uma severidade específica,
// simulando uma escalada de problema. Um cenário de crise real normalmente
// é modelado como uma sequência de bursts com severidade crescente.
type BurstConfig struct {
	DelayBeforeMs int    `json:"delay_before_ms"`
	Severity      string `json:"severity"`
	Count         int    `json:"count"`
	IntervalMs    int    `json:"interval_ms"`
	HostsInvolved int    `json:"hosts_involved"`
}

// Scenario descreve um cenário de simulação completo: os hosts existentes,
// o ruído de fundo e a sequência de bursts que representam a crise.
type Scenario struct {
	Name            string        `json:"name"`
	Description     string        `json:"description"`
	Hosts           []HostProfile `json:"hosts"`
	BackgroundNoise NoiseConfig   `json:"background_noise"`
	Bursts          []BurstConfig `json:"bursts"`
}

// LoadScenario lê e valida um arquivo de cenário em JSON.
func LoadScenario(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler cenário %q: %w", path, err)
	}

	var s Scenario
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("erro ao parsear cenário %q: %w", path, err)
	}

	if len(s.Hosts) == 0 {
		return nil, fmt.Errorf("cenário %q não define nenhum host", path)
	}
	for i, h := range s.Hosts {
		if len(h.Triggers) == 0 {
			return nil, fmt.Errorf("host %q (índice %d) não define nenhum trigger", h.Name, i)
		}
	}

	return &s, nil
}
