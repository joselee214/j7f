package service_register

import (
	"context"
	"time"
)

const minHeartBeatTime = 500 * time.Millisecond

type RegisterOpts struct {
	RegisterData *Service
	RegisterFunc RegisterFunc
}

type RegisterFunc interface {
	Register(ctx context.Context, s *Service) error
	DeRegister(ctx context.Context, s *Service) error
}

func NewRegisterOpts(s *Service, registerFunc RegisterFunc) *RegisterOpts {
	//var nr *RegisterOpts = new(RegisterOpts)
	//nr.RegisterData = s
	//nr.RegisterFunc = registerFunc
	//return nr
	return &RegisterOpts{
		RegisterData: s,
		RegisterFunc: registerFunc,
	}
}

func (r *RegisterOpts) Register() error {
	return r.RegisterFunc.Register(context.Background(), r.RegisterData)
}

func (r *RegisterOpts) DeRegister() error {
	return r.RegisterFunc.DeRegister(context.Background(), r.RegisterData)
}
