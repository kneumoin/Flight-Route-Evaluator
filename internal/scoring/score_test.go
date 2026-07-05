package scoring

import (
	"testing"
	"time"

	"github.com/kneumoin/nepal/internal/model"
)

func TestNormalizeMoney_USD(t *testing.T) {
	m := model.Money{Amount: 85000, Currency: "USD"}
	n, err := NormalizeMoney(m, "USD")
	if err != nil || n.Amount != 85000 {
		t.Fatalf("got %+v err=%v", n, err)
	}
}

func TestNormalizeMoney_RUB(t *testing.T) {
	m := model.Money{Amount: 7200000, Currency: "RUB"}
	n, err := NormalizeMoney(m, "USD")
	if err != nil {
		t.Fatal(err)
	}
	if n.Amount <= 0 {
		t.Fatalf("expected positive USD amount, got %d", n.Amount)
	}
}

func TestNormalizeMoney_Unconvertible(t *testing.T) {
	m := model.Money{Amount: 100, Currency: "XYZ"}
	_, err := NormalizeMoney(m, "USD")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestScore_PenalizesSelfTransfer(t *testing.T) {
	cfg := testScoringConfig()
	offer := baseOffer()
	stBranch := testBranch("self_transfer")
	normalBranch := testBranch("single_ticket")
	sST, _ := ScoreOffer(offer, stBranch, cfg, testConstraints())
	sNormal, _ := ScoreOffer(offer, normalBranch, cfg, testConstraints())
	if sST >= sNormal {
		t.Fatalf("self-transfer branch should score lower: %v vs %v", sST, sNormal)
	}
}

func TestScore_CheaperRouteScoresBetter(t *testing.T) {
	cfg := testScoringConfig()
	br := testBranch("single_ticket")
	cheap := baseOffer()
	cheap.Price = model.Money{Amount: 50000, Currency: "USD"}
	cheap.PriceNormalized = &model.Money{Amount: 50000, Currency: "USD"}
	expensive := baseOffer()
	expensive.Price = model.Money{Amount: 90000, Currency: "USD"}
	expensive.PriceNormalized = &model.Money{Amount: 90000, Currency: "USD"}
	sCheap, _ := ScoreOffer(cheap, br, cfg, testConstraints())
	sExp, _ := ScoreOffer(expensive, br, cfg, testConstraints())
	if sCheap <= sExp {
		t.Fatalf("cheaper should score higher: %v vs %v", sCheap, sExp)
	}
}

func TestScore_VisaRejectRemovesBranch(t *testing.T) {
	if !RequiresTransitVisa("DEL") {
		t.Fatal("DEL should require visa for stub")
	}
}

func TestScore_PenalizesMissingBaggage(t *testing.T) {
	cfg := testScoringConfig()
	br := testBranch("single_ticket")
	offer := baseOffer()
	zero := 0
	offer.CheckedBaggageKg = &zero
	sLow, _ := ScoreOffer(offer, br, cfg, testConstraints())
	offer.CheckedBaggageKg = intPtr(30)
	sHigh, _ := ScoreOffer(offer, br, cfg, testConstraints())
	if sLow >= sHigh {
		t.Fatalf("missing baggage penalty expected")
	}
}

func TestScore_PenalizesUnknownBaggage(t *testing.T) {
	rejected, codes := CheckBaggage(nil, testConstraints())
	if !rejected || codes[0] != model.ReasonBaggageUnknown {
		t.Fatalf("expected BAGGAGE_UNKNOWN reject")
	}
}

func TestScore_CurrencyUnconvertible(t *testing.T) {
	o := baseOffer()
	o.Price = model.Money{Amount: 100, Currency: "XYZ"}
	err := NormalizeOfferPrice(&o, "USD")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestScore_LateArrival(t *testing.T) {
	cfg := testScoringConfig()
	br := testBranch("single_ticket")
	early := baseOffer()
	late := baseOffer()
	ktm, _ := AirportLocation("KTM")
	early.Segments[len(early.Segments)-1].Arrival = mustLocal(2026, 9, 29, 16, 0, ktm)
	late.Segments[len(late.Segments)-1].Arrival = mustLocal(2026, 9, 29, 20, 0, ktm)
	sEarly, _ := ScoreOffer(early, br, cfg, testConstraints())
	sLate, _ := ScoreOffer(late, br, cfg, testConstraints())
	if sEarly <= sLate {
		t.Fatalf("late arrival should score lower")
	}
}

func intPtr(v int) *int { return &v }

func mustLocal(y, m, d, hh, mm int, loc *time.Location) time.Time {
	return time.Date(y, time.Month(m), d, hh, mm, 0, 0, loc)
}
