package indicator

// MA 回傳簡單移動平均資料點。若 len(prices) < period 回傳 nil。
func MA(prices []Price, period int) []DataPoint {
	if len(prices) < period {
		return nil
	}
	result := make([]DataPoint, 0, len(prices)-period+1)
	for i := period - 1; i < len(prices); i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += prices[j].Close
		}
		result = append(result, DataPoint{Date: prices[i].Date, Value: sum / float64(period)})
	}
	return result
}

// EMA 回傳指數移動平均資料點，以第一個價格作為種子值。
// 結果長度等於 prices 長度。空輸入回傳 nil。
func EMA(prices []Price, period int) []DataPoint {
	if len(prices) == 0 {
		return nil
	}
	k := 2.0 / float64(period+1)
	result := make([]DataPoint, len(prices))
	result[0] = DataPoint{Date: prices[0].Date, Value: prices[0].Close}
	for i := 1; i < len(prices); i++ {
		result[i] = DataPoint{
			Date:  prices[i].Date,
			Value: prices[i].Close*k + result[i-1].Value*(1-k),
		}
	}
	return result
}
