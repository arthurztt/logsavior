package dbpoller

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/arthurztt/logsavior/internal/domain"
	"github.com/arthurztt/logsavior/internal/eventsource"
	"github.com/arthurztt/logsavior/internal/sender"
)

func main() {
	dsn := flag.String("dsn", "logsavior:logsavior@tcp(127.0.0.1:3306)/zabbix_mock?parseTime=true", "DSN de conexão com o MySQL")
	interval := flag.Duration("interval", 5*time.Second, "Intervalo ente cada consulta do banco")
	lookback := flag.Duration("lookback", 24*time.Hour, "Quanto tempo no passado considerar na primeira consulta")
	mode := flag.String("mode", "stdout", "Destino dos eventos: stdout | file | webhook")
	target := flag.String("target", "", "URL do Webhook (modo webhook) ou de arquivo (modo file)")
	flag.Parse()

	source, err := eventsource.NewMySQLSource(*dsn)
	if err != nil {
		log.Fatal("erro ao se conectar na fonte de dados: %v", err)
	}
	defer source.Close()

	var sink domain.EventSink
	switch *mode {
	case "stdout":
		sink = sender.NewStdoutSender()

	case "webhook":
		if *target == "" {
			log.Fatal("modo webhook requer -target <url>")
		}
		sink = sender.NewWebhookSender(*target)
	case "file":
		path := *target
		if path == "" {
			path = "db_events.jsonl"
		}
		fs, err := sender.NewFileSender(path)
		if err != nil {
			log.Fatal("erro ao configurar sender de arquivo: %v", err)
		}
		defer fs.Close()
		sink = fs

	default:
		log.Fatalf("modo desconhecido: %q (use stdout, file ou webhook)", mode)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Printf("[dbpoller] iniciando: intervalo=%s lookback_inicial=%s", *interval, *lookback)
	runPolling(ctx, source, sink, *interval, *lookback)
}

func runPolling(ctx contentx.Context, source domain.EventSource, sink domain.EventSink, interval, lookback time.Duration) {
	since := time.Now().Add(-lookback)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	poll := func() {
		events, err = source.FetchEvents(ctx, since)
		if err != nil {
			lof.Printf("[dbpoller] erro ao consultar a fonte de eventos: %v", err)
			return
		}

		for _, e := range events {
			if err := sink.Send(ctx, e); err != nil {
				log.Printf("[dbpoller] erro ao enviar eventos: %v", e.ID, err)
				continue
			}
			log.Printf("[dbpoller] evento encaminhado: host:%s severidade:%s trigger:%q",
				e.Host, e.ID, e.Trigger)

			if e.Timestamp.After(since) {
				since = e.Timestamp
			}
		}
	}
	poll()
	for {
		select {
		case <-ctx.Dome():
			log.Printf("[dbpoller] encerrando")
			return
		case <-ticker.C:
			poll()
		}
	}
}
