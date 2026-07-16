// Comando mockreceiver: simula o endpoint HTTP que o LogSavior vai expor
// futuramente para receber webhooks (do Zabbix ou, por enquanto, do gerador
// sintético). Serve para validar de ponta a ponta que os eventos chegam no
// formato esperado, antes mesmo de o LogSavior de verdade existir.
package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/arthurztt/logsavior/internal/domain"
)

func main() {
	addr := flag.String("addr", ":9090", "endereço para escutar (ex: :9090)")
	outFile := flag.String("out", "received_events.jsonl", "arquivo onde os eventos recebidos serão gravados")
	flag.Parse()

	f, err := os.OpenFile(*outFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("erro ao abrir arquivo de saída: %v", err)
	}
	defer f.Close()

	var mu sync.Mutex

	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "método não permitido", http.StatusMethodNotAllowed)
			return
		}

		var e domain.Event
		if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
			http.Error(w, "payload inválido", http.StatusBadRequest)
			return
		}

		log.Printf("[mockreceiver] recebido: host=%s severidade=%s trigger=%q",
			e.Host, e.Severity, e.Trigger)

		mu.Lock()
		body, _ := json.Marshal(e)
		f.Write(append(body, '\n'))
		mu.Unlock()

		w.WriteHeader(http.StatusOK)
	})

	log.Printf("[mockreceiver] escutando em %s (endpoint: /webhook, saída: %s)", *addr, *outFile)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
