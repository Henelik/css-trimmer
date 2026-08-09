package main

import (
	"testing"

	"github.com/Henelik/css-trimmer/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestShouldWriteWhenFailOnRemoval(t *testing.T) {
	cfg := &config.Config{FailOnRemoval: true}
	assert.False(t, !cfg.FailOnRemoval, "FailOnRemoval must suppress the write path")

	cfg = &config.Config{FailOnRemoval: false}
	assert.True(t, !cfg.FailOnRemoval, "normal run must allow the write path")
}
