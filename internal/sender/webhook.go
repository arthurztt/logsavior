package sender

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/arthurztt/logsavior/internal/domain"
)

// WebhookSender envia cada evento via HTTP POST, simulando exatamente o que
// uma Action com media type "Webhook" do Zabbix faria ao disparar um problema.
// Isso permite testar o receptor (o futuro LogSavior) de ponta a ponta sem
// depender do Zabbix real.
type WebhookSender struct {
	URL    string
	client *http.Client
}

func NewWebhookSender(url string) *WebhookSender {
	return &WebhookSender{
		URL:    url,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

func (w *WebhookSender) Send(ctx context.Context, e domain.Event) error {
	body, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("erro ao serializar evento: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("erro ao montar requisição: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao enviar webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("destino retornou status %d", resp.StatusCode)
	}
	return nil
}
