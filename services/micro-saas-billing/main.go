package microsaasbilling

import "errors"

type Subscription struct {
	CustomerID, PriceID string
	Active              bool
}
type Usage struct {
	CustomerID, Metric string
	Quantity           int64
}
type Gateway interface {
	CreateSubscription(customer, price string) (Subscription, error)
	ReportUsage(Usage) error
}
type Service struct{ Gateway Gateway }

func (s Service) Subscribe(customer, price string) (Subscription, error) {
	if customer == "" || price == "" {
		return Subscription{}, errors.New("customer and price are required")
	}
	return s.Gateway.CreateSubscription(customer, price)
}
func (s Service) RecordUsage(usage Usage) error {
	if usage.CustomerID == "" || usage.Metric == "" || usage.Quantity <= 0 {
		return errors.New("valid usage is required")
	}
	return s.Gateway.ReportUsage(usage)
}
