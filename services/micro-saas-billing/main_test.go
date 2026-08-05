package microsaasbilling

import "testing"

type fakeGateway struct{ usage Usage }

func (f *fakeGateway) CreateSubscription(c, p string) (Subscription, error) {
	return Subscription{c, p, true}, nil
}
func (f *fakeGateway) ReportUsage(u Usage) error { f.usage = u; return nil }
func TestBillingDelegatesValidRequests(t *testing.T) {
	g := &fakeGateway{}
	s := Service{g}
	sub, err := s.Subscribe("cus_1", "price_1")
	if err != nil || !sub.Active {
		t.Fatal(sub, err)
	}
	if err := s.RecordUsage(Usage{"cus_1", "tokens", 3}); err != nil || g.usage.Quantity != 3 {
		t.Fatal(err, g.usage)
	}
}
