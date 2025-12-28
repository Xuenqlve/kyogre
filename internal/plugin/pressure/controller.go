package pressure

import (
	"context"
	"sync"

	"github.com/xuenqlve/common/log"
	"github.com/xuenqlve/kyogre/internal/message"
)

type Controller struct {
	wg        sync.WaitGroup
	pressures map[string]Pressure
}

func NewController(pressures []Pressure) *Controller {
	pressureMap := make(map[string]Pressure)
	for _, pressure := range pressures {
		pressureMap[pressure.MessageType()] = pressure
	}
	return &Controller{
		pressures: pressureMap,
	}
}

func (c *Controller) Start(ctx context.Context, msgChan message.OutPoint) error {
	for _, pressure := range c.pressures {
		if err := pressure.Start(ctx); err != nil {
			return err
		}
	}
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case msg, ok := <-msgChan:
				if !ok {
					return
				}
				p, exist := c.pressures[msg.Type()]
				if !exist {
					log.Errorf("pressure %s not found", msg.Type())
					return
				}
				p.Execute(msg)
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (c *Controller) Close() error {
	c.wg.Wait()
	for _, pressure := range c.pressures {
		if err := pressure.Close(); err != nil {
			log.Errorf("pressure %s close error: %v", pressure.MessageType(), err)
		}
	}
	return nil
}
