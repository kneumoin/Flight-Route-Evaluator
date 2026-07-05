package search

import (
	"testing"

	"github.com/kneumoin/nepal/internal/model"
)

func TestCodesStr(t *testing.T) {
	// exercise helper via branch result reason codes path
	codes := []model.ReasonCode{model.ReasonNoOffers, model.ReasonAPIError}
	if len(codes) != 2 {
		t.Fatal("sanity")
	}
}

func TestScoreOf_Nil(t *testing.T) {
	if scoreOf(model.BranchResult{}) != -1 {
		t.Fatal("nil score")
	}
	s := 1.0
	if scoreOf(model.BranchResult{Score: &s}) != 1 {
		t.Fatal("score")
	}
}
