package indicator

// RSI 使用 Wilder 平滑法計算相對強弱指數。
// 若 len(prices) <= period 回傳 nil。
// 結果長度 = len(prices) - period。
func RSI(prices []Price, period int) []DataPoint {
	if len(prices) <= period {
		return nil
	}
	var gainSum, lossSum float64
	for i := 1; i <= period; i++ {
		change := prices[i].Close - prices[i-1].Close
		if change > 0 {
			gainSum += change
		} else {
			lossSum -= change
		}
	}
	avgGain := gainSum / float64(period)
	avgLoss := lossSum / float64(period)

	result := make([]DataPoint, 0, len(prices)-period)
	result = append(result, DataPoint{Date: prices[period].Date, Value: rsiValue(avgGain, avgLoss)})

	for i := period + 1; i < len(prices); i++ {
		change := prices[i].Close - prices[i-1].Close
		gain, loss := 0.0, 0.0
		if change > 0 {
			gain = change
		} else {
			loss = -change
		}
		avgGain = (avgGain*float64(period-1) + gain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)
		result = append(result, DataPoint{Date: prices[i].Date, Value: rsiValue(avgGain, avgLoss)})
	}
	return result
}

func rsiValue(avgGain, avgLoss float64) float64 {
	if avgLoss == 0 {
		if avgGain == 0 {
			return 50.0
		}
		return 100.0
	}
	return 100 - 100/(1+avgGain/avgLoss)
}
