package types

import "github/eugenix-io/logx-inf-backend/libs/db"

type PlaceOrderMsg struct {
	Order *db.OrderTable
}

type PlaceConditionalOrderMsg struct {
	Order *db.OrderTable
}

type UpdateOrderMsg struct {
	OldOrder db.OrderTable
	NewOrder db.OrderTable
}

type CancelOrdersBulkMsg struct {
	Orders *[]db.OrderTable
}

type CancelConditonalOrdersBulkMsg = CancelOrdersBulkMsg

type CancelBulkAndPlaceOrderMsg struct {
	CancelOrders *[]db.OrderTable
	PlaceOrder   *db.OrderTable
}

type CancelBulkAndPlaceBulkOrderMsg struct {
	CancelOrders *[]db.OrderTable
	PlaceOrders  *[]db.OrderTable
}
