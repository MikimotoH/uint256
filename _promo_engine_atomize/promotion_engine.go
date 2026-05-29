package promo_engine_atomize

import (
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"

	con "pos-backend/pkg/constants"
	"pos-backend/pkg/contexts"

	"pos-backend/pkg/entities"

	"pos-backend/pkg/services/dtos/grpc_inv"
	"pos-backend/pkg/services/promo_engine_v2"
	"pos-backend/pkg/services/shared"

	"github.com/R167/go-sets"
	"github.com/shopspring/decimal"
)

type (
	StoreItem          = grpc_inv.StoreItem
	PosItemDraft       = grpc_inv.PosItemDraft
	ServiceErrorDetail = shared.ServiceErrorDetail
	MemberInfo         = promo_engine_v2.MemberInfo
	// QuoteDetail        = shared.QuoteDetail
	QuoteLine = promo_engine_v2.QuoteLine
	// QuoteAtom = promo_engine_v2.QuoteAtom
)

var (
	NewMultiSchemaMdlV2                = shared.NewMultiSchemaMdlV2
	ConvertUserInfoToInventoryUserInfo = grpc_inv.ConvertUserInfoToInventoryUserInfo
	NewServiceErrorDetail              = shared.NewServiceErrorDetail
)

const (
	NotExist     = shared.NotExist
	Incorrect    = shared.Incorrect
	GrpcError    = shared.GrpcError
	UnknownError = shared.UnknownError

	PromotionV2SharedSchema = shared.PromotionV2SharedSchema
)

type PromotionEngine struct {
	mu      sync.RWMutex
	StoreID uint // 門市ID (總部不適用pos結帳)

	CategoryTypeLargeRuleMultiMap      RuleMultiMap // 大類別規則
	CategoryTypeMediumRuleMultiMap     RuleMultiMap // 中類別規則
	CategoryTypeSmallRuleMultiMap      RuleMultiMap // 小類別規則
	CategoryTypeColorLabelRuleMultiMap RuleMultiMap // 色標規則
	CategoryTypePromotionRuleMultiMap  RuleMultiMap // 促銷類別規則
	CategoryTypeItemIDRuleMultiMap     RuleMultiMap // CategoryTypeItemID規則

	CategoryRuleMultiMap map[con.CategoryType]*RuleMultiMap
	StoreItemMap         map[uint]StoreItem    // itemID 對應的 StoreItem (從庫存服務獲取)
	PosItemDraftMap      map[uint]PosItemDraft // posItemDraftID 對應的組合商品資訊 (從庫存服務獲取)
	LastReloadedTime     time.Time
}

// Singleton Pattern - 每個門市一個 PromotionEngine 實例，促銷規則變動時通知對應門市的 PromotionEngine 重新載入規則資料
var promotionEngineRegistry = struct {
	mu      sync.Mutex
	engines map[uint]*PromotionEngine
}{
	engines: make(map[uint]*PromotionEngine),
}

// get singleton instance of PromotionEngine for a storeID
func GetPromotionEngine(storeID uint) *PromotionEngine {
	promotionEngineRegistry.mu.Lock()
	defer promotionEngineRegistry.mu.Unlock()

	engine, exists := promotionEngineRegistry.engines[storeID]
	if exists {
		return engine
	}

	newEngine := NewPromotionEngine(storeID)
	promotionEngineRegistry.engines[storeID] = newEngine
	return newEngine
}

func NewPromotionEngine(storeID uint) *PromotionEngine {
	engine := PromotionEngine{
		StoreID:         storeID,
		StoreItemMap:    make(map[uint]StoreItem),
		PosItemDraftMap: make(map[uint]PosItemDraft),
	}
	engine.CategoryTypeLargeRuleMultiMap = *NewRuleMultiMap()
	engine.CategoryTypeMediumRuleMultiMap = *NewRuleMultiMap()
	engine.CategoryTypeSmallRuleMultiMap = *NewRuleMultiMap()
	engine.CategoryTypeColorLabelRuleMultiMap = *NewRuleMultiMap()
	engine.CategoryTypePromotionRuleMultiMap = *NewRuleMultiMap()
	engine.CategoryTypeItemIDRuleMultiMap = *NewRuleMultiMap()
	engine.CategoryRuleMultiMap = map[con.CategoryType]*RuleMultiMap{
		con.CategoryTypeLarge:      &engine.CategoryTypeLargeRuleMultiMap,
		con.CategoryTypeMedium:     &engine.CategoryTypeMediumRuleMultiMap,
		con.CategoryTypeSmall:      &engine.CategoryTypeSmallRuleMultiMap,
		con.CategoryTypeColorLabel: &engine.CategoryTypeColorLabelRuleMultiMap,
		con.CategoryTypePromotion:  &engine.CategoryTypePromotionRuleMultiMap,
		con.CategoryTypeItemID:     &engine.CategoryTypeItemIDRuleMultiMap,
	}
	return &engine
}

func (engine *PromotionEngine) NotifyCampaignChange() {
	engine.mu.Lock()
	defer engine.mu.Unlock()
	engine.LastReloadedTime = time.Time{}
}

func (engine *PromotionEngine) GetRulesForItem(itemID uint) []PromotionRule {
	rules := make([]PromotionRule, 0)
	for _, rule := range engine.CategoryTypeItemIDRuleMultiMap.Values() {
		switch rule.GetBase().RuleSource {
		case RuleSourceProductPromotionAmountOff:
			rule := rule.(ProductPromotionAmountOffRule)
			if rule.ItemID == itemID {
				rules = append(rules, rule)
			}
		case RuleSourceProductPromotionPercentage:
			rule := rule.(ProductPromotionPercentageRule)
			if rule.ItemID == itemID {
				rules = append(rules, rule)
			}
		case RuleSourceProductPromotionFixedPrice:
			rule := rule.(ProductPromotionFixedPriceRule)
			if rule.ItemID == itemID {
				rules = append(rules, rule)
			}
		}
	}
	return rules
}

func (engine *PromotionEngine) GetRulesForStoreItem(storeItem StoreItem) ([]PromotionRule, bool) {
	rules := make([]PromotionRule, 0)
	if storeItem.LargeCategoryID != 0 {
		categoryRuleSet, found := engine.CategoryTypeLargeRuleMultiMap.Get(storeItem.LargeCategoryID)
		if found {
			rules = append(rules, categoryRuleSet.GetSlice()...)
		}
	}
	if storeItem.MediumCategoryID != 0 {
		categoryRuleSet, found := engine.CategoryTypeMediumRuleMultiMap.Get(storeItem.MediumCategoryID)
		if found {
			rules = append(rules, categoryRuleSet.GetSlice()...)
		}
	}
	if storeItem.SmallCategoryID != 0 {
		categoryRuleSet, found := engine.CategoryTypeSmallRuleMultiMap.Get(storeItem.SmallCategoryID)
		if found {
			rules = append(rules, categoryRuleSet.GetSlice()...)
		}
	}
	if storeItem.ColorLabelID != 0 {
		categoryRuleSet, found := engine.CategoryTypeColorLabelRuleMultiMap.Get(storeItem.ColorLabelID)
		if found {
			rules = append(rules, categoryRuleSet.GetSlice()...)
		}
	}
	if storeItem.PromotionCategoryID != 0 {
		categoryRuleSet, found := engine.CategoryTypePromotionRuleMultiMap.Get(storeItem.PromotionCategoryID)
		if found {
			rules = append(rules, categoryRuleSet.GetSlice()...)
		}
	}
	if storeItem.ItemID != 0 {
		categoryRuleSet, found := engine.CategoryTypeItemIDRuleMultiMap.Get(storeItem.ItemID)
		if found {
			rules = append(rules, categoryRuleSet.GetSlice()...)
		}
	}

	return rules, true
}

// func (engine *PromotionEngine) ValidateQuoteDetails(details []QuoteLine, memberInfo MemberInfo) []*ServiceErrorDetail {
// 	engine.mu.RLock()
// 	defer engine.mu.RUnlock()

// 	validateErrList := make([]*ServiceErrorDetail, 0)
// 	for index, detail := range details {
// 		switch {
// 		case detail.ItemID != nil:
// 			storeItem, exist := engine.StoreItemMap[*detail.ItemID]
// 			if !exist {
// 				field := "item_id"
// 				validateErrList = append(validateErrList, NewServiceErrorDetail(&index, &field, NotExist, fmt.Sprintf("商品ID %d 不存在", *detail.ItemID), nil))
// 				continue
// 			}
// 			if !storeItem.AllowSale {
// 				field := "item_id"
// 				validateErrList = append(validateErrList, NewServiceErrorDetail(&index, &field, Incorrect, fmt.Sprintf("商品ID %d 不允許銷售", *detail.ItemID), nil))
// 				continue
// 			}
// 			if detail.ChangedUnitPrice != nil && !storeItem.AllowPriceChanged {
// 				field := "changed_unit_price"
// 				validateErrList = append(validateErrList, NewServiceErrorDetail(&index, &field, Incorrect, fmt.Sprintf("商品ID %d 不允許變價", *detail.ItemID), nil))
// 				continue
// 			}
// 			if detail.ChangedUnitPrice != nil && detail.ChangedUnitPrice.LessThan(decimal.Zero) {
// 				field := "changed_unit_price"
// 				validateErrList = append(validateErrList, NewServiceErrorDetail(&index, &field, Incorrect, fmt.Sprintf("商品ID %d 變價後單價不能為負", *detail.ItemID), nil))
// 				continue
// 			}
// 			if detail.CampaignDetailID != nil {
// 				rules := engine.GetRulesForItem(*detail.ItemID)
// 				if len(rules) == 0 {
// 					field := "campaign_detail_id"
// 					validateErrList = append(validateErrList, NewServiceErrorDetail(&index, &field, NotExist, fmt.Sprintf("商品ID %d 沒有可用促銷方案", *detail.ItemID), nil))
// 					continue
// 				}

// 				matched := false
// 				var matchedRule RuleBase
// 				for _, rule := range rules {
// 					if rule.GetBase().CampaignDetailID == *detail.CampaignDetailID {
// 						matched = true
// 						matchedRule = rule.GetBase()
// 						break
// 					}
// 				}
// 				if !matched {
// 					field := "campaign_detail_id"
// 					validateErrList = append(validateErrList, NewServiceErrorDetail(&index, &field, NotExist, fmt.Sprintf("商品ID %d 不存在 campaign_detail_id %d 的促銷方案", *detail.ItemID, *detail.CampaignDetailID), nil))
// 					continue
// 				}
// 				if len(matchedRule.CardTypeIDSet) > 0 && !sets.FromMap(matchedRule.CardTypeIDSet).Has(memberInfo.CardTypeID) {
// 					field := "membership_id"
// 					sets.FromMap(matchedRule.CardTypeIDSet)
// 					validateErrList = append(validateErrList, NewServiceErrorDetail(&index, &field, NotExist, fmt.Sprintf("促銷方案 campaign_detail_id %d 只適用會員卡別ID %s", *detail.CampaignDetailID, matchedRule.CardTypeIDSet.String()), nil))
// 					continue
// 				}

// 				// reject if quantity doesn't meet the threshold
// 				if detail.Quantity.LessThan(matchedRule.Quantity) {
// 					field := "campaign_detail_id"
// 					validateErrList = append(validateErrList, NewServiceErrorDetail(&index, &field, Incorrect,
// 						fmt.Sprintf("促銷方案 campaign_detail_id %d 需要數量至少 %s, 目前數量 %s 不符資格",
// 							*detail.CampaignDetailID, matchedRule.Quantity.String(), detail.Quantity.String()), nil))
// 					continue
// 				}
// 			}
// 		case detail.PosItemDraftID != nil:
// 			_, exist := engine.PosItemDraftMap[*detail.PosItemDraftID]
// 			if !exist {
// 				field := "pos_item_draft_id"
// 				validateErrList = append(validateErrList, NewServiceErrorDetail(&index, &field, NotExist, fmt.Sprintf("組合商品ID %d 不存在", *detail.PosItemDraftID), nil))
// 				continue
// 			}
// 			if detail.ChangedUnitPrice != nil {
// 				field := "changed_unit_price"
// 				validateErrList = append(validateErrList, NewServiceErrorDetail(&index, &field, Incorrect, fmt.Sprintf("組合商品ID %d 不允許變價", *detail.PosItemDraftID), nil))
// 				continue
// 			}
// 			if detail.ManualDiscount != nil {
// 				field := "manual_discount"
// 				validateErrList = append(validateErrList, NewServiceErrorDetail(&index, &field, Incorrect, fmt.Sprintf("組合商品ID %d 不允許手動折讓", *detail.PosItemDraftID), nil))
// 				continue
// 			}
// 			if detail.ManualDiscountRatio != nil {
// 				field := "manual_discount_ratio"
// 				validateErrList = append(validateErrList, NewServiceErrorDetail(&index, &field, Incorrect, fmt.Sprintf("組合商品ID %d 不允許手動折扣", *detail.PosItemDraftID), nil))
// 				continue
// 			}
// 		default:
// 			field := "item_id / pos_item_draft_id"
// 			validateErrList = append(validateErrList, NewServiceErrorDetail(&index, &field, Incorrect, "必須提供商品ID或組合商品ID", nil))
// 			continue
// 		}
// 	}

// 	return validateErrList
// }

func (engine *PromotionEngine) UpdatePosItemDraft(c *contexts.Quote, requested map[uint]struct{}) error {
	engine.mu.Lock()
	defer engine.mu.Unlock()

	missingPosItemDraftIDSet := sets.FromMap(requested).Difference(sets.FromMap(engine.PosItemDraftMap))
	if len(missingPosItemDraftIDSet) > 0 {
		err := engine.updateStoreItemMapFromGrpcForPosItemDraft(c, missingPosItemDraftIDSet)
		if err != nil {
			return err
		}
	}
	return nil
}

func (engine *PromotionEngine) updateStoreItemMapFromGrpcForPosItemDraft(c *contexts.Quote, posItemDraftIDSet sets.Set[uint]) error {
	invGrpc := grpc_inv.NewInvGrpcWrapper2(c.InventoryGrpcClient, &c.UserInfo)
	grpcPosItemDraftMap, err := invGrpc.GetPosItemDrafts(posItemDraftIDSet.Slice())
	if err != nil {
		return err
	}
	if len(grpcPosItemDraftMap) < len(posItemDraftIDSet) {
		fmt.Printf("從Inventory GRPC API獲取的組合商品資訊數量與請求的數量不匹配,可能是部分itemID不存在. requested itemIDs: %v, response store items count: %d", posItemDraftIDSet.Slice(), len(grpcPosItemDraftMap))
		diffSet := posItemDraftIDSet.Difference(sets.FromMap(grpcPosItemDraftMap))
		return NewServiceErrorDetail(nil, nil, NotExist, fmt.Sprintf("從Inventory GRPC API獲取的組合商品資訊缺少itemID: %v", diffSet.Slice()), nil)
	}
	for _, posItemDraft := range grpcPosItemDraftMap {
		engine.PosItemDraftMap[posItemDraft.ID] = posItemDraft
	}
	return nil
}

func (engine *PromotionEngine) UpdateStoreItem(c *contexts.Quote, requested sets.Set[uint]) error {
	engine.mu.Lock()
	defer engine.mu.Unlock()

	missingItemIDSet := requested.Difference(sets.FromMap(engine.StoreItemMap))
	if len(missingItemIDSet) > 0 {
		err := engine.UpdateStoreItemMapFromGrpc(c, missingItemIDSet)
		if err != nil {
			return err
		}
	}
	return nil
}

func (engine *PromotionEngine) Reload(c *contexts.Quote, itemIDList []uint) error {
	engine.mu.Lock()
	defer engine.mu.Unlock()

	if !engine.LastReloadedTime.Equal(time.Time{}) && time.Since(engine.LastReloadedTime) < 3*time.Minute {
		fmt.Printf("促銷規則資料在3分鐘內已載入過,跳過重新載入\n")
		return nil
	}

	schemaMdl := NewMultiSchemaMdlV2(c.UserInfo, PromotionV2SharedSchema)
	// collect ItemIDToRule and RuleMultiMap
	campDetails, err := c.CampaignRepo.GetActiveCampaignDetailList(schemaMdl, engine.StoreID)
	if err != nil {
		return err
	}
	if len(campDetails) == 0 {
		fmt.Printf("沒有促銷規則資料,不需要載入商品資訊\n")
		engine.LastReloadedTime = time.Now()
		return nil
	}
	for _, campDetail := range campDetails {
		if campDetail.ProductPromotion != nil {
			promo := *campDetail.ProductPromotion
			cardTypeIDs, err := c.ProductPromotionRepo.GetCardTypeIDs(schemaMdl, promo.ID, false) // 順便載入會員卡別資料
			if err != nil {
				return err
			}
			cardTypeIDSet := sets.New(cardTypeIDs...)
			switch promo.DiscountMethod {
			case con.DiscountMethodAmountOff:
				for _, detail := range promo.AmountOffDetails {
					rule := RuleFromProductPromotionAmountOffDetail(campDetail, detail, promo, cardTypeIDSet)
					engine.CategoryTypeItemIDRuleMultiMap.Put(detail.ItemID, rule)
				}
			case con.DiscountMethodPercentage:
				for _, detail := range promo.PercentageDetails {
					rule := RuleFromProductPromotionPercentageDetail(campDetail, detail, promo, cardTypeIDSet)
					engine.CategoryTypeItemIDRuleMultiMap.Put(detail.ItemID, rule)
				}
			case con.DiscountMethodFixedPrice:
				for _, detail := range promo.FixedPriceDetails {
					rule := RuleFromProductPromotionFixedPriceDetail(campDetail, detail, promo, cardTypeIDSet)
					engine.CategoryTypeItemIDRuleMultiMap.Put(detail.ItemID, rule)
				}
			}

		} else if campDetail.CategoricalPromotion != nil {
			promo := *campDetail.CategoricalPromotion
			cardTypeIDs, err := c.CategoricalPromotionRepo.GetCardTypeIDs(schemaMdl, promo.ID, false) // 順便載入會員卡別資料
			if err != nil {
				return err
			}
			cardTypeIDSet := sets.New(cardTypeIDs...)
			for _, detail := range promo.AllowList {
				rule := RuleFromCategoricalPromotionAllowListDetail(campDetail, detail, promo, cardTypeIDSet)
				engine.CategoryRuleMultiMap[detail.CategoryType].Put(detail.CategoryID, rule)
			}
			for _, detail := range promo.BlockList {
				rule := RuleFromCategoricalPromotionBlockListDetail(campDetail, detail, promo, cardTypeIDSet)
				engine.CategoryRuleMultiMap[detail.CategoryType].Put(detail.CategoryID, rule)
			}
		}
	}

	engine.StoreItemMap = make(map[uint]StoreItem, len(itemIDList))
	if len(itemIDList) > 0 {
		itemIDSet := sets.New(itemIDList...)
		err = engine.UpdateStoreItemMapFromGrpc(c, itemIDSet)
		if err != nil {
			return err
		}
	}
	engine.PosItemDraftMap = make(map[uint]PosItemDraft, len(itemIDList))
	engine.LastReloadedTime = time.Now()
	return nil
}

func (engine *PromotionEngine) UpdateStoreItemMapFromGrpc(c *contexts.Quote, missingStoreItemIDSet sets.Set[uint]) error {
	invGrpc := grpc_inv.NewInvGrpcWrapper2(c.InventoryGrpcClient, &c.UserInfo)
	storeItemMap, err := invGrpc.GetStoreItemMap(missingStoreItemIDSet.Slice())
	if err != nil {
		return err
	}
	if len(storeItemMap) < len(missingStoreItemIDSet) {
		fmt.Printf("從Inventory GRPC API獲取的商品資訊數量與請求的數量不匹配,可能是部分itemID不存在. requested itemIDs: %v, response store items count: %d\n", missingStoreItemIDSet.Slice(), len(storeItemMap))
		storeItemIDSet := sets.FromMap(storeItemMap)
		diffSet := missingStoreItemIDSet.Difference(storeItemIDSet)
		return NewServiceErrorDetail(nil, nil, NotExist, fmt.Sprintf("從Inventory GRPC API獲取的商品資訊缺少itemID: %v", diffSet.Slice()), nil)
	}
	for _, storeItem := range storeItemMap {
		engine.StoreItemMap[storeItem.ItemID] = storeItem
	}
	return nil
}

// // QuoteManualPromoDetail 處理「前端手動指定促銷」的單筆明細。
// // 輸出的 CampaignDetailID 一律沿用前端傳入的值，即使數量未達門檻而未能享有折扣，
// // 也必須保留此欄位，以確保該筆明細不會被合併至沒有 CampaignDetailID 的自動促銷明細。
// func (engine *PromotionEngine) QuoteManualPromoDetail(detail dtos.QuoteDetail, memberInfo MemberInfo) viewmodels.QuoteDetail {
// 	engine.mu.RLock()
// 	defer engine.mu.RUnlock()

// 	storeItem := engine.StoreItemMap[*detail.ItemID]
// 	unitPrice := storeItem.GetPriceByType(memberInfo.PriceType, memberInfo.Discount)
// 	originalSubtotal := unitPrice.Mul(detail.Quantity)

// 	quoteResult := engine.quoteItemByCampaignDetailID(*detail.ItemID, detail.Quantity, unitPrice, memberInfo.Discount, *detail.CampaignDetailID)
// 	pointsRedeemed := detail.PointsRedeemed
// 	vmDetail := QuoteDetailDTOToVM(detail)
// 	vmDetail.ItemName = storeItem.Name
// 	vmDetail.ItemNo = storeItem.No
// 	vmDetail.OriginalUnitPrice = unitPrice
// 	vmDetail.OriginalSubtotal = originalSubtotal
// 	vmDetail.AppliedSubtotal = quoteResult.AppliedSubtotal
// 	vmDetail.PromotionName = quoteResult.PromotionName
// 	vmDetail.PointsRedeemed = pointsRedeemed
// 	return vmDetail
// }

// func QuoteDetailDTOToVM(dto dtos.QuoteDetail) viewmodels.QuoteDetail {
// 	return viewmodels.QuoteDetail{
// 		ItemID:         dto.ItemID,
// 		PosItemDraftID: dto.PosItemDraftID,
// 		CashierID:      dto.CashierID,
// 		Remark:         dto.Remark,
// 		Quantity:       dto.Quantity,

// 		LineIndex:        dto.LineIndex,
// 		CampaignDetailID: dto.CampaignDetailID,
// 		PointsRedeemed:   dto.PointsRedeemed,

// 		QuoteLineKind:     dto.QuoteLineKind,
// 		MemberWarehouseID: dto.MemberWarehouseID,

// 		IsManualGift:        dto.IsManualGift,
// 		ChangedUnitPrice:    dto.ChangedUnitPrice,
// 		ManualDiscount:      dto.ManualDiscount,
// 		ManualDiscountRatio: dto.ManualDiscountRatio,
// 		QuotePriceSource:    dto.QuotePriceSource,

// 		IncomeExpensesReasonID: dto.IncomeExpensesReasonID,
// 		TaxType:                dto.TaxType,
// 		LotNo:                  dto.LotNo,
// 	}
// }

// func (engine *PromotionEngine) CalcManualAdjustDetails(manualAdjustDetails []dtos.QuoteDetail, memberInfo MemberInfo) (vmDetails []viewmodels.QuoteDetail) {
// 	for _, detail := range manualAdjustDetails {
// 		vmDetail := viewmodels.QuoteDetail{
// 			ItemID:         detail.ItemID,
// 			PosItemDraftID: detail.PosItemDraftID,
// 			CashierID:      detail.CashierID,
// 			Remark:         detail.Remark,
// 			Quantity:       detail.Quantity,

// 			LineIndex:         detail.LineIndex,
// 			CampaignDetailID:  detail.CampaignDetailID,
// 			PointsRedeemed:    detail.PointsRedeemed,
// 			QuoteLineKind:     detail.QuoteLineKind,
// 			MemberWarehouseID: detail.MemberWarehouseID,

// 			IsManualGift:        detail.IsManualGift,
// 			ChangedUnitPrice:    detail.ChangedUnitPrice,
// 			ManualDiscount:      detail.ManualDiscount,
// 			ManualDiscountRatio: detail.ManualDiscountRatio,
// 			QuotePriceSource:    detail.QuotePriceSource,

// 			IncomeExpensesReasonID: detail.IncomeExpensesReasonID,
// 			TaxType:                detail.TaxType,
// 			LotNo:                  detail.LotNo,
// 		}
// 		if detail.ChangedUnitPrice != nil {
// 			vmDetail.AppliedSubtotal = detail.ChangedUnitPrice.Mul(detail.Quantity)
// 		} else if detail.ManualDiscount != nil {
// 			storeItem := engine.StoreItemMap[*detail.ItemID]
// 			unitPrice := storeItem.GetPriceByType(memberInfo.PriceType, memberInfo.Discount)
// 			discountedUnitPrice := unitPrice.Sub(*detail.ManualDiscount)
// 			vmDetail.AppliedSubtotal = discountedUnitPrice.Mul(detail.Quantity)
// 		} else if detail.ManualDiscountRatio != nil {
// 			storeItem := engine.StoreItemMap[*detail.ItemID]
// 			unitPrice := storeItem.GetPriceByType(memberInfo.PriceType, memberInfo.Discount)
// 			ratio := (*detail.ManualDiscountRatio).Div(decimal.NewFromInt(100))
// 			discountedUnitPrice := unitPrice.Mul(ratio)
// 			vmDetail.AppliedSubtotal = discountedUnitPrice.Mul(detail.Quantity)
// 		}
// 		vmDetails = append(vmDetails, vmDetail)
// 	}
// 	return
// }

// func (engine *PromotionEngine) CalcManualPromoDetails(manualPromoDetails []dtos.QuoteDetail, memberInfo MemberInfo) (vmDetails []viewmodels.QuoteDetail) {
// 	vmDetails = make([]viewmodels.QuoteDetail, 0, len(manualPromoDetails))
// 	for _, detail := range manualPromoDetails {

// 		campaignDetailID := *detail.CampaignDetailID
// 		promoRule, isExist := engine.GetRuleByCampaignDetailID(campaignDetailID)
// 		if !isExist {
// 			fmt.Printf("無法找到 campaign detail ID %d 對應的促銷規則\n", campaignDetailID)
// 			continue
// 		}
// 		if len(promoRule.GetBase().CardTypeIDSet) > 0 && !sets.FromMap(promoRule.GetBase().CardTypeIDSet).Has(memberInfo.CardTypeID) {
// 			fmt.Printf("會員卡別ID %d 不符合促銷方案 campaign detail ID %d 的資格要求\n", memberInfo.CardTypeID, campaignDetailID)
// 			continue
// 		}
// 		storeItem := engine.StoreItemMap[*detail.ItemID]
// 		originalUnitPrice := storeItem.GetPriceByType(memberInfo.PriceType, memberInfo.Discount)
// 		originalSubtotal := originalUnitPrice.Mul(detail.Quantity)
// 		appliedPrice := promoRule.Calculate(storeItem, detail.Quantity, originalUnitPrice, memberInfo.Discount)
// 		vmDetail := viewmodels.QuoteDetail{
// 			ItemID:         detail.ItemID,
// 			PosItemDraftID: detail.PosItemDraftID,
// 			CashierID:      detail.CashierID,
// 			Remark:         detail.Remark,
// 			Quantity:       detail.Quantity,

// 			LineIndex:         detail.LineIndex,
// 			CampaignDetailID:  detail.CampaignDetailID,
// 			PointsRedeemed:    detail.PointsRedeemed,
// 			QuoteLineKind:     detail.QuoteLineKind,
// 			MemberWarehouseID: detail.MemberWarehouseID,

// 			IsManualGift:      detail.IsManualGift,
// 			OriginalUnitPrice: originalUnitPrice,
// 			OriginalSubtotal:  originalSubtotal,
// 			AppliedSubtotal:   appliedPrice,
// 			QuotePriceSource:  detail.QuotePriceSource,

// 			IncomeExpensesReasonID: detail.IncomeExpensesReasonID,
// 			TaxType:                detail.TaxType,
// 			LotNo:                  detail.LotNo,
// 		}
// 		vmDetails = append(vmDetails, vmDetail)
// 	}
// 	return vmDetails
// }

// func (engine *PromotionEngine) CalcReturnDetails(returnDetails []dtos.QuoteDetail, memberInfo MemberInfo) (vmDetails []viewmodels.QuoteDetail) {
// 	for _, detail := range returnDetails {
// 		storeItem := engine.StoreItemMap[*detail.ItemID]
// 		unitPrice := storeItem.GetPriceByType(memberInfo.PriceType, memberInfo.Discount)
// 		originalSubtotal := unitPrice.Mul(detail.Quantity)

// 		vmDetail := viewmodels.QuoteDetail{
// 			ItemID:         detail.ItemID,
// 			PosItemDraftID: detail.PosItemDraftID,
// 			CashierID:      detail.CashierID,
// 			Remark:         detail.Remark,
// 			Quantity:       detail.Quantity,

// 			LineIndex:         detail.LineIndex,
// 			CampaignDetailID:  detail.CampaignDetailID,
// 			PointsRedeemed:    detail.PointsRedeemed,
// 			QuoteLineKind:     detail.QuoteLineKind,
// 			MemberWarehouseID: detail.MemberWarehouseID,

// 			IsManualGift:      detail.IsManualGift,
// 			OriginalUnitPrice: unitPrice.Neg(), // 退貨單價取負, 商家給顧客錢
// 			OriginalSubtotal:  originalSubtotal.Neg(),
// 			AppliedSubtotal:   originalSubtotal.Neg(), // 退貨不考慮促銷折扣
// 			QuotePriceSource:  detail.QuotePriceSource,

// 			IncomeExpensesReasonID: detail.IncomeExpensesReasonID,
// 			TaxType:                detail.TaxType,
// 			LotNo:                  detail.LotNo,
// 		}
// 		vmDetails = append(vmDetails, vmDetail)
// 	}
// 	return vmDetails
// }

// func (engine *PromotionEngine) CalcPickupDetails(pickupDetails []dtos.QuoteDetail, memberInfo MemberInfo) (vmDetails []viewmodels.QuoteDetail) {
// 	for _, detail := range pickupDetails {
// 		vmDetail := viewmodels.QuoteDetail{
// 			ItemID:         detail.ItemID,
// 			PosItemDraftID: detail.PosItemDraftID,
// 			CashierID:      detail.CashierID,
// 			Remark:         detail.Remark,
// 			Quantity:       detail.Quantity,

// 			LineIndex:         detail.LineIndex,
// 			CampaignDetailID:  detail.CampaignDetailID,
// 			PointsRedeemed:    detail.PointsRedeemed,
// 			QuoteLineKind:     detail.QuoteLineKind,
// 			MemberWarehouseID: detail.MemberWarehouseID,

// 			IsManualGift:      detail.IsManualGift,
// 			OriginalUnitPrice: decimal.Zero,
// 			OriginalSubtotal:  decimal.Zero,
// 			AppliedSubtotal:   decimal.Zero, // 取貨不考慮花費
// 			QuotePriceSource:  detail.QuotePriceSource,

// 			IncomeExpensesReasonID: detail.IncomeExpensesReasonID,
// 			TaxType:                detail.TaxType,
// 			LotNo:                  detail.LotNo,
// 		}
// 		vmDetails = append(vmDetails, vmDetail)
// 	}
// 	return vmDetails
// }

// func (engine *PromotionEngine) CalcReturnPickupDetails(returnPickupDetails []dtos.QuoteDetail, memberInfo MemberInfo) (vmDetails []viewmodels.QuoteDetail) {
// 	for _, detail := range returnPickupDetails {
// 		vmDetail := viewmodels.QuoteDetail{
// 			ItemID:         detail.ItemID,
// 			PosItemDraftID: detail.PosItemDraftID,
// 			CashierID:      detail.CashierID,
// 			Remark:         detail.Remark,
// 			Quantity:       detail.Quantity,

// 			LineIndex:         detail.LineIndex,
// 			CampaignDetailID:  detail.CampaignDetailID,
// 			PointsRedeemed:    detail.PointsRedeemed,
// 			QuoteLineKind:     detail.QuoteLineKind,
// 			MemberWarehouseID: detail.MemberWarehouseID,

// 			IsManualGift:      detail.IsManualGift,
// 			OriginalUnitPrice: decimal.Zero,
// 			OriginalSubtotal:  decimal.Zero,
// 			AppliedSubtotal:   decimal.Zero, // 退取貨不考慮退錢
// 			QuotePriceSource:  detail.QuotePriceSource,

// 			IncomeExpensesReasonID: detail.IncomeExpensesReasonID,
// 			TaxType:                detail.TaxType,
// 			LotNo:                  detail.LotNo,
// 		}
// 		vmDetails = append(vmDetails, vmDetail)
// 	}
// 	return vmDetails
// }

// func (engine *PromotionEngine) quoteItemByCampaignDetailID(itemID uint, quantity, unitPrice, memberDiscount decimal.Decimal, campaignDetailID uint) QuoteLine {
// 	originalSubtotal := quantity.Mul(unitPrice)
// 	result := QuoteLine{
// 		ItemID:           itemID,
// 		Quantity:         quantity,
// 		UnitPrice:        unitPrice,
// 		OriginalSubtotal: originalSubtotal,
// 		AppliedSubtotal:  originalSubtotal,
// 		CampaignDetailID: &campaignDetailID,
// 	}
// 	storeItem := engine.StoreItemMap[itemID]
// 	rule, found := engine.GetRuleByCampaignDetailID(campaignDetailID)
// 	if found {
// 		appliedSubtotal := rule.Calculate(storeItem, quantity, unitPrice, memberDiscount)
// 		result.AppliedSubtotal = appliedSubtotal
// 		// result.WinningRule = rule.GetBase()
// 	}
// 	return result
// }

// func (engine *PromotionEngine) buildQuoteLine(autoPromoDetails []dtos.QuoteDetail, memberInfo MemberInfo) (quoteLines []QuoteLine) {
// 	quoteLines = make([]QuoteLine, 0, len(autoPromoDetails)) // 預先分配足夠容量以避免後續 append 時多次擴容
// 	for _, detail := range autoPromoDetails {
// 		storeItem := engine.StoreItemMap[*detail.ItemID] // 確保 StoreItemMap 中有對應的商品資訊, 以避免後續計算促銷時找不到商品資訊導致錯誤
// 		quoteLine := QuoteLine{
// 			ItemID:          storeItem.ItemID,
// 			MainCategoryID:  storeItem.MainCategoryID,
// 			CategoryID:      storeItem.CategoryID,
// 			SubcategoryID:   storeItem.SubcategoryID,
// 			SalesLabelID:    storeItem.SalesLabelID,
// 			PromotionID:     storeItem.PromotionID,
// 			UnitPrice:       storeItem.GetPriceByType(memberInfo.PriceType, memberInfo.Discount),
// 			AtomID:          fmt.Sprintf("%d", detail.LineIndex),
// 			SourceLineIndex: detail.LineIndex,
// 			Quantity:        detail.Quantity,
// 		}
// 		quoteLines = append(quoteLines, quoteLine)
// 	}
// 	return quoteLines
// }

type EligibleLine struct {
	// LineIndex       int
	ItemID          uint
	Quantity        int64
	AppliedSubtotal decimal.Decimal
	// UnitPrice decimal.Decimal
	Consumed bool
}

type RewardLine struct {
	LineIndex      int
	ItemID         uint
	Qty            decimal.Decimal
	DiscountAmount decimal.Decimal
	FinalUnitPrice decimal.Decimal
}

type PromotionCandidate struct {
	CampaignDetailID uint
	Rule             PromotionRule

	EligibleLines []EligibleLine
	RewardLines   []RewardLine

	DiscountAmount decimal.Decimal
	Priority       *big.Int

	AppliedQty decimal.Decimal
}

func (engine *PromotionEngine) matchPromotionRules(quoteLines []QuoteLine, memberInfo MemberInfo) (candidates []PromotionCandidate) {
	ruleList := make([]PromotionRule, 0)
	ruleList = append(ruleList, engine.CategoryTypeLargeRuleMultiMap.Values()...)
	ruleList = append(ruleList, engine.CategoryTypeMediumRuleMultiMap.Values()...)
	ruleList = append(ruleList, engine.CategoryTypeSmallRuleMultiMap.Values()...)
	ruleList = append(ruleList, engine.CategoryTypeColorLabelRuleMultiMap.Values()...)
	ruleList = append(ruleList, engine.CategoryTypePromotionRuleMultiMap.Values()...)
	ruleList = append(ruleList, engine.CategoryTypeItemIDRuleMultiMap.Values()...)
	sort.SliceStable(ruleList, func(i, j int) bool {
		return ruleList[i].GetBase().HasHigherPriorityThan(ruleList[j].GetBase())
	})
	for _, rule := range ruleList {
		for _, line := range quoteLines {
			if rule.IsEligible(line, memberInfo) {
				candicate := PromotionCandidate{
					CampaignDetailID: rule.GetBase().CampaignDetailID,
					Rule:             rule,
					EligibleLines: []EligibleLine{
						{
							ItemID:          line.ItemID,
							Quantity:        line.Quantity,
							AppliedSubtotal: line.AppliedSubtotal,
							Consumed:        true, // 預設同一筆明細只能被一個促銷規則消費, 後續套用促銷規則時會檢查此欄位來確保不會重複套用
						},
					},
					DiscountAmount: decimal.Zero,                 // 先預設為0, 實際折扣金額要等到套用促銷規則時計算
					Priority:       rule.GetBase().GetAbsValue(), // 以促銷規則的權重作為優先順序, 權重數字越大優先套用
					AppliedQty:     decimal.Zero,                 // 先預設為0, 實際套用數量要等到套用促銷規則時計算
					RewardLines:    []RewardLine{},
				}
				candidates = append(candidates, candicate)
			}
		}
	}
	return
}

func (engine *PromotionEngine) applyPromotionRules(quoteLines []QuoteLine, candidates []PromotionCandidate, memberInfo MemberInfo) (updatedQuoteLines []QuoteLine) {
	updatedQuoteLines = make([]QuoteLine, len(quoteLines))
	copy(updatedQuoteLines, quoteLines)

	// 初始化每筆明細的原始小計與套用小計
	for i := range updatedQuoteLines {
		original := updatedQuoteLines[i].UnitPrice.Mul(decimal.NewFromInt(updatedQuoteLines[i].Quantity))
		updatedQuoteLines[i].AppliedSubtotal = original
	}

	// 紀錄已被促銷消費的明細 LineIndex，確保同一筆明細只套用一個促銷規則
	// consumedLines := make(map[int]bool)

	for _, candidate := range candidates {
		rule := candidate.Rule
		base := rule.GetBase()
		base = base

		for _, eligibleLine := range candidate.EligibleLines {
			for i := range updatedQuoteLines {

				// 封鎖清單規則：標記為已消費但不套用折扣，避免 AllowList 規則誤套用
				if _, isBlockList := rule.(CategoricalPromotionBlockListRule); isBlockList {
					break
				}

				storeItem := engine.StoreItemMap[eligibleLine.ItemID]
				appliedSubtotal := rule.Calculate(storeItem, eligibleLine.Quantity, eligibleLine.UnitPrice, memberInfo.Discount)
				updatedQuoteLines[i].AppliedSubtotal = appliedSubtotal
				break
			}
		}
	}
	return
}

// func (engine *PromotionEngine) QuoteLinesToVmDetails(quoteLines []QuoteLine, autoPromoDetails []dtos.QuoteDetail) (vmDetails []viewmodels.QuoteDetail) {
// 	vmDetails = make([]viewmodels.QuoteDetail, 0, len(quoteLines))
// 	for _, quoteLine := range quoteLines {
// 		var sourceQuoteDetail *dtos.QuoteDetail = nil
// 		for i := range autoPromoDetails {
// 			if autoPromoDetails[i].LineIndex == quoteLine.SourceLineIndex {
// 				sourceQuoteDetail = &autoPromoDetails[i]
// 				break
// 			}
// 		}

// 		vmDetail := viewmodels.QuoteDetail{
// 			ItemID:    sourceQuoteDetail.ItemID,
// 			CashierID: sourceQuoteDetail.CashierID,
// 			Remark:    sourceQuoteDetail.Remark,
// 			Quantity:  quoteLine.Quantity,

// 			AtomID:           quoteLine.AtomID,
// 			CampaignDetailID: quoteLine.CampaignDetailID,
// 			QuoteLineKind:    sourceQuoteDetail.QuoteLineKind,

// 			OriginalSubtotal: quoteLine.OriginalSubtotal,
// 			AppliedSubtotal:  quoteLine.AppliedSubtotal,
// 			QuotePriceSource: sourceQuoteDetail.QuotePriceSource,
// 			PromotionName:    quoteLine.PromotionName,

// 			TaxType: sourceQuoteDetail.TaxType,
// 			LotNo:   sourceQuoteDetail.LotNo,
// 		}
// 		vmDetails = append(vmDetails, vmDetail)
// 	}
// 	return vmDetails
// }

// func (engine *PromotionEngine) CalcAutoPromo(autoPromoDetails []dtos.QuoteDetail, memberInfo MemberInfo) (vmDetails []viewmodels.QuoteDetail) {
// 	quoteLines := engine.buildQuoteLine(autoPromoDetails, memberInfo)
// 	candidates := engine.matchPromotionRules(quoteLines, memberInfo) // 先套用符合條件的促銷規則, 以確保後續套用促銷規則時能正確計算出折扣後的價格來判斷是否符合資格
// 	quoteLines = engine.applyPromotionRules(quoteLines, candidates, memberInfo)
// 	vmDetail := engine.QuoteLinesToVmDetails(quoteLines, autoPromoDetails) // 再套用促銷規則計算出最終的折扣金額
// 	return vmDetail
// }

func (engine *PromotionEngine) GetRuleByCampaignDetailID(campaignDetailID uint) (promo PromotionRule, isExist bool) {
	engine.mu.RLock()
	defer engine.mu.RUnlock()

	for _, ruleMultiMap := range engine.CategoryRuleMultiMap {
		for _, rule := range ruleMultiMap.Values() {
			if rule.GetBase().CampaignDetailID == campaignDetailID {
				return rule, true
			}
		}
	}
	return nil, false
}

func RuleFromProductPromotionAmountOffDetail(campaignDetail entities.CampaignDetail, detail entities.ProductPromotionAmountOffDetail, promo entities.ProductPromotion, cardTypeIDSet sets.Set[uint]) ProductPromotionAmountOffRule {
	return ProductPromotionAmountOffRule{
		RuleBase: RuleBase{
			RuleSource:             RuleSourceProductPromotionAmountOff,
			PromotionDetailID:      detail.ID,
			Title:                  promo.Title,
			Remark:                 promo.Remark,
			CardTypeIDSet:          cardTypeIDSet,
			Weight:                 campaignDetail.Weight,
			Sequence:               campaignDetail.Sequence,
			CampaignDetailID:       campaignDetail.ID,
			CanEarnPoints:          promo.CanEarnPoints,
			CanDoublePoints:        promo.CanDoublePoints,
			CanAccumulateSpend:     promo.CanAccumulateSpend,
			CanMeetThreshold:       promo.CanMeetThreshold,
			QuantityDiscountMethod: promo.QuantityDiscountMethod,
			Quantity:               detail.Quantity,
			PurchaseLimit:          detail.PurchaseLimit,
			ResetDays:              detail.ResetDays,
			PointsRedemption:       detail.PointsRedemption,
		},
		ItemID:     detail.ItemID,
		PriceLevel: detail.PriceLevel,
		AmountOff:  detail.AmountOff,
	}
}

func RuleFromProductPromotionPercentageDetail(campaignDetail entities.CampaignDetail, detail entities.ProductPromotionPercentageDetail, promo entities.ProductPromotion, cardTypeIDSet sets.Set[uint]) ProductPromotionPercentageRule {
	return ProductPromotionPercentageRule{
		RuleBase: RuleBase{
			RuleSource:             RuleSourceProductPromotionPercentage,
			PromotionDetailID:      detail.ID,
			Title:                  promo.Title,
			Remark:                 promo.Remark,
			CardTypeIDSet:          cardTypeIDSet,
			Weight:                 campaignDetail.Weight,
			Sequence:               campaignDetail.Sequence,
			CampaignDetailID:       campaignDetail.ID,
			CanEarnPoints:          promo.CanEarnPoints,
			CanDoublePoints:        promo.CanDoublePoints,
			CanAccumulateSpend:     promo.CanAccumulateSpend,
			CanMeetThreshold:       promo.CanMeetThreshold,
			QuantityDiscountMethod: promo.QuantityDiscountMethod,
			Quantity:               detail.Quantity,
			PurchaseLimit:          detail.PurchaseLimit,
			ResetDays:              detail.ResetDays,
			PointsRedemption:       detail.PointsRedemption,
		},
		ItemID:     detail.ItemID,
		PriceLevel: detail.PriceLevel,
		Percentage: detail.Percentage,
	}
}

func RuleFromProductPromotionFixedPriceDetail(campaignDetail entities.CampaignDetail, detail entities.ProductPromotionFixedPriceDetail, promo entities.ProductPromotion, cardTypeIDSet sets.Set[uint]) ProductPromotionFixedPriceRule {
	return ProductPromotionFixedPriceRule{
		RuleBase: RuleBase{
			RuleSource:             RuleSourceProductPromotionFixedPrice,
			PromotionDetailID:      detail.ID,
			Title:                  promo.Title,
			Remark:                 promo.Remark,
			CardTypeIDSet:          cardTypeIDSet,
			Weight:                 campaignDetail.Weight,
			Sequence:               campaignDetail.Sequence,
			CampaignDetailID:       campaignDetail.ID,
			CanEarnPoints:          promo.CanEarnPoints,
			CanDoublePoints:        promo.CanDoublePoints,
			CanAccumulateSpend:     promo.CanAccumulateSpend,
			CanMeetThreshold:       promo.CanMeetThreshold,
			QuantityDiscountMethod: promo.QuantityDiscountMethod,
			Quantity:               detail.Quantity,
			PurchaseLimit:          detail.PurchaseLimit,
			ResetDays:              detail.ResetDays,
			PointsRedemption:       detail.PointsRedemption,
		},
		ItemID:     detail.ItemID,
		FixedPrice: detail.Price,
	}
}

func RuleFromCategoricalPromotionAllowListDetail(campaignDetail entities.CampaignDetail, detail entities.CategoricalPromotionAllowDetail, promo entities.CategoricalPromotion, cardTypeIDSet sets.Set[uint]) CategoricalPromotionAllowListRule {
	return CategoricalPromotionAllowListRule{
		RuleBase: RuleBase{
			RuleSource:             RuleSourceCategoricalPromotionAllowList,
			PromotionDetailID:      detail.ID,
			Title:                  promo.Title,
			Remark:                 promo.Remark,
			CardTypeIDSet:          cardTypeIDSet,
			Weight:                 campaignDetail.Weight,
			Sequence:               campaignDetail.Sequence,
			CampaignDetailID:       campaignDetail.ID,
			CanEarnPoints:          promo.CanEarnPoints,
			CanDoublePoints:        promo.CanDoublePoints,
			CanAccumulateSpend:     promo.CanAccumulateSpend,
			CanMeetThreshold:       promo.CanMeetThreshold,
			QuantityDiscountMethod: promo.QuantityDiscountMethod,
			Quantity:               promo.Quantity,
			PurchaseLimit:          promo.PurchaseLimit,
			ResetDays:              promo.ResetDays,
			PointsRedemption:       promo.PointsRedemption,
		},
		PriceLevel: promo.PriceLevel,
		Percentage: promo.Percentage,
		CategoryIDTuple: con.CategoryIDTuple{
			CategoryType: detail.CategoryType,
			CategoryID:   detail.CategoryID,
		},
	}
}

func RuleFromCategoricalPromotionBlockListDetail(campaignDetail entities.CampaignDetail, detail entities.CategoricalPromotionBlockDetail, promo entities.CategoricalPromotion, cardTypeIDSet sets.Set[uint]) CategoricalPromotionBlockListRule {
	return CategoricalPromotionBlockListRule{
		RuleBase: RuleBase{
			RuleSource:             RuleSourceCategoricalPromotionBlockList,
			PromotionDetailID:      detail.ID,
			Title:                  promo.Title,
			Remark:                 promo.Remark,
			CardTypeIDSet:          cardTypeIDSet,
			Weight:                 campaignDetail.Weight,
			Sequence:               campaignDetail.Sequence,
			CampaignDetailID:       campaignDetail.ID,
			CanEarnPoints:          promo.CanEarnPoints,
			CanDoublePoints:        promo.CanDoublePoints,
			CanAccumulateSpend:     promo.CanAccumulateSpend,
			CanMeetThreshold:       promo.CanMeetThreshold,
			QuantityDiscountMethod: promo.QuantityDiscountMethod,
			Quantity:               promo.Quantity,
			PurchaseLimit:          promo.PurchaseLimit,
			ResetDays:              promo.ResetDays,
			PointsRedemption:       promo.PointsRedemption,
		},
		PriceLevel: promo.PriceLevel,
		Percentage: promo.Percentage,
		CategoryIDTuple: con.CategoryIDTuple{
			CategoryType: detail.CategoryType,
			CategoryID:   detail.CategoryID,
		},
	}
}
