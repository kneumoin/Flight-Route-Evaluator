package scoring

import (
	"testing"
	"time"

	"github.com/kneumoin/nepal/internal/config"
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
	sST, _ := ScoreOffer(offer, stBranch, cfg, testConstraints(), VisaLow, model.OperationalDisruptionLow, testOperationalRisk())
	sNormal, _ := ScoreOffer(offer, normalBranch, cfg, testConstraints(), VisaLow, model.OperationalDisruptionLow, testOperationalRisk())
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
	sCheap, _ := ScoreOffer(cheap, br, cfg, testConstraints(), VisaLow, model.OperationalDisruptionLow, testOperationalRisk())
	sExp, _ := ScoreOffer(expensive, br, cfg, testConstraints(), VisaLow, model.OperationalDisruptionLow, testOperationalRisk())
	if sCheap <= sExp {
		t.Fatalf("cheaper should score higher: %v vs %v", sCheap, sExp)
	}
}

func TestScore_VisaRequiresVisaPenalized(t *testing.T) {
	if HubVisaCategory("DEL") != VisaRequiresVisa {
		t.Fatal("DEL should be REQUIRES_VISA")
	}
	cfg := testScoringConfig()
	br := testBranch("single_ticket")
	offer := baseOffer()
	sLow, _ := ScoreOffer(offer, br, cfg, testConstraints(), VisaRequiresVisa, model.OperationalDisruptionLow, testOperationalRisk())
	sHigh, _ := ScoreOffer(offer, br, cfg, testConstraints(), VisaLow, model.OperationalDisruptionLow, testOperationalRisk())
	if sLow >= sHigh {
		t.Fatalf("visa required hub should score lower: %v vs %v", sLow, sHigh)
	}
}

func TestScore_PenalizesMissingBaggage(t *testing.T) {
	cfg := testScoringConfig()
	br := testBranch("single_ticket")
	offer := baseOffer()
	zero := 0
	offer.CheckedBaggageKg = &zero
	sLow, _ := ScoreOffer(offer, br, cfg, testConstraints(), VisaLow, model.OperationalDisruptionLow, testOperationalRisk())
	offer.CheckedBaggageKg = intPtr(30)
	sHigh, _ := ScoreOffer(offer, br, cfg, testConstraints(), VisaLow, model.OperationalDisruptionLow, testOperationalRisk())
	if sLow >= sHigh {
		t.Fatalf("missing baggage penalty expected")
	}
}

func TestScore_PenalizesUnknownBaggage(t *testing.T) {
	rejected, codes := CheckBaggage(nil, model.DataQualityRealtime, testConstraints())
	if !rejected || codes[0] != model.ReasonBaggageUnknown {
		t.Fatalf("expected BAGGAGE_UNKNOWN reject")
	}
}

func TestCheckBaggage_CachedAllowsUnknown(t *testing.T) {
	c := testConstraints()
	c.Baggage.AllowUnknownForCachedProvider = true
	rejected, codes := CheckBaggage(nil, model.DataQualityCached, c)
	if rejected || len(codes) != 0 {
		t.Fatalf("cached unknown baggage should not reject when allowed: reject=%v codes=%v", rejected, codes)
	}
}

func TestCheckBaggage_BrowserAllowsUnknown(t *testing.T) {
	c := testConstraints()
	c.Baggage.AllowUnknownForCachedProvider = true
	rejected, codes := CheckBaggage(nil, model.DataQualityBrowserCollected, c)
	if rejected || len(codes) != 0 {
		t.Fatalf("browser collected unknown baggage should not reject when allowed")
	}
}

func TestCheckBaggage_CachedRejectsWhenDisabled(t *testing.T) {
	c := testConstraints()
	c.Baggage.AllowUnknownForCachedProvider = false
	rejected, codes := CheckBaggage(nil, model.DataQualityCached, c)
	if !rejected || codes[0] != model.ReasonBaggageUnknown {
		t.Fatalf("expected reject when allow_unknown_for_cached_provider is false")
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
	sEarly, _ := ScoreOffer(early, br, cfg, testConstraints(), VisaLow, model.OperationalDisruptionLow, testOperationalRisk())
	sLate, _ := ScoreOffer(late, br, cfg, testConstraints(), VisaLow, model.OperationalDisruptionLow, testOperationalRisk())
	if sEarly <= sLate {
		t.Fatalf("late arrival should score lower")
	}
}

func TestScore_OperationalDisruptionPenalty(t *testing.T) {
	cfg := testScoringConfig()
	riskCfg := config.OperationalDisruptionConfig{
		Enabled: true, DefaultLevel: "LOW",
		Hubs:      map[string]string{"DOH": "HIGH"},
		Penalties: map[string]float64{"HIGH": 12, "LOW": 0},
	}
	br := testBranch("single_ticket")
	offer := baseOffer()
	sLow, _ := ScoreOffer(offer, br, cfg, testConstraints(), VisaLow, model.OperationalDisruptionLow, riskCfg)
	sHigh, bd := ScoreOffer(offer, br, cfg, testConstraints(), VisaLow, model.OperationalDisruptionHigh, riskCfg)
	if sHigh >= sLow {
		t.Fatalf("HIGH disruption should lower score: %v vs %v", sHigh, sLow)
	}
	if bd.RegionalDisruptionPenalty != 12 {
		t.Fatalf("penalty=%v", bd.RegionalDisruptionPenalty)
	}
}

func TestHubOperationalDisruption_ISTLow(t *testing.T) {
	cfg := config.OperationalDisruptionConfig{
		Enabled: true, DefaultLevel: "LOW",
		Hubs: map[string]string{"DOH": "HIGH"},
	}
	if HubOperationalDisruptionRisk("IST", cfg) != model.OperationalDisruptionLow {
		t.Fatal("IST should default LOW")
	}
}

func TestHubOperationalDisruption_SaudiLow(t *testing.T) {
	cfg := config.OperationalDisruptionConfig{
		Enabled: true, DefaultLevel: "LOW",
		Hubs: map[string]string{"KWI": "HIGH"},
	}
	if HubOperationalDisruptionRisk("DMM", cfg) != model.OperationalDisruptionLow {
		t.Fatal("DMM should default LOW")
	}
	if HubOperationalDisruptionRisk("RUH", cfg) != model.OperationalDisruptionLow {
		t.Fatal("RUH should default LOW")
	}
}

func TestHubOperationalDisruption_GulfHigh(t *testing.T) {
	cfg := config.OperationalDisruptionConfig{Enabled: true, Hubs: map[string]string{"DOH": "HIGH"}}
	if HubOperationalDisruptionRisk("DOH", cfg) != model.OperationalDisruptionHigh {
		t.Fatal("DOH should be HIGH")
	}
}

func intPtr(v int) *int { return &v }

func mustLocal(y, m, d, hh, mm int, loc *time.Location) time.Time {
	return time.Date(y, time.Month(m), d, hh, mm, 0, 0, loc)
}
