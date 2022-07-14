package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	cfg, err := LoadConfig()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	// TODO: Here we can add more tests like check weather the configs are appropriately set
	// It protects against people changing the envconfig tag by mistake.
}
