package utils

import "strings"

func SplitContractName(contractNameAndVersion string) string {
	contractName := strings.Split(contractNameAndVersion, "#")[0]
	return contractName
}
