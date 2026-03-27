# TechnicalAnalysis

台股技術分析 REST API，以 Go 開發，提供上市上櫃股票的歷史價格查詢與技術指標計算功能，並搭配前端網頁進行視覺化呈現。

---

## 專案簡介

本專案串接台灣證券交易所（TWSE）與櫃買中心（TPEx）的免費公開 API，定時抓取台股日K資料並儲存於 PostgreSQL，再透過 REST API 提供技術指標計算結果給前端應用使用。

---

## 功能

### 股票資料
- 查詢所有追蹤的上市、上櫃股票清單
- 查詢單一股票基本資料（公司名稱、所屬市場）
- 查詢指定股票的日K歷史價格（開盤、最高、最低、收盤、成交量）

### 技術指標
支援以下六種常用技術指標，可指定計算週期：

| 指標 | 說明 |
|------|------|
| MA（移動平均線） | 簡單移動平均，平滑價格趨勢 |
| EMA（指數移動平均） | 對近期價格給予較高權重的移動平均 |
| RSI（相對強弱指數） | 衡量超買超賣，數值 0–100 |
| MACD | 趨勢動能指標，包含 DIF、Signal、Histogram |
| KD（隨機指標） | 判斷短期超買超賣與交叉訊號 |
| Bollinger Bands（布林通道） | 中軌 ± 2 個標準差，判斷價格波動區間 |

### 資料同步
- 每日收盤後（週一至週五 18:30）自動從 TWSE / TPEx 抓取最新資料
- 提供手動觸發同步的 API endpoint
- 同步失敗時自動於 5 分鐘後重試一次
- 可查詢最後同步時間與狀態

---

## 技術架構

- **語言**：Go
- **HTTP Framework**：Gin
- **資料庫**：PostgreSQL（Docker）
- **DB 查詢**：sqlc（type-safe SQL）
- **Schema 管理**：golang-migrate
- **排程**：robfig/cron
- **資料來源**：TWSE Open API、TPEx Open API（免費，無需 token）

---

## 開發環境啟動

```bash
# 啟動 PostgreSQL
docker-compose up -d

# 執行 schema migration
migrate -path ./migrations -database $DATABASE_URL up

# 啟動 API server
go run ./cmd/api
```

---

## API 概覽

```
GET  /api/v1/stocks                          查詢所有股票
GET  /api/v1/stocks/:symbol                  查詢單一股票資料
GET  /api/v1/stocks/:symbol/prices           查詢日K歷史價格
GET  /api/v1/stocks/:symbol/indicators       查詢技術指標
POST /api/v1/sync                            手動觸發全量同步
POST /api/v1/sync/:symbol                    手動觸發單一股票同步
GET  /api/v1/sync/status                     查詢同步狀態
```
