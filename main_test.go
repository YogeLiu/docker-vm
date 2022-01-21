package docker_go

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func TestTmp(t *testing.T) {
	processName := "contract2:1.0.0#0#5#1"

	nameList := strings.Split(processName, "#")

	crossKey := strings.Join(nameList[:3], "#")
	height, _ := strconv.Atoi(nameList[3])

	fmt.Println(crossKey)
	fmt.Println(height)
}
