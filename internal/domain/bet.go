package domain

import (
	"context"
	"math"

	"github.com/FatwaArya/pm-ingest/internal"
)

type PredictionResult struct {
	BrierScore         float64 // 0 is perfect, 1 is total error
	Calibration        float64 // Accuracy of your probability estimates (0-100%)
	WinRate            float64 // Historical win rate
	ConfidenceInterval float64 // The +/- margin of error for the next bet
	SampleSize         int     // Total number of trades analyzed
	AvgRealizedPnl     float64 // Average realized profit/loss
	TotalRealizedPnl   float64 // Total realized profit/loss
}

// CalculateConfidence calculates user confidence metrics based on closed positions
func CalculateConfidence(closedPositions []internal.ClosedPosition) PredictionResult {
	if len(closedPositions) == 0 {
		return PredictionResult{
			BrierScore:         0.0,
			Calibration:        0.0,
			WinRate:            0.0,
			ConfidenceInterval: 0.0,
			SampleSize:         0,
			AvgRealizedPnl:     0.0,
			TotalRealizedPnl:   0.0,
		}
	}

	sampleSize := len(closedPositions)
	var wins, totalPnl, brierSum float64
	var pnlValues []float64

	// Group positions by price buckets for calibration
	priceBuckets := make(map[int][]bool) // bucket -> []bool (true = win, false = loss)

	for _, pos := range closedPositions {
		// Determine if this position was a win (positive realized PnL)
		isWin := pos.RealizedPnl > 0
		if isWin {
			wins++
		}

		// Accumulate PnL
		totalPnl += pos.RealizedPnl
		pnlValues = append(pnlValues, pos.RealizedPnl)

		// Calculate Brier score
		// avgPrice represents the user's probability estimate (0-1)
		// actual outcome: 1 if profitable, 0 if not
		actualOutcome := 0.0
		if isWin {
			actualOutcome = 1.0
		}
		predictedProb := pos.AvgPrice
		brierSum += math.Pow(predictedProb-actualOutcome, 2)

		// Group by price bucket for calibration (10 buckets: 0-0.1, 0.1-0.2, ..., 0.9-1.0)
		bucket := int(math.Floor(predictedProb * 10))
		if bucket >= 10 {
			bucket = 9
		}
		if priceBuckets[bucket] == nil {
			priceBuckets[bucket] = make([]bool, 0)
		}
		priceBuckets[bucket] = append(priceBuckets[bucket], isWin)
	}

	// Calculate metrics
	winRate := wins / float64(sampleSize)
	avgPnl := totalPnl / float64(sampleSize)
	brierScore := brierSum / float64(sampleSize)

	// Calculate calibration: how well predicted probabilities match actual win rates
	// For each price bucket, compare predicted probability with actual win rate
	var calibrationSum float64
	var calibrationCount int
	for bucket, outcomes := range priceBuckets {
		if len(outcomes) < 3 { // Skip buckets with too few samples
			continue
		}
		predictedProb := (float64(bucket) + 0.5) / 10.0 // Midpoint of bucket
		actualWinRate := 0.0
		for _, isWin := range outcomes {
			if isWin {
				actualWinRate++
			}
		}
		actualWinRate /= float64(len(outcomes))
		// Calibration error: difference between predicted and actual
		calibrationSum += math.Abs(predictedProb - actualWinRate)
		calibrationCount++
	}

	calibration := 0.0
	if calibrationCount > 0 {
		avgCalibrationError := calibrationSum / float64(calibrationCount)
		// Convert to percentage (0-100%), where 0% error = 100% calibration
		calibration = (1.0 - avgCalibrationError) * 100.0
		if calibration < 0 {
			calibration = 0
		}
	}

	// Calculate confidence interval using standard deviation of PnL
	confidenceInterval := 0.0
	if len(pnlValues) > 1 {
		// Calculate standard deviation
		var variance float64
		for _, pnl := range pnlValues {
			variance += math.Pow(pnl-avgPnl, 2)
		}
		variance /= float64(len(pnlValues) - 1)
		stdDev := math.Sqrt(variance)

		// 95% confidence interval (approximately 1.96 standard deviations)
		// Normalized by sample size (larger sample = tighter interval)
		confidenceInterval = (1.96 * stdDev) / math.Sqrt(float64(sampleSize))
	}

	return PredictionResult{
		BrierScore:         brierScore,
		Calibration:        calibration,
		WinRate:            winRate * 100.0, // Convert to percentage
		ConfidenceInterval: confidenceInterval,
		SampleSize:         sampleSize,
		AvgRealizedPnl:     avgPnl,
		TotalRealizedPnl:   totalPnl,
	}
}

// CalculateConfidenceForUser calculates confidence for a specific user address
// This is a helper that combines fetching closed positions and calculating confidence
func CalculateConfidenceForUser(ctx context.Context, apiClient *internal.PolymarketAPIClient, userAddress string, limit int) (PredictionResult, error) {
	if limit <= 0 {
		limit = 1000 // Default to max allowed
	}

	params := internal.ClosedPositionsQueryParams{
		User:          userAddress,
		Limit:         limit,
		SortBy:        "REALIZEDPNL",
		SortDirection: "DESC",
	}

	closedPositions, err := apiClient.GetClosedPositions(ctx, params)
	if err != nil {
		return PredictionResult{}, err
	}

	return CalculateConfidence(closedPositions), nil
}
