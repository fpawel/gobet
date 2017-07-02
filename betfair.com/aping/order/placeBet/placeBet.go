package placeBet

import (
	"github.com/user/gobet/betfair.com/aping/order/placeOrders"
	"github.com/user/gobet/betfair.com/aping/order"
	"github.com/user/gobet/utils"
	"github.com/user/gobet/betfair.com/aping/order/cancelOrders"
	"github.com/user/gobet/betfair.com/aping/order/replaceOrder"
)

func PlaceBet(request *placeOrders.Request)( *order.PlaceOrderReport, error){
	if request.Size >= 4 {
		return request.PlaceSingleOrder()
	}
	request1 := request
	request1.Size = 4.
	if request.Side == "LAY" {
		request1.Price = 1.01
	} else {
		request1.Price = 1000.
	}
	order1, err := request1.PlaceSingleOrder()
	if err != nil{
		return nil, err
	}

	cancelOrderRequest := cancelOrders.Request{
		BetID : order1.BetID,
		MarketID : request.MarketID,
		User : request.User,
		SizeReduction: utils.Float64ToFixed(4 - request.Size,2),
	}

	_, err = cancelOrderRequest.CancelSingleOrder()
	if err != nil{
		return nil, err
	}

	replaceOrderRequest := replaceOrder.Request{
		BetID : order1.BetID,
		NewPrice : request.Price,
		User: request.User,
		MarketID:request.MarketID,
	}

	return replaceOrderRequest.ReplaceSingleOrder()
}
