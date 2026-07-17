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

func runPolling() {
	
}
