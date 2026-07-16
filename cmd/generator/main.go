// Comando generator: lê um arquivo de cenário e simula eventos de
// monitoramento de rede, enviando-os para stdout, um arquivo local ou
// um webhook HTTP (simulando o que o Zabbix faria de verdade).
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/arthurztt/logsavior/internal/generator"
	"github.com/arthurztt/logsavior/internal/sender"
)

func main() {
	scenarioPath := flag.String("scenario", "scenarios/network_outage.json", "caminho do arquivo de cenário (JSON)")
	mode := flag.String("mode", "stdout", "destino dos eventos: stdout | file | webhook")
	target := flag.String("target", "", "URL do webhook (modo webhook) ou caminho do arquivo (modo file)")
	flag.Parse()

	scenario, err := generator.LoadScenario(*scenarioPath)
	if err != nil {
		log.Fatalf("erro ao carregar cenário: %v", err)
	}

	var evtSender generator.EventSender
	switch *mode {
	case "stdout":
		evtSender = sender.NewStdoutSender()

	case "webhook":
		if *target == "" {
			log.Fatal("modo webhook requer -target <url>, ex: -target http://localhost:9090/webhook")
		}
		evtSender = sender.NewWebhookSender(*target)

	case "file":
		path := *target
		if path == "" {
			path = "synthetic_events.jsonl"
		}
		fs, err := sender.NewFileSender(path)
		if err != nil {
			log.Fatalf("erro ao configurar sender de arquivo: %v", err)
		}
		defer fs.Close()
		evtSender = fs

	default:
		log.Fatalf("modo desconhecido: %q (use stdout, file ou webhook)", *mode)
	}

	// Encerra a simulação de forma limpa em Ctrl+C / SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	gen := generator.New(scenario, evtSender)
	if err := gen.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("erro durante a execução do cenário: %v", err)
	}
}
