package metadata

import "sync"

var defaultTemplate = NewTemplate()

type Templates struct {
	mu        sync.RWMutex
	templates map[string]GeneratorMetadata
}

type GeneratorMetadata func() any

func NewTemplate() *Templates {
	return &Templates{
		templates: make(map[string]GeneratorMetadata),
	}
}

func (t *Templates) RegisterTemplate(key string, generator GeneratorMetadata) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.templates == nil {
		t.templates = make(map[string]GeneratorMetadata)
	}
	t.templates[key] = generator
}

func (t *Templates) GetTemplateValue(key string) (bool, any) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	factory, ok := t.templates[key]
	return ok, factory()
}

func MustRegister(key string, generator GeneratorMetadata) {
	defaultTemplate.RegisterTemplate(key, generator)
}

func Template(key string) (bool, any) {
	return defaultTemplate.GetTemplateValue(key)
}
