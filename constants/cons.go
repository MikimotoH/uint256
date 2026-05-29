package constants

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
