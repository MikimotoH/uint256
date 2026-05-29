package promo_engine_atomize

import (
	"fmt"

	con "pos-backend/pkg/constants"

	"github.com/R167/go-sets"
	"github.com/shopspring/decimal"
)

// implements PromotionRule interface

// ProductPromotion
type ProductPromotionAmountOffRule struct {
	RuleBase
	ItemID     uint
	PriceLevel uint8
	AmountOff  decimal.Decimal
}
type ProductPromotionPercentageRule struct {
	RuleBase
	ItemID     uint
	PriceLevel uint8
	Percentage decimal.Decimal
}
type ProductPromotionFixedPriceRule struct {
	RuleBase
	ItemID     uint
	FixedPrice decimal.Decimal
}

// CategoricalPromotion
type CategoricalPromotionAllowListRule struct {
	RuleBase

	PriceLevel      uint8
	Percentage      decimal.Decimal
	CategoryIDTuple con.CategoryIDTuple
}
type CategoricalPromotionBlockListRule struct {
	RuleBase

	PriceLevel      uint8
	Percentage      decimal.Decimal
	CategoryIDTuple con.CategoryIDTuple
}

// Implements Interface PromotionRule (Getter)
func (r ProductPromotionAmountOffRule) GetBase() RuleBase     { return r.RuleBase }
func (r ProductPromotionPercentageRule) GetBase() RuleBase    { return r.RuleBase }
func (r ProductPromotionFixedPriceRule) GetBase() RuleBase    { return r.RuleBase }
func (r CategoricalPromotionAllowListRule) GetBase() RuleBase { return r.RuleBase }
func (r CategoricalPromotionBlockListRule) GetBase() RuleBase { return r.RuleBase }

func MatchCardType(cardTypeIDSet sets.Set[uint], memberInfo MemberInfo) bool {
	return len(cardTypeIDSet) == 0 || cardTypeIDSet.Has(memberInfo.CardTypeID)
}

func MatchCategoryTypeID(categoryIDTuple con.CategoryIDTuple, line QuoteLine) bool {
	switch categoryIDTuple.CategoryType {
	case con.CategoryTypeLarge:
		return line.LargeCategoryID != 0 && line.LargeCategoryID == categoryIDTuple.CategoryID
	case con.CategoryTypeMedium:
		return line.MediumCategoryID != 0 && line.MediumCategoryID == categoryIDTuple.CategoryID
	case con.CategoryTypeSmall:
		return line.SmallCategoryID != 0 && line.SmallCategoryID == categoryIDTuple.CategoryID
	case con.CategoryTypeColorLabel:
		return line.ColorLabelID != 0 && line.ColorLabelID == categoryIDTuple.CategoryID
	case con.CategoryTypePromotion:
		return line.PromotionCategoryID != 0 && line.PromotionCategoryID == categoryIDTuple.CategoryID
	case con.CategoryTypeItemID:
		return line.ItemID != 0 && line.ItemID == categoryIDTuple.CategoryID
	}
	panic(fmt.Errorf("unreachable: unknown category type: %d", categoryIDTuple.CategoryType))
}

func (r CategoricalPromotionAllowListRule) IsEligible(line QuoteLine, memberInfo MemberInfo) bool {
	return MatchCategoryTypeID(r.CategoryIDTuple, line) && MatchCardType(r.CardTypeIDSet, memberInfo) && r.GetBase().Quantity.LessThanOrEqual(line.Quantity)
}

func (r CategoricalPromotionBlockListRule) IsEligible(line QuoteLine, memberInfo MemberInfo) bool {
	return MatchCategoryTypeID(r.CategoryIDTuple, line)
}

func (r ProductPromotionAmountOffRule) IsEligible(line QuoteLine, memberInfo MemberInfo) bool {
	return line.ItemID == r.ItemID && MatchCardType(r.CardTypeIDSet, memberInfo) && r.GetBase().Quantity.LessThanOrEqual(line.Quantity)
}

func (r ProductPromotionPercentageRule) IsEligible(line QuoteLine, memberInfo MemberInfo) bool {
	return line.ItemID == r.ItemID && MatchCardType(r.CardTypeIDSet, memberInfo) && r.GetBase().Quantity.LessThanOrEqual(line.Quantity)
}

func (r ProductPromotionFixedPriceRule) IsEligible(line QuoteLine, memberInfo MemberInfo) bool {
	return line.ItemID == r.ItemID && MatchCardType(r.CardTypeIDSet, memberInfo) && r.GetBase().Quantity.LessThanOrEqual(line.Quantity)
}

func (r ProductPromotionAmountOffRule) Calculate(storeItem StoreItem, quantity decimal.Decimal, unitPrice decimal.Decimal, memberDiscount decimal.Decimal) decimal.Decimal {
	discountedUnitPrice := unitPrice.Sub(r.AmountOff)
	if discountedUnitPrice.LessThan(decimal.Zero) {
		discountedUnitPrice = decimal.Zero
	}
	return discountedUnitPrice.Mul(quantity)
}

func (r ProductPromotionPercentageRule) Calculate(storeItem StoreItem, quantity decimal.Decimal, unitPrice decimal.Decimal, memberDiscount decimal.Decimal) decimal.Decimal {
	discountedUnitPrice := unitPrice.Mul(r.Percentage).Div(decimal.NewFromInt(100))
	return discountedUnitPrice.Mul(quantity)
}

func (r ProductPromotionFixedPriceRule) Calculate(storeItem StoreItem, quantity decimal.Decimal, unitPrice decimal.Decimal, memberDiscount decimal.Decimal) decimal.Decimal {
	return r.FixedPrice.Mul(quantity)
}

func (r CategoricalPromotionAllowListRule) Calculate(storeItem StoreItem, quantity decimal.Decimal, unitPrice decimal.Decimal, memberDiscount decimal.Decimal) decimal.Decimal {
	discountedUnitPrice := unitPrice.Mul(r.Percentage).Div(decimal.NewFromInt(100))
	return discountedUnitPrice.Mul(quantity)
}

func (r CategoricalPromotionBlockListRule) Calculate(storeItem StoreItem, quantity decimal.Decimal, unitPrice decimal.Decimal, memberDiscount decimal.Decimal) decimal.Decimal {
	return decimal.Zero
}
