## pos 報價 分流 

ItemID 商品 銷售 退貨 提貨 退提貨
PosItemDraftID 組合商品初稿 銷售 退貨
CashierID  分銀員 
Quantity 商品 銷售 退貨 提貨 退提貨
CampaignDetailID 手動指定促銷活動 商品 銷售
PointsRedeemed 指定促銷時要花費的點數 商品 銷售

IsManualGift
ChangedUnitPrice
ManualDiscount
ManualDiscountRatio

IncomeExpensesReasonID 收支原因ID 不同於 商品、商品初稿、
                       只能 銷售 不能退貨？

MemberWarehouseID 提貨、退提貨時用到

TaxType 
LotNo 

### Frontend Shading
PromotionName
ItemName
ItemNo

OriginalUnitPrice// 原始單價
OriginalSubtotal // 原價小計
AppliedSubtotal  // 套用最低價後小計


OriginalTotalAmount // 原價總金額
TotalAmount         // 折價後總金額
PointsBalance // 點數餘額 前端不用給，後端回傳
PointsEarned  // 此交易累積賺取點數 前端不用給，後端回傳
Deposit       // 訂金 前端不用給，後端回傳
AmountOwed    // 賖欠 前端不用給，後端回傳


## Pos Head
EmployeeID // 員工id 員工變成消費者，使用員工價
MembershipID // 會員ID
CashierID
MachineID
