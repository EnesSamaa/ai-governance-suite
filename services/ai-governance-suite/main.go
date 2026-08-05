package aigovernancesuite

import "errors"

type PolicyGateway interface {
	Authorize(agent, tool, payload string) error
}
type AuditSink interface {
	Record(agent, tool string) error
}
type Suite struct {
	Gateway PolicyGateway
	Audit   AuditSink
}

func (s Suite) Execute(agent, tool, payload string) error {
	if s.Gateway == nil || s.Audit == nil {
		return errors.New("gateway and audit sink are required")
	}
	if err := s.Gateway.Authorize(agent, tool, payload); err != nil {
		return err
	}
	return s.Audit.Record(agent, tool)
}
