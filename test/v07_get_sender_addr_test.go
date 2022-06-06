package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDockerGoGetSenderAddr(t *testing.T) {
	setupTest(t)
	mockTxQueryCertFromChain(mockTxContext)
	mockGetSender(mockTxContext)
	mockTxGetChainConf(mockTxContext)
	testData := []struct {
		/*
			| MemberType            | AddrType          |
			| ---                   | ---               |
			| MemberType_CERT       | AddrType_ZXL      |
			| MemberType_CERT_HASH  | AddrType_ZXL      |
			| MemberType_PUBLIC_KEY | AddrType_ZXL      |
			| MemberType_CERT       | AddrType_ETHEREUM |
			| MemberType_CERT_HASH  | AddrType_ETHEREUM |
			| MemberType_PUBLIC_KEY | AddrType_ETHEREUM |
		*/
		wantAddr string
	}{
		{zxlCertAddressFromCert},
		{zxlCertAddressFromCert},
		{zxlPKAddress},
		{cmCertAddressFromCert},
		{cmCertAddressFromCert},
		{cmPKAddress},
	}

	parameters := generateInitParams()
	parameters["method"] = []byte("get_sender_address")
	//mockContractId2 := &commonPb.Contract{
	//	Name:        ContractNameTest,
	//	Version:     "v2.0.0",
	//	RuntimeType: commonPb.RuntimeType_DOCKER_GO,
	//}
	//
	//filePath := fmt.Sprintf("./testdata/%s.7z", ContractNameTest)
	//contractBin, contractFileErr := ioutil.ReadFile(filePath)
	//if contractFileErr != nil {
	//	log.Fatal(fmt.Errorf("get byte code failed %v", contractFileErr))
	//}

	//initParameters := generateInitParams()
	//result, _ := mockRuntimeInstance.Invoke(mockContractId2, initMethod, contractBin, initParameters,
	//	mockTxContext, uint64(123))
	//if result.Code == 0 {
	//	fmt.Printf("deploy user contract successfully\n")
	//}

	for index, data := range testData {
		result, _ := mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
			parameters, mockTxContext, uint64(123))
		assert.Equal(t, uint32(0), result.GetCode())
		assert.Equal(t, data.wantAddr, string(result.GetResult()))
		t.Logf("addr[%d] : [%s]", index, result.GetResult())
	}

	tearDownTest()
}
