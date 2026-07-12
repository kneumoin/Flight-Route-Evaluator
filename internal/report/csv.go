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
		"visa_risk", "visa_category", "operational_disruption_risk", "operational_disruption_penalty", "operational_disruption_notes",
		"self_transfer", "score", "provider", "data_quality", "notes",
		"leg_airlines",
		"price_target", "price_target_currency", "price_minus_1", "price_plus_1",
		"price_window_min", "price_window_min_date", "price_window_days",
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
	if err := w.Error(); err != nil {
		return err
	}
	return WriteCoverageCSV(filepath.Join(filepath.Dir(path), "coverage.csv"), result)
}

func WriteCoverageCSV(path string, result *model.EvaluationResult) error {
	if len(result.RouteCoverage) == 0 {
		return nil
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	headers := []string{
		"branch_id", "branch_name", "leg_index", "from", "to", "target_date",
		"cheapest_date_near_target", "min_price", "currency", "airline", "available_airlines", "transfers",
		"coverage_days", "selected_date", "estimated_date",
	}
	if err := w.Write(headers); err != nil {
		return err
	}
	for _, r := range result.RouteCoverage {
		price, cur := "", ""
		if r.MinPrice != nil {
			price = fmt.Sprintf("%d", r.MinPrice.Amount)
			cur = r.MinPrice.Currency
		}
		transfers := ""
		if r.Transfers != nil {
			transfers = fmt.Sprintf("%d", *r.Transfers)
		}
		if err := w.Write([]string{
			r.BranchID, r.BranchName, fmt.Sprintf("%d", r.LegIndex), r.From, r.To, r.TargetDate,
			r.CheapestDateNearTarget, price, cur, r.Airline, model.FormatAirlineList(r.AvailableAirlines), transfers,
			fmt.Sprintf("%d", r.CoverageDays), r.SelectedDate, fmt.Sprintf("%t", r.EstimatedDate),
		}); err != nil {
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
	dq, notes := "", ""
	visaCat := string(br.VisaCategory)
	regRisk := string(br.OperationalDisruptionRisk)
	regPenalty := ""
	regNotes := br.OperationalDisruptionNotes
	pt, ptc, pm1, pp1, pwmin, pwmind, pwd := "", "", "", "", "", "", ""
	legAirlines := ""
	for _, la := range br.LegAirlines {
		if legAirlines != "" {
			legAirlines += " | "
		}
		legAirlines += fmt.Sprintf("%s-%s:%s", la.From, la.To, model.FormatAirlineList(la.AvailableAirlines))
	}

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
		dq = string(br.Offer.DataQuality)
		notes = strings.Join(br.Offer.Notes, "; ")
	}
	if br.Score != nil {
		score = fmt.Sprintf("%.2f", *br.Score)
	}
	if br.OperationalDisruptionPenalty > 0 {
		regPenalty = fmt.Sprintf("%.0f", br.OperationalDisruptionPenalty)
	}
	if br.PriceComparison != nil {
		pc := br.PriceComparison
		if pc.PriceTarget != nil {
			pt = fmt.Sprintf("%d", pc.PriceTarget.Amount)
			ptc = pc.PriceTargetCurrency
			if ptc == "" {
				ptc = pc.PriceTarget.Currency
			}
		}
		if pc.PriceMinus1 != nil {
			pm1 = fmt.Sprintf("%d", pc.PriceMinus1.Amount)
		}
		if pc.PricePlus1 != nil {
			pp1 = fmt.Sprintf("%d", pc.PricePlus1.Amount)
		}
		if pc.PriceWindowMin != nil {
			pwmin = fmt.Sprintf("%d", pc.PriceWindowMin.Amount)
			pwmind = pc.PriceWindowMinDate
		}
		if pc.PriceWindowDays > 0 {
			pwd = fmt.Sprintf("%d", pc.PriceWindowDays)
		}
	}
	return []string{
		br.BranchID, br.BranchName, string(br.Status), codes,
		price, cur, norm, dur, conn, bag, visa, visaCat, regRisk, regPenalty, regNotes,
		self, score, prov, dq, notes,
		legAirlines,
		pt, ptc, pm1, pp1, pwmin, pwmind, pwd,
	}
}

func codesStr(codes []model.ReasonCode) string {
	parts := make([]string, len(codes))
	for i, c := range codes {
		parts[i] = string(c)
	}
	return strings.Join(parts, ",")
}
