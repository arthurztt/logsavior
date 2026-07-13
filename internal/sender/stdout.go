package sender

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/empresa/logsavior-generator/internal/domain"
)

// StdoutSender apenas imprime o JSON do evento no terminal.
// Útil para depurar um cenário sem precisar de nenhum receptor rodando.
type StdoutSender struct{}

func NewStdoutSender() *StdoutSender {
	return &StdoutSender{}
}

func (s *StdoutSender) Send(_ context.Context, e domain.Event) error {
	body, err := json.Marshal(e)
	if err != nil {
		return err
	}
	fmt.Println(string(body))
	return nil
}
