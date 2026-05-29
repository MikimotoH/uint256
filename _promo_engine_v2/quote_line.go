package promo_engine_v2

import "github.com/shopspring/decimal"

type QuoteLine struct {
	Price               decimal.Decimal
	Quantity            uint64
	ItemID              uint64
	LargeCategoryTypeID uint64

	LargeCategoryID     uint // 大分類ID con.CategoryTypeLarge
	MediumCategoryID    uint // 中分類ID con.CategoryTypeMedium
	SmallCategoryID     uint // 小分類ID con.CategoryTypeSmall
	ColorLabelID        uint // 顏色標籤ID con.CategoryTypeColorLabel
	PromotionCategoryID uint // 促銷類別ID con.CategoryTypePromotion

}

// StoreItem 從庫存服務獲取的門市商品資訊
type StoreItem struct {
	ItemID uint // con.CategoryTypeItemID

	NormalPrice   decimal.Decimal // 原價，非會員使用此
	VipPrice      decimal.NullDecimal
	WholePrice    decimal.NullDecimal
	SalePrice     decimal.NullDecimal
	EmployeePrice decimal.NullDecimal // 員工價

	Price6  decimal.NullDecimal
	Price7  decimal.NullDecimal
	Price8  decimal.NullDecimal
	Price9  decimal.NullDecimal
	Price10 decimal.NullDecimal
	Price11 decimal.NullDecimal
	Price12 decimal.NullDecimal

	AllowPriceChanged bool // 是否允許變價
	AllowSale         bool // 是否允許銷售
	AllowPromotion    bool // 是否允許促銷
	AllowPosExchange  bool // 是否允許POS退貨換貨

	// category info
	LargeCategoryID     uint // 大分類ID con.CategoryTypeLarge
	MediumCategoryID    uint // 中分類ID con.CategoryTypeMedium
	SmallCategoryID     uint // 小分類ID con.CategoryTypeSmall
	ColorLabelID        uint // 顏色標籤ID con.CategoryTypeColorLabel
	PromotionCategoryID uint // 促銷類別ID con.CategoryTypePromotion
}
