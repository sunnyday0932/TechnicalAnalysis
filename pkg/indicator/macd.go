package indicator

// MACD 使用 EMA(12)、EMA(26)、Signal EMA(9) 計算 MACD 指標。
// 三條結果序列長度均等於 prices 長度。
func MACD(prices []Price) MACDResult {
	ema12 := EMA(prices, 12)
	ema26 := EMA(prices, 26)

	dif := make([]DataPoint, len(prices))
	for i := range prices {
		dif[i] = DataPoint{Date: prices[i].Date, Value: ema12[i].Value - ema26[i].Value}
	}

	difPrices := make([]Price, len(dif))
	for i, d := range dif {
		difPrices[i] = Price{Date: d.Date, Close: d.Value}
	}
	signal := EMA(difPrices, 9)

	histogram := make([]DataPoint, len(dif))
	for i := range dif {
		histogram[i] = DataPoint{Date: dif[i].Date, Value: dif[i].Value - signal[i].Value}
	}
	return MACDResult{DIF: dif, Signal: signal, Histogram: histogram}
}
