package promo_engine_v2

import (
	cons "github.com/mikimotoh/uint256/constants"
	"github.com/shopspring/decimal"
)

type Quote struct { // 結帳報價 inputmodel，包含前端傳入的資料以及促銷引擎計算後的結果
	// 以下兩者擇一，或都不選(沒有會員卡的一般消費者)
	EmployeeID   *uint `json:"employee_id"`   // 員工id 員工變成消費者，使用員工價
	MembershipID *uint `json:"membership_id"` // 會員ID
	CashierID    uint  `json:"cashier_id"`    // 收銀員id  必須在employees 存在的

	MachineID           uint                      `json:"machine_id"`                             // 台號
	Remark              string                    `json:"remark"`                                 // 備註
	SalesDetails        []QuoteSalesDetail        `json:"sales_details" validate:"required,dive"` // 結帳明細
	ReturnDetails       []QuoteReturnDetail       `json:"return_details" validate:"dive"`         // 退貨明細，沒有退貨時可以不傳或傳空陣列
	PickupDetails       []QuotePickupDetail       `json:"pickup_details" validate:"dive"`         // 提貨明細，沒有提貨時可以不傳或傳空陣列
	ReturnPickupDetails []QuoteReturnPickupDetail `json:"return_pickup_details" validate:"dive"`  // 退提貨明細，沒有退提貨時可以不傳或傳空陣列
}

type QuoteSalesDetail struct { // 結帳報價明細
	// 以下兩者擇一必填
	ItemID         *uint           `json:"item_id"`                      // 商品ID
	PosItemDraftID *uint           `json:"pos_item_draft_id"`            // 商品組合ID ID。若有此時，item_id會被忽略
	Remark         string          `json:"remark"`                       // 明細備註(C60)
	CashierID      *uint           `json:"cashier_id"`                   // 收銀員id  必須在employees 存在的
	Quantity       decimal.Decimal `json:"quantity" validate:"required"` // 數量
	// 促銷引擎規則相關欄位
	CampaignDetailID *uint                 `json:"campaign_detail_id"` // 前端指定促銷活動明細ID。後端必須驗證促銷活動是否存在且符合資格
	PointsRedeemed   decimal.Decimal       `json:"points_redeemed"`    // 前端指定兌換點數
	QuotePriceSource cons.QuotePriceSource `json:"quote_price_source"` // 價格來源:1自動促銷,2手動促銷,3手動調整
	// 銷售 才用

	// 收銀員手動調整欄位，不能同時與 CampaignDetailID 並存
	IsManualGift        bool             `json:"is_manual_gift"`        // 前端指定是否手動贈品
	ChangedUnitPrice    *decimal.Decimal `json:"changed_unit_price"`    // 前端手動變價過後的單價
	ManualDiscount      *decimal.Decimal `json:"manual_discount"`       // 前端手動折讓，正數價格變便宜
	ManualDiscountRatio *decimal.Decimal `json:"manual_discount_ratio"` // 前端手動折扣 百分比。 80%代表價格打八折

	IncomeExpensesReasonID *uint `json:"income_expenses_reason_id"` // 收支原因ID，當交易型別為收支原因時使用
	// 銷售會用，退貨會用，提貨和退提貨不會用

	TaxType uint8 `json:"tax_type"` // 稅別(1.應稅內含 2.應稅外加 3.零稅率 4.免稅)
	// 銷售 退貨會用，提貨和退提貨不會用

	LotNo string `json:"lot_no"` // 批號，前端會從庫存抓
	// 銷售 退貨 提貨和退提貨 都會用
}

type QuoteReturnDetail = QuoteSalesDetail // 退貨明細與銷售明細結構相同，可以共用

type QuotePickupDetail struct {
	ItemID            uint            `json:"item_id"`                      // 商品ID
	Remark            string          `json:"remark"`                       // 明細備註(C60)
	CashierID         *uint           `json:"cashier_id"`                   // 收銀員id  必須在employees 存在的
	Quantity          decimal.Decimal `json:"quantity" validate:"required"` // 數量
	MemberWarehouseID *uint           `json:"member_warehouse_id"`          // 寄庫提貨指定倉庫ID，當QuoteLineKind為提貨或退提貨時可填

	LotNo string `json:"lot_no"` // 批號，前端會從庫存抓
}

type QuoteReturnPickupDetail = QuotePickupDetail // 退提貨明細與提貨明細結構相同，可以共用

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
