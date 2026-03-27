package indicator

import "time"

// Price 是所有指標函式的統一輸入型別。
type Price struct {
	Date   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int64
}

// DataPoint 是單一時間序列值（MA、EMA、RSI、Volume 使用）。
type DataPoint struct {
	Date  time.Time
	Value float64
}

// MACDResult 持有三條 MACD 序列。
type MACDResult struct {
	DIF       []DataPoint
	Signal    []DataPoint
	Histogram []DataPoint
}

// KDResult 持有 K 和 D 序列。
type KDResult struct {
	K []DataPoint
	D []DataPoint
}

// BBResult 持有布林通道序列。
type BBResult struct {
	Upper []DataPoint
	Mid   []DataPoint
	Lower []DataPoint
}
