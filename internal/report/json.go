package report

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/kneumoin/nepal/internal/model"
)

func WriteJSON(path string, result *model.EvaluationResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
