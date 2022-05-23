package test

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"strconv"
	"sync"
	"testing"
	"time"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"

	"chainmaker.org/chainmaker/logger/v2"
	docker_go "chainmaker.org/chainmaker/vm-docker-go/v2"
)

const (
	performContractVersion2 = "1.0.1"
)

func TestMultiContract(t *testing.T) {

	//step1: get chainmaker configuration setting from mocked data
	fmt.Printf("=== step 1 load mocked chainmaker configuration file ===\n")
	cmConfig, err := getMockedCMConfig()
	if err != nil {
		log.Fatalf("get the mocked chainmaker configuration failed %v\n", err)
	}

	//step2: generate a docker manager instance
	fmt.Printf("=== step 2 Create docker instance ===\n")
	mockDockerManager = docker_go.NewDockerManager(chainId, cmConfig)

	//step3: start docker VM
	fmt.Printf("=== step 3 start Docker VM ===\n")
	dockerContainErr := mockDockerManager.StartVM()
	if dockerContainErr != nil {
		log.Fatalf("start docmer manager instance failed %v\n", dockerContainErr)
	}

	//step4: mock sim context
	fmt.Printf("===step 4 Mock txContext====\n")
	performTxContext = InitContextTest()
	mockLogger = logger.GetLogger(logger.MODULE_VM)

	testDeployCutVersion(performContractVersion)
	testDeployCutVersion(performContractVersion2)
	time.Sleep(10 * time.Second)

	//testCutSave()
	//testDeployZXL()

	performMultiContractTxs(loopNum, threadNum, []string{performContractVersion, performContractVersion2})

	time.Sleep(5 * time.Second)
	fmt.Println("tear down")
	tearDownTest()
}

func performMultiContractTxs(loopNum, threadNum int, versions []string) {
	fmt.Println("--------- Ready to analysis --------------")
	fmt.Println("---------- Start -------------------------")

	var totalTps float64

	// declare a slice to store group exec info
	groupResultList := make([]*GroupRunInfo, 0, loopNum)

	for loopIndex := 0; loopIndex < loopNum; loopIndex++ {

		// create a groupRunInfo instance
		groupRunInfo := &GroupRunInfo{
			groupName: "#group" + strconv.Itoa(loopIndex),
			startTime: math.MaxInt64,
			endTime:   math.MinInt64,
			taskNum:   int64(threadNum),
		}

		// create startTimeCh  endTimeCh
		startTimeCh := make(chan int64, threadNum)
		endTimeCh := make(chan int64, threadNum)

		wg := sync.WaitGroup{}

		for threadIndex := 0; threadIndex < threadNum; threadIndex++ {

			wg.Add(1)

			go func(i int) {

				startTimeCh <- time.Now().UnixNano()

				//testCutFindByHash()
				testCutSaveVersion(versions[loopIndex%len(versions)])
				//testZXLAddPoint()

				endTimeCh <- time.Now().UnixNano()

				wg.Done()

			}(threadIndex)
		}

		wg.Wait()
		// close all channel
		close(startTimeCh)
		close(endTimeCh)

		// get start time and end time

		// for range startTimeCh (min start time)
		for timeStamp := range startTimeCh {
			if timeStamp < groupRunInfo.startTime {
				groupRunInfo.startTime = timeStamp
			}
		}

		// for range endTimeCh (max end time)
		for timeStamp := range endTimeCh {
			if timeStamp > groupRunInfo.endTime {
				groupRunInfo.endTime = timeStamp
			}
		}

		// compute tps
		groupRunInfo.runtime = float64(groupRunInfo.endTime-groupRunInfo.startTime) / 1e9
		groupRunInfo.tps = cutOutFloat64(float64(groupRunInfo.taskNum)/groupRunInfo.runtime, 2)

		totalTps += groupRunInfo.tps

		fmt.Printf("finished group[%d]: tps:%v\n", loopIndex, groupRunInfo.tps)

		groupResultList = append(groupResultList, groupRunInfo)

	}

	fmt.Println("--------- Finished analysis --------------")
	averageTps := totalTps / float64(loopNum)
	fmt.Println("total tps: ", averageTps)

	//resultFileName := fmt.Sprintf("./result/%s-%d-%d_analysis_result.log", performContractName, loopNum, threadNum)
	//chartFileName := fmt.Sprintf("./result/%s-%d-%d_analysis_chart.html", performContractName, loopNum, threadNum)
	//PrintGroupRunResultAndDraw("result", groupResultList, resultFileName, chartFileName)
	//time.Sleep(3 * time.Second)
}

// ============================= contract_cut functions ===============================

func testDeployCutVersion(version string) {

	fmt.Printf("=== step 6 deploy 【%s】 ===\n", performContractName)

	filePath := fmt.Sprintf("./testdata/%s.7z", performContractName)
	contractBin, contractFileErr := ioutil.ReadFile(filePath)
	if contractFileErr != nil {
		log.Fatal(fmt.Errorf("get byte code failed %v", contractFileErr))
	}

	newRuntimeInstance, _ := mockDockerManager.NewRuntimeInstance(nil, chainId, "",
		"", nil, nil, mockLogger)

	newContractId := &commonPb.Contract{
		Name:        performContractName,
		Version:     version,
		RuntimeType: commonPb.RuntimeType_DOCKER_GO,
	}

	parameters := generateInitParams()
	result, _ := newRuntimeInstance.Invoke(newContractId, initMethod, contractBin, parameters,
		performTxContext, uint64(123))
	if result.Code == 0 {
		fmt.Printf("deploy user contract successfully\n")
	}

}

func testCutSaveVersion(version string) {
	newRuntimeInstance, _ := mockDockerManager.NewRuntimeInstance(nil, chainId, "",
		"", nil, nil, mockLogger)

	newContractId := &commonPb.Contract{
		Name:        performContractName,
		Version:     version,
		RuntimeType: commonPb.RuntimeType_DOCKER_GO,
	}

	parameters := generateInitParams()
	parameters["method"] = []byte("save")
	parameters["file_key"] = []byte("key")
	parameters["file_name"] = []byte("name")

	newRuntimeInstance.Invoke(newContractId, invokeMethod, nil, parameters,
		performTxContext, uint64(123))
}
