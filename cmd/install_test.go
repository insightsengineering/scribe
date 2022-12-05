package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_executeInstallation(t *testing.T) {
	err := executeInstallation("/testdata/BiocBaseUtils", "BiocBaseUtils")
	assert.NoError(t, err)
}
