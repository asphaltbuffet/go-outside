package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/asphaltbuffet/go-outside/internal/config"
)

func TestThresholdDefaultsToZeroMeaningUnset(t *testing.T) {
	cfg := config.GetDefaults()
	assert.Equal(t, 0, cfg.Threshold)
}
