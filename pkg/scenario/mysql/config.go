package mysql

type Config struct {
	WorkerCount int `mapstructure:"worker-count" yaml:"worker-count"`
}

func (c *Config) Validate() error {
	if c.WorkerCount == 0 {
		c.WorkerCount = 1
	}
	return nil
}
