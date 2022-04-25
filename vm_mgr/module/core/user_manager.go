/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (

	"fmt"
	"sync"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/zap"

	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"
)

const baseUid = 10000                    // user id start from base uid
const addUserFormat = "useradd -u %d %s" // add user cmd

// UserManager is linux user manager
type UserManager struct {
	userQueue *utils.FixedFIFO   // user queue, always pop oldest queue
	logger    *zap.SugaredLogger // user manager logger
	userNum   int                // total user num
}

// NewUsersManager returns user manager
func NewUsersManager() *UserManager {

	return &UserManager{
		userQueue: utils.NewFixedFIFO(config.DockerVMConfig.GetMaxUserNum()),
		logger:    logger.NewDockerLogger(logger.MODULE_USERCONTROLLER),
		userNum:   config.DockerVMConfig.GetMaxUserNum(),
	}
}

// BatchCreateUsers create new users in docker from 10000 as uid
func (u *UserManager) BatchCreateUsers() error {

	var err error
	var wg sync.WaitGroup
	batchCreateUsersThreadNum := config.DockerVMConfig.Process.MaxOriginalProcessNum // thread num for batch create users
	createUserNumPerThread := protocol.CallContractDepth + 1                         // user num per thread

	startTime := time.Now()
	createdUserNum := atomic.NewInt64(0)
	for i := 0; i < batchCreateUsersThreadNum; i++ {
		wg.Add(1)
		go func(i int) {
			for j := 0; j < createUserNumPerThread; j++ {
				id := baseUid + i*batchCreateUsersThreadNum + j
				err = u.generateNewUser(id)
				if err != nil {
					u.logger.Errorf("fail to create user [%d]", id)
				} else {
					createdUserNum.Add(1)
				}
			}
			wg.Done()
		}(i)
	}
	wg.Wait()

	u.logger.Infof("init uids succeed, time: [%s], total user num: [%s]", time.Since(startTime),
		createdUserNum.String())

	return nil
}

//  generateNewUser generate a new user of process
func (u *UserManager) generateNewUser(uid int) error {

	user := NewUser(uid)
	addUserCommand := fmt.Sprintf(addUserFormat, uid, user.UserName)

	createSuccess := false

	// it may fail to create user in centos, so add retry until it success
	for !createSuccess {
		if err := utils.RunCmd(addUserCommand); err != nil {
			u.logger.Warnf("failed to create user [%+v], err: [%s] and begin to retry", user, err)
			continue
		}

		createSuccess = true
	}
	u.logger.Debugf("create user succeed: %+v", user)

	// add created user to queue
	err := u.userQueue.Enqueue(user)
	if err != nil {
		return fmt.Errorf("fail to add created user %+v to queue, %v", user, err)
	}
	u.logger.Debugf("success add user to user queue: %+v", user)

	return nil
}

// GetAvailableUser pop user from queue header
func (u *UserManager) GetAvailableUser() (interfaces.User, error) {

	user, err := u.userQueue.DequeueOrWaitForNextElement()
	if err != nil {
		return nil, fmt.Errorf("fail to call DequeueOrWaitForNextElement, %v", err)
	}

	u.logger.Debugf("get available user: [%v]", user)
	return user.(interfaces.User), nil
}

// FreeUser add user to queue tail, user can be dequeue then
func (u *UserManager) FreeUser(user interfaces.User) error {

	err := u.userQueue.Enqueue(user)
	if err != nil {
		return fmt.Errorf("fail to enqueue user: %v", err)
	}
	u.logger.Debugf("free user: %v", user)
	return nil
}
