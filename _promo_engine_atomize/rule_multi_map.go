package promo_engine_atomize

import (
	"fmt"
	"math"
	"math/big"

	con "pos-backend/pkg/constants"

	"github.com/R167/go-sets"
	"github.com/shopspring/decimal"
)

type RuleSource int

const (
	RuleSource組合特價 RuleSource = 11 // 組合特價
	RuleSource任選擇贈 RuleSource = 22
	RuleSource滿件擇贈 RuleSource = 33
	RuleSource滿額折贈 RuleSource = 44

	RuleSourceProductPromotionAmountOff     RuleSource = 61 // 商品特價 - 減價
	RuleSourceProductPromotionPercentage    RuleSource = 62 // 商品特價 - 百分比
	RuleSourceProductPromotionFixedPrice    RuleSource = 63 // 商品特價 - 固定價格
	RuleSourceCategoricalPromotionAllowList RuleSource = 71 // 類別特價 - 允許清單
	RuleSourceCategoricalPromotionBlockList RuleSource = 72 // 類別特價 - 阻擋清單
)

type RuleBase struct {
	RuleSource        RuleSource
	PromotionDetailID uint
	Title             string
	Remark            string
	CardTypeIDSet     sets.Set[uint]
	Weight            uint
	Sequence          uint
	CampaignDetailID  uint
	// 控制點數與消費累計的 Flags
	CanEarnPoints          bool
	CanDoublePoints        bool
	CanAccumulateSpend     bool
	CanMeetThreshold       bool
	QuantityDiscountMethod con.QuantityDiscountMethod

	Quantity         decimal.Decimal
	PurchaseLimit    decimal.Decimal
	ResetDays        uint
	PointsRedemption decimal.Decimal
}

func (b RuleBase) Equal(other RuleBase) bool {
	return b.RuleSource == other.RuleSource &&
		b.PromotionDetailID == other.PromotionDetailID
}

func (b RuleBase) GetAbsValue() *big.Int {
	source := new(big.Int).SetUint64(uint64(b.RuleSource))
	weight := new(big.Int).SetUint64(math.MaxUint64 - uint64(b.Weight))
	sequence := new(big.Int).SetUint64(uint64(b.Sequence))
	campaignDetailID := new(big.Int).SetUint64(uint64(b.CampaignDetailID))

	result := new(big.Int).Lsh(source, 192)
	result.Or(result, new(big.Int).Lsh(weight, 128))
	result.Or(result, new(big.Int).Lsh(sequence, 64))
	result.Or(result, campaignDetailID)

	return result
}

func (b RuleBase) HasHigherPriorityThan(other RuleBase) bool {
	myAbs := b.GetAbsValue()
	otherAbs := other.GetAbsValue()
	return myAbs.Cmp(otherAbs) > 0
}

func (b RuleBase) Key() string {
	return fmt.Sprintf("%d-%d", b.RuleSource, b.PromotionDetailID)
}

type PromotionRule interface {
	GetBase() RuleBase
	IsEligible(line QuoteLine, memberInfo MemberInfo) bool
	Calculate(storeItem StoreItem, quantity decimal.Decimal, unitPrice decimal.Decimal, memberDiscount decimal.Decimal) decimal.Decimal
}

type RuleSet map[string]PromotionRule

func (ruleMap RuleSet) MergeRuleCandidate(rule PromotionRule) RuleSet {
	key := rule.GetBase().Key()
	existing, found := ruleMap[key]
	if found && existing.GetBase().HasHigherPriorityThan(rule.GetBase()) {
		return ruleMap
	}
	ruleMap[key] = rule
	return ruleMap
}

func (ruleMap RuleSet) Add(rule PromotionRule) RuleSet {
	return ruleMap.MergeRuleCandidate(rule)
}

func (ruleMap RuleSet) GetSlice() []PromotionRule {
	result := make([]PromotionRule, 0, len(ruleMap))
	for _, rule := range ruleMap {
		result = append(result, rule)
	}
	return result
}

func NewRuleSetFromSlice(rules []PromotionRule) RuleSet {
	result := make(RuleSet)
	for _, rule := range rules {
		result[rule.GetBase().Key()] = rule
	}
	return result
}

func (ruleMap RuleSet) Has(rule PromotionRule) bool {
	_, found := ruleMap[rule.GetBase().Key()]
	return found
}

func (ruleMap RuleSet) RemoveKey(rule PromotionRule) {
	delete(ruleMap, rule.GetBase().Key())
}

type RuleMultiMap struct {
	data map[uint]RuleSet
}

func NewRuleMultiMap() *RuleMultiMap {
	return &RuleMultiMap{
		data: make(map[uint]RuleSet),
	}
}

func (m *RuleMultiMap) Put(itemID uint, rule PromotionRule) {
	if m.data[itemID] == nil {
		m.data[itemID] = make(RuleSet)
	}
	if m.data[itemID].Has(rule) {
		return
	}
	ruleSet := m.data[itemID]
	ruleSet.MergeRuleCandidate(rule)
}

func (m RuleMultiMap) Get(itemID uint) (RuleSet, bool) {
	ruleSet, found := m.data[itemID]
	if !found {
		return nil, false
	}
	return ruleSet, true
}

func (m *RuleMultiMap) ContainsKey(itemID uint) bool {
	_, found := m.data[itemID]
	return found
}

func (m *RuleMultiMap) Contains(itemID uint, rule PromotionRule) bool {
	ruleSet, found := m.data[itemID]
	if !found {
		return false
	}
	return ruleSet.Has(rule)
}

func (m *RuleMultiMap) Remove(itemID uint, rule PromotionRule) {
	ruleSet, found := m.data[itemID]
	if !found {
		return
	}

	ruleSet.RemoveKey(rule)
	if len(ruleSet) == 0 {
		delete(m.data, itemID)
	}
}

func (m *RuleMultiMap) RemoveAll(itemID uint) {
	delete(m.data, itemID)
}

func (m RuleMultiMap) Keys() []uint {
	keys := make([]uint, 0, len(m.data))
	for key := range m.data {
		keys = append(keys, key)
	}
	return keys
}

func (m RuleMultiMap) Values() []PromotionRule {
	values := make([]PromotionRule, 0)
	for _, ruleSet := range m.data {
		values = append(values, ruleSet.GetSlice()...)
	}
	return values
}

func (m *RuleMultiMap) Clear() {
	m.data = make(map[uint]RuleSet)
}
