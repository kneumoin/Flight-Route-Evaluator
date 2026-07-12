package search

import (
	"fmt"
	"os"
	"time"

	"github.com/kneumoin/nepal/internal/config"
	"github.com/kneumoin/nepal/internal/model"
)

func (e *Evaluator) progressBranchStart(index, total int, branch config.BranchConfig) {
	if !e.Progress {
		return
	}
	fmt.Fprintf(os.Stderr, "[%d/%d] %s — %s\n", index, total, branch.ID, branch.Name)
}

func (e *Evaluator) progressLegStart(direction string, legIndex, legTotal int, from, to string) {
	if !e.Progress {
		return
	}
	prefix := ""
	if direction != "" {
		prefix = direction + " "
	}
	fmt.Fprintf(os.Stderr, "  %sleg %d/%d %s→%s ...\n", prefix, legIndex, legTotal, from, to)
}

func (e *Evaluator) progressLegDone(offerCount int, elapsed time.Duration) {
	if !e.Progress {
		return
	}
	fmt.Fprintf(os.Stderr, "           %d offer(s) in %s\n", offerCount, elapsed.Round(time.Second))
}

func (e *Evaluator) progressBranchDone(br model.BranchResult, elapsed time.Duration) {
	if !e.Progress {
		return
	}
	msg := fmt.Sprintf("  → %s", br.Status)
	if br.Score != nil {
		msg += fmt.Sprintf(" score=%.1f", *br.Score)
	}
	if len(br.ReasonCodes) > 0 && br.Status != model.StatusOK {
		msg += fmt.Sprintf(" (%s)", br.ReasonCodes[0])
	}
	msg += fmt.Sprintf(" — %s\n", elapsed.Round(time.Second))
	fmt.Fprint(os.Stderr, msg)
}

func (e *Evaluator) PrintProgressSummary(branches []model.BranchResult, outDir string, elapsed time.Duration) {
	if !e.Progress {
		return
	}
	var ok, partial, other int
	for _, br := range branches {
		switch br.Status {
		case model.StatusOK:
			ok++
		case model.StatusPartial:
			partial++
		default:
			other++
		}
	}
	fmt.Fprintf(os.Stderr, "\nFinished in %s: %d ok, %d partial, %d other — wrote %s\n",
		elapsed.Round(time.Second), ok, partial, other, outDir)
}
