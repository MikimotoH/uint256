package promo_engine_v2

import "github.com/shopspring/decimal"

type QuoteLineKind int

const (
	QuoteLineKindSales        QuoteLineKind = 1
	QuoteLineKindReturn       QuoteLineKind = 2
	QuoteLineKindPickup       QuoteLineKind = 3
	QuoteLineKindReturnPickup QuoteLineKind = 4
)

type QuotePriceSource int

const (
	QuotePriceSourceAutoPromo    QuotePriceSource = 1
	QuotePriceSourceManualPromo  QuotePriceSource = 2
	QuotePriceSourceManualAdjust QuotePriceSource = 3
)

type QuoteDetail struct {
	Price     decimal.Decimal
	Quantity  uint64
	CashierID *uint64
	Remark    string

	ItemID           *uint64
	PosItemDraftID   *uint64
	IncomeExpensesID *uint64
	QuoteLineKind

	// 價格來源
	QuotePriceSource
	CamapignDetailID         *uint64
	ManualGift               bool
	ChangedPrice             *decimal.Decimal
	ManualDiscount           *decimal.Decimal
	ManualDiscountPercentage string

	LotNo string
	TaxNo int
}
