/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"errors"
	"fmt"
	"os/user"
	"strconv"
	"sync"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/zap"

	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/utils"
)

const (
	_baseUid          = 10000              // user id start from base uid
	_addUserFormat    = "useradd -u %d %s" // add user cmd
	_deleteUserFormat = "userdel -r %s"    // add user cmd
)

// UserManager is linux user manager
type UserManager struct {
	userQueue *utils.FixedFIFO   // user queue, always pop oldest queue
	logger    *zap.SugaredLogger // user manager logger
	userNum   int                // total user num
}

// check interface implement
var _ interfaces.UserManager = (*UserManager)(nil)

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

	origProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum // thread num for batch create users
	maxDepth := protocol.CallContractDepth + 1                            // user num per thread
	totalNum := origProcessNum * maxDepth                                 // total user num
	startTime := time.Now()
	createdUserNum := atomic.NewInt64(0)

	for i := 0; i < totalNum; i++ {
		wg.Add(1)
		go func(i int) {
			id := _baseUid + i
			err = u.generateNewUser(id)
			if err != nil {
				u.logger.Errorf("failed to create user [%d]", id)
			} else {
				createdUserNum.Add(1)
			}
			wg.Done()
		}(i)
	}
	wg.Wait()

	u.logger.Infof("init uids succeed, time: [%s], total user num: [%s]", time.Since(startTime),
		createdUserNum.String())

	return nil
}

// GetAvailableUser pop user from queue header
func (u *UserManager) GetAvailableUser() (interfaces.User, error) {

	newUser, err := u.userQueue.DequeueOrWaitForNextElement()
	if err != nil {
		return nil, fmt.Errorf("failed to call DequeueOrWaitForNextElement, %v", err)
	}

	u.logger.Debugf("get available newUser: [%v]", newUser)
	return newUser.(interfaces.User), nil
}

// FreeUser add user to queue tail, user can be dequeue then
func (u *UserManager) FreeUser(user interfaces.User) error {

	err := u.userQueue.Enqueue(user)
	if err != nil {
		return fmt.Errorf("failed to enqueue user: %v", err)
	}
	u.logger.Debugf("free user: %v", user)
	return nil
}

// ReleaseUsers release all users
func (u *UserManager) ReleaseUsers() error {

	var err error
	var wg sync.WaitGroup
	origProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum // thread num for batch create users
	maxDepth := protocol.CallContractDepth + 1                            // user num per thread
	totalNum := origProcessNum * maxDepth                                 // total user num

	for i := 0; i < totalNum; i++ {
		wg.Add(1)
		go func(i int) {
			id := _baseUid + i
			err = u.releaseUser(id)
			if err != nil {
				u.logger.Warnf("failed to delete user %v", err)
			}
			wg.Done()
		}(i)
	}
	wg.Wait()

	return nil
}

//  generateNewUser generate a new user of process
func (u *UserManager) generateNewUser(uid int) error {

	newUser := NewUser(uid)
	_, err := user.LookupId(strconv.Itoa(uid))
	if err != nil {
		if !errors.Is(err, user.UnknownUserIdError(uid)) {
			return err
		}

		addUserCommand := fmt.Sprintf(_addUserFormat, uid, newUser.UserName)

		createSuccess := false
		// it may failed to create newUser in centos, so add retry until it success
		for !createSuccess {
			if err := utils.RunCmd(addUserCommand); err != nil {
				u.logger.Debugf("failed to create user [%+v], err: [%s] and begin to retry", newUser, err)
				continue
			}
			createSuccess = true
		}
	}

	// add created newUser to queue
	err = u.userQueue.Enqueue(newUser)
	if err != nil {
		return fmt.Errorf("failed to add created user %+v to queue, %v", newUser, err)
	}
	//u.logger.Debugf("success add newUser to newUser queue: %+v", newUser)

	return nil
}

// releaseUser release user
func (u *UserManager) releaseUser(id int) error {
	newUser := NewUser(id)
	delUserCommand := fmt.Sprintf(_deleteUserFormat, newUser.UserName)
	if err := utils.RunCmd(delUserCommand); err != nil {
		return fmt.Errorf("failed to exec [%s], [%+v], %v", delUserCommand, newUser, err)
	}
	return nil
}
