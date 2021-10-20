/*
	Copyright (C) BABEC. All rights reserved.
	SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/module/security"
	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/utils"
	"go.uber.org/zap"
)

type UsersManager struct {
	userQueue *utils.FixedFIFO
	logger    *zap.SugaredLogger
	userNum   int
}

// NewUsersManager new user manager
func NewUsersManager() *UsersManager {

	userNumConfig := os.Getenv(config.ENV_USER_NUM)
	userNum, err := strconv.Atoi(userNumConfig)
	if err != nil {
		userNum = 50
	}

	userQueue := utils.NewFixedFIFO(userNum)

	usersManager := &UsersManager{
		userQueue: userQueue,
		logger:    logger.NewDockerLogger(logger.MODULE_USERCONTROLLER, config.DockerLogDir),
		userNum:   userNum,
	}

	return usersManager
}

// CreateNewUsers create new users in docker from 10000 as uid
func (u *UsersManager) CreateNewUsers() error {

	var err error

	startTime := time.Now()
	const baseUid = 10000

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			for j := 0; j < u.userNum/10; j++ {
				newUserId := baseUid + i*u.userNum/10 + j
				err = u.generateNewUser(newUserId)

				if err != nil {
					u.logger.Errorf("fail to create user [%d]", newUserId)
				}
			}
			wg.Done()
		}(i)
	}

	wg.Wait()
	u.logger.Infof("init uids time: [%s]", time.Since(startTime))

	return nil
}

func (u *UsersManager) generateNewUser(newUserId int) error {

	const addUserFormat = "useradd -u %d %s"

	newUser := u.constructNewUser(newUserId)
	addUserCommand := fmt.Sprintf(addUserFormat, newUserId, newUser.UserName)

	if err := utils.RunCmd(addUserCommand); err != nil {
		u.logger.Errorf("fail to run cmd : [%s], [%s]", addUserCommand, err)
		return err
	}

	// add created user to queue
	err := u.userQueue.Enqueue(newUser)
	if err != nil {
		u.logger.Errorf("fail to add created user to queue, newUser : [%v]", newUser)
		return err
	}
	return nil
}

func (u *UsersManager) constructNewUser(userId int) *security.User {

	userName := fmt.Sprintf("u-%d", userId)
	sockPath := filepath.Join(config.DMSDir, config.DMSSockPath)

	return &security.User{
		Uid:      userId,
		Gid:      userId,
		UserName: userName,
		SockPath: sockPath,
	}
}

// GetAvailableUser pop user from queue header
func (u *UsersManager) GetAvailableUser() (*security.User, error) {

	user, err := u.userQueue.DequeueOrWaitForNextElement()
	if err != nil {
		u.logger.Errorf("fail to call DequeueOrWaitForNextElement")
		return nil, err
	}

	u.logger.Debugf("get available user: [%v]", user)
	return user.(*security.User), nil
}

// FreeUser add user to queue tail
func (u *UsersManager) FreeUser(user *security.User) error {
	err := u.userQueue.Enqueue(user)
	if err != nil {
		u.logger.Errorf("fail to call Enqueue")
		return err
	}
	u.logger.Debugf("free user: [%v]", user)
	return nil
}
