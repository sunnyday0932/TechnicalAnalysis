package indicator

import "math"

// BollingerBands 計算布林通道：中軌 ± 2 個標準差。
// 若 len(prices) < period 回傳空 BBResult。
func BollingerBands(prices []Price, period int) BBResult {
	mas := MA(prices, period)
	if mas == nil {
		return BBResult{}
	}
	upper := make([]DataPoint, len(mas))
	lower := make([]DataPoint, len(mas))
	for i, ma := range mas {
		priceIdx := i + period - 1
		sum := 0.0
		for j := priceIdx - period + 1; j <= priceIdx; j++ {
			diff := prices[j].Close - ma.Value
			sum += diff * diff
		}
		std := math.Sqrt(sum / float64(period))
		upper[i] = DataPoint{Date: ma.Date, Value: ma.Value + 2*std}
		lower[i] = DataPoint{Date: ma.Date, Value: ma.Value - 2*std}
	}
	return BBResult{Upper: upper, Mid: mas, Lower: lower}
}
