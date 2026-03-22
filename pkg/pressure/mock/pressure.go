package mock

import (
	"context"
	"sync"

	"github.com/mitchellh/mapstructure"
	"github.com/xuenqlve/common/errors"
	"github.com/xuenqlve/common/log"
	"github.com/xuenqlve/kyogre/internal/message"
	"github.com/xuenqlve/kyogre/internal/plugin/pressure"
	message2 "github.com/xuenqlve/kyogre/pkg/message"
)

const PressureType pressure.PressureType = "mock"

type Config struct {
	Name string `mapstructure:"name"`
}

type Pressure struct {
	cfg Config
	ctx context.Context
	mu  sync.Mutex
}

func init() {
	pressure.RegisterPressure(PressureType, &Pressure{}, false)
}

func (p *Pressure) Configure(pipeline string, data map[string]any) error {
	if err := mapstructure.Decode(data, &p.cfg); err != nil {
		return errors.Trace(err)
	}
	if p.cfg.Name == "" {
		p.cfg.Name = pipeline
	}
	return nil
}

func (p *Pressure) MessageType() string {
	return message2.MockType
}

func (p *Pressure) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ctx = ctx
	return nil
}

func (p *Pressure) Execute(msg message.Message) {
	if msg == nil {
		log.Infof("[%s] pipeline idle message received", p.cfg.Name)
		return
	}
	log.Infof("[%s] recv message type=%s value=%v\n", p.cfg.Name, msg.Type(), msg)
}

func (p *Pressure) Close() error {
	return nil
}
