package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_executeInstallation(t *testing.T) {
	err := executeInstallation("/testdata/BiocBaseUtils", "BiocBaseUtils")
	assert.NoError(t, err)
}

func Test_executeInstallationFromTargz(t *testing.T) {
	cases := []struct{ targz, packageName string }{
		{"testdata/targz/OrdinalLogisticBiplot_0.4.tar.gz", "OrdinalLogisticBiplot"},
		{"testdata/targz/curl_4.3.2.tar.gz", "curl"},
	}
	for _, v := range cases {
		err := executeInstallation(v.targz, v.packageName)
		assert.NoError(t, err)
	}
}

func Test_getInstalledPackagesWithVersion(t *testing.T) {
	pkgVer := getInstalledPackagesWithVersion([]string{"/usr/lib/R/site-library"})
	assert.NotEmpty(t, pkgVer)
}
