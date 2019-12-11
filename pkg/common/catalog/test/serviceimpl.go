package test

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/spiffe/spire/pkg/common/catalog"
)

func NewService() Service {
	return &testService{}
}

type testService struct {
	log hclog.Logger
	hs  HostService
}

func (s *testService) SetLogger(log hclog.Logger) {
	s.log = log.Named("serviceimpl")
}

func (s *testService) BrokerHostServices(broker catalog.HostServiceBroker) error {
	has, err := broker.GetHostService(HostServiceHostServiceClient(&s.hs))
	if err != nil {
		return err
	}
	if !has {
		s.log.Warn("Host service not available.", "hostservice", "HostService")
	}
	return nil
}

func (s *testService) CallService(ctx context.Context, req *Request) (*Response, error) {
	out := req.In
	if s.hs != nil {
		resp, err := s.hs.CallHostService(ctx, &Request{
			In: req.In,
		})
		if err != nil {
			return nil, err
		}
		out = resp.Out
	}
	return &Response{
		Out: fmt.Sprintf("service(%s)", out),
	}, nil
}
