package jobs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/riverqueue/river"

	"github.com/use-golang/api/db"
	"github.com/use-golang/api/graph/model"
	"github.com/use-golang/api/llm"
)

type EstimateCaloriesArgs struct {
	QueryID string `json:"query_id"`
}

func (EstimateCaloriesArgs) Kind() string { return "estimate_calories" }

func (EstimateCaloriesArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:    "ai",
		Priority: 1, // highest priority
	}
}

type EstimateCaloriesWorker struct {
	river.WorkerDefaults[EstimateCaloriesArgs]

	DB         *db.DB
	CallLLM    func(ctx context.Context, model string, prompt string, reasoningEffort string) (*llm.Result, error)
	OnComplete func(result *model.CalorieQuery)
}

func (w *EstimateCaloriesWorker) Work(ctx context.Context, job *river.Job[EstimateCaloriesArgs]) error {
	q, err := w.DB.GetCalorieQuery(ctx, job.Args.QueryID)
	if err != nil {
		return fmt.Errorf("failed to fetch calorie query %s: %w", job.Args.QueryID, err)
	}

	prompt := fmt.Sprintf(
		"Tell me how many calories are in this meal: %s. Respond with just a single number, nothing else.",
		q.MealText,
	)

	var reasoningEffort string
	if q.ReasoningEffort != nil {
		reasoningEffort = *q.ReasoningEffort
	}

	start := time.Now()
	result, err := w.CallLLM(ctx, q.Model, prompt, reasoningEffort)
	elapsed := int(time.Since(start).Milliseconds())

	if err != nil {
		status := "failed"
		errMsg := err.Error()
		_ = w.DB.UpdateCalorieQuery(ctx, q.ID, nil, &elapsed, nil, status, &errMsg)
		q.Status = status
		q.ResponseTimeMs = &elapsed
		q.ErrorMessage = &errMsg
		if w.OnComplete != nil {
			w.OnComplete(q)
		}
		return nil // don't retry on LLM errors
	}

	calories := parseCalories(result.Text)
	totalTokens := result.TotalTokens
	status := "completed"
	if err := w.DB.UpdateCalorieQuery(ctx, q.ID, &calories, &elapsed, &totalTokens, status, nil); err != nil {
		return fmt.Errorf("failed to update calorie query %s: %w", q.ID, err)
	}

	q.Calories = &calories
	q.ResponseTimeMs = &elapsed
	q.TotalTokens = &totalTokens
	q.Status = status

	if w.OnComplete != nil {
		w.OnComplete(q)
	}

	return nil
}

func parseCalories(response string) int {
	response = strings.TrimSpace(response)
	var num int
	fmt.Sscanf(response, "%d", &num)
	return num
}
