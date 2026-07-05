package report

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kneumoin/nepal/internal/model"
)

func WriteCSV(path string, result *model.EvaluationResult) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	headers := []string{
		"branch_id", "branch_name", "status", "reason_codes", "price", "currency",
		"price_normalized", "duration_minutes", "connection_minutes", "baggage_kg",
		"visa_risk", "self_transfer", "score", "provider",
	}
	if err := w.Write(headers); err != nil {
		return err
	}

	for _, br := range result.Branches {
		row := branchCSVRow(br)
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func branchCSVRow(br model.BranchResult) []string {
	codes := codesStr(br.ReasonCodes)
	price, cur, norm := "", "", ""
	dur, conn, bag := "", "", ""
	visa, self, score, prov := "", "", "", strings.Join(br.Providers, "+")

	if br.Offer != nil {
		price = fmt.Sprintf("%d", br.Offer.Price.Amount)
		cur = br.Offer.Price.Currency
		if br.Offer.PriceNormalized != nil {
			norm = fmt.Sprintf("%d", br.Offer.PriceNormalized.Amount)
		}
		dur = fmt.Sprintf("%d", int(br.Offer.TotalDuration.Minutes()))
		conn = fmt.Sprintf("%d", int(br.Offer.ConnectionDuration.Minutes()))
		if br.Offer.CheckedBaggageKg != nil {
			bag = fmt.Sprintf("%d", *br.Offer.CheckedBaggageKg)
		}
		visa = string(br.Offer.VisaRisk)
		self = fmt.Sprintf("%t", br.Offer.SelfTransfer)
	}
	if br.Score != nil {
		score = fmt.Sprintf("%.2f", *br.Score)
	}
	return []string{
		br.BranchID, br.BranchName, string(br.Status), codes,
		price, cur, norm, dur, conn, bag, visa, self, score, prov,
	}
}

func codesStr(codes []model.ReasonCode) string {
	parts := make([]string, len(codes))
	for i, c := range codes {
		parts[i] = string(c)
	}
	return strings.Join(parts, ",")
}
