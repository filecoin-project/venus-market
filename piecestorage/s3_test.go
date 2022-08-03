package piecestorage

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseS3Endpoint(t *testing.T) {
	Endpoit := "http://oss-cn-shanghai.aliyuncs.com/venus-market-test"
	endpoint, region, err := parseS3Endpoint(Endpoit)
	if err != nil {
		panic(err)
	}
	assert.Equal(t, "http://oss-cn-shanghai.aliyuncs.com", endpoint)
	assert.Equal(t, "oss-cn-shanghai", region)
}
