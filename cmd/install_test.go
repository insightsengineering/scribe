package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_executeInstallation(t *testing.T) {
	t.Skip("skipping integration test")
	err := executeInstallation("/testdata/BiocBaseUtils", "BiocBaseUtils", "test.out")
	assert.NoError(t, err)
}

func Test_executeInstallationFromTargz(t *testing.T) {
	t.Skip("skipping integration test")
	cases := []struct{ targz, packageName string }{
		{"testdata/targz/OrdinalLogisticBiplot_0.4.tar.gz", "OrdinalLogisticBiplot"},
		{"testdata/targz/curl_4.3.2.tar.gz", "curl"},
		{"testdata/targz/bitops_1.0-7.tar.gz", "bitops"},
		{"testdata/targz/CompQuadForm_1.4.3.tar.gz", "CompQuadForm"},
		{"testdata/targz/dotCall64_1.0-1.tar.gz", "dotCall64"},
		{"testdata/targz/tripack_1.3-9.1.tar.gz", "tripack"},
	}
	for _, v := range cases {
		err := executeInstallation(v.targz, v.packageName, v.packageName+".out")
		assert.NoError(t, err)
	}
}

func Test_getInstalledPackagesWithVersion(t *testing.T) {
	pkgVer := getInstalledPackagesWithVersion([]string{"/usr/lib/R/site-library"})
	assert.NotEmpty(t, pkgVer)
}
