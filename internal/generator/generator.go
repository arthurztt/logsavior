package generator

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/arthurztt/logsavior/internal/domain"
)

// EventSender é implementado por qualquer destino capaz de receber os
// eventos gerados: stdout, um arquivo local (JSONL) ou um webhook HTTP.
// É a mesma abstração que permitirá, no futuro, plugar o LogSavior de
// verdade recebendo eventos reais do Zabbix sem mudar essa camada.
type EventSender interface {
	Send(ctx context.Context, e domain.Event) error
}

// Generator executa um Scenario: dispara ruído de fundo em background e,
// em paralelo, percorre a lista de bursts respeitando os atrasos configurados.
type Generator struct {
	scenario *Scenario
	sender   EventSender
	rng      *rand.Rand
	eventSeq int
}

func New(scenario *Scenario, sender EventSender) *Generator {
	return &Generator{
		scenario: scenario,
		sender:   sender,
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Run inicia a simulação e bloqueia até o cenário terminar ou o contexto
// ser cancelado (ex: Ctrl+C).
func (g *Generator) Run(ctx context.Context) error {
	log.Printf("[generator] iniciando cenário: %s", g.scenario.Name)
	if g.scenario.Description != "" {
		log.Printf("[generator] %s", g.scenario.Description)
	}

	noiseCtx, stopNoise := context.WithCancel(ctx)
	defer stopNoise()

	noiseDone := make(chan struct{})
	if g.scenario.BackgroundNoise.Enabled {
		go func() {
			defer close(noiseDone)
			g.runBackgroundNoise(noiseCtx)
		}()
	} else {
		close(noiseDone)
	}

	for i, burst := range g.scenario.Bursts {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(burst.DelayBeforeMs) * time.Millisecond):
		}

		log.Printf("[generator] iniciando burst %d/%d - severidade=%s count=%d hosts_envolvidos=%d",
			i+1, len(g.scenario.Bursts), burst.Severity, burst.Count, burst.HostsInvolved)

		if err := g.runBurst(ctx, burst); err != nil {
			return err
		}
	}

	log.Printf("[generator] cenário finalizado, encerrando ruído de fundo")
	stopNoise()
	<-noiseDone
	return nil
}

// runBackgroundNoise emite eventos de baixa severidade periodicamente,
// escolhendo host e severidade aleatoriamente dentro do pool configurado.
func (g *Generator) runBackgroundNoise(ctx context.Context) {
	noise := g.scenario.BackgroundNoise
	everyMs := noise.EveryMs
	if everyMs <= 0 {
		everyMs = 5000
	}
	ticker := time.NewTicker(time.Duration(everyMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if len(noise.Severities) == 0 {
				continue
			}
			host := g.randomHost()
			sevName := noise.Severities[g.rng.Intn(len(noise.Severities))]
			sev, err := domain.ParseSeverity(sevName)
			if err != nil {
				log.Printf("[generator] severidade de ruído inválida: %v", err)
				continue
			}
			g.dispatch(ctx, g.buildEvent(host, sev))
		}
	}
}

// runBurst dispara 'Count' eventos de uma severidade específica, distribuídos
// ciclicamente entre 'HostsInvolved' hosts distintos escolhidos aleatoriamente.
// Isso simula tanto "um único dispositivo piscando" (hosts_involved=1) quanto
// "vários dispositivos caindo juntos" (hosts_involved=N), que é o padrão real
// de uma queda geral de rede.
func (g *Generator) runBurst(ctx context.Context, burst BurstConfig) error {
	sev, err := domain.ParseSeverity(burst.Severity)
	if err != nil {
		return fmt.Errorf("burst com severidade inválida: %w", err)
	}

	hostsInvolved := burst.HostsInvolved
	if hostsInvolved <= 0 || hostsInvolved > len(g.scenario.Hosts) {
		hostsInvolved = len(g.scenario.Hosts)
	}
	involvedHosts := g.pickRandomHosts(hostsInvolved)

	for i := 0; i < burst.Count; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		host := involvedHosts[i%len(involvedHosts)]
		g.dispatch(ctx, g.buildEvent(host, sev))

		if burst.IntervalMs > 0 && i < burst.Count-1 {
			time.Sleep(time.Duration(burst.IntervalMs) * time.Millisecond)
		}
	}

	return nil
}

// buildEvent monta um Event a partir de um host, escolhendo um trigger
// plausível para aquele host e atribuindo um ID sequencial e o timestamp atual.
func (g *Generator) buildEvent(host HostProfile, sev domain.Severity) domain.Event {
	g.eventSeq++
	trigger := host.Triggers[g.rng.Intn(len(host.Triggers))]

	return domain.Event{
		ID:        fmt.Sprintf("%d", 100000+g.eventSeq),
		Host:      host.Name,
		Trigger:   trigger,
		Severity:  sev,
		Timestamp: time.Now(),
		Message:   fmt.Sprintf("[%s] %s - %s", sev.String(), host.Name, trigger),
	}
}

func (g *Generator) dispatch(ctx context.Context, e domain.Event) {
	if err := g.sender.Send(ctx, e); err != nil {
		log.Printf("[generator] erro ao enviar evento %s: %v", e.ID, err)
		return
	}
	log.Printf("[generator] evento enviado: host=%s severidade=%s trigger=%q",
		e.Host, e.Severity, e.Trigger)
}

func (g *Generator) randomHost() HostProfile {
	return g.scenario.Hosts[g.rng.Intn(len(g.scenario.Hosts))]
}

// pickRandomHosts seleciona 'n' hosts distintos, sem repetição, usando uma
// permutação aleatória dos índices disponíveis.
func (g *Generator) pickRandomHosts(n int) []HostProfile {
	perm := g.rng.Perm(len(g.scenario.Hosts))
	picked := make([]HostProfile, 0, n)
	for i := 0; i < n && i < len(perm); i++ {
		picked = append(picked, g.scenario.Hosts[perm[i]])
	}
	return picked
}
