package indicator

// KD 計算台灣式隨機指標（KD），初始 K=D=50，使用 1/3 平滑。
// 若 len(prices) < period 回傳空 KDResult。
func KD(prices []Price, period int) KDResult {
	if len(prices) < period {
		return KDResult{}
	}
	k, d := 50.0, 50.0
	kValues := make([]DataPoint, 0, len(prices)-period+1)
	dValues := make([]DataPoint, 0, len(prices)-period+1)

	for i := period - 1; i < len(prices); i++ {
		highest := prices[i-period+1].High
		lowest := prices[i-period+1].Low
		for j := i - period + 2; j <= i; j++ {
			if prices[j].High > highest {
				highest = prices[j].High
			}
			if prices[j].Low < lowest {
				lowest = prices[j].Low
			}
		}
		rsv := 50.0
		if highest != lowest {
			rsv = (prices[i].Close - lowest) / (highest - lowest) * 100
		}
		k = k*2/3 + rsv*1/3
		d = d*2/3 + k*1/3
		kValues = append(kValues, DataPoint{Date: prices[i].Date, Value: k})
		dValues = append(dValues, DataPoint{Date: prices[i].Date, Value: d})
	}
	return KDResult{K: kValues, D: dValues}
}
