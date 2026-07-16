package sender

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/arthurztt/logsavior/internal/domain"
)

// FileSender acrescenta cada evento como uma linha JSON (formato JSONL) em
// um arquivo local. Serve tanto para depuração quanto como uma simulação
// simples de "armazenamento local persistente" — que é a essência do que
// o LogSavior fará de verdade, salvando localmente mesmo sem rede.
type FileSender struct {
	mu   sync.Mutex
	file *os.File
}

func NewFileSender(path string) (*FileSender, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir arquivo %q: %w", path, err)
	}
	return &FileSender{file: f}, nil
}

func (fs *FileSender) Send(_ context.Context, e domain.Event) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	body, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if _, err := fs.file.Write(append(body, '\n')); err != nil {
		return err
	}
	return nil
}

func (fs *FileSender) Close() error {
	return fs.file.Close()
}
