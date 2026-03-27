package indicator

// Volume 將價格資料的 Volume 欄位轉換為 DataPoint 切片。
func Volume(prices []Price) []DataPoint {
	result := make([]DataPoint, len(prices))
	for i, p := range prices {
		result[i] = DataPoint{Date: p.Date, Value: float64(p.Volume)}
	}
	return result
}
