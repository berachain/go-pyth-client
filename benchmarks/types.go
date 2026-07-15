package benchmarks

// JSON response returned from the `v1/updates/price/{timestamp}` endpoint.
//
// revive:disable:nested-structs // needed for JSON unmarshalling.
type priceResponse struct {
	Parsed []struct {
		ID       string `json:"id"`
		Price    price  `json:"price"`
		EmaPrice price  `json:"ema_price"`
	} `json:"parsed"`
}

// revive:enable:nested-structs

// JSON formatted price returned from the `v1/updates/price/{timestamp}` endpoint.
type price struct {
	Price       string `json:"price"`
	Conf        string `json:"conf"`
	Expo        int32  `json:"expo"`
	PublishTime int64  `json:"publish_time"`
}
