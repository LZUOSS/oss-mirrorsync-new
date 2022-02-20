/*
 * Copyright (c) 2021 LZUOSS
 * ChimataMS is licensed under Mulan PSL v2.
 * You can use this software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain a copy of Mulan PSL v2 at:
 *  	http://license.coscl.org.cn/MulanPSL2
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
 * EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
 * MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
 * See the Mulan PSL v2 for more details.
 */
package tools

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
)

func handleScript(file *os.File) {
	_ = file.Close()
	_ = os.Remove(file.Name())
}

func getProcessStatus(cmd *exec.Cmd) chan bool {
	status := make(chan bool)
	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Println(err)
		}
		status <- cmd.ProcessState.Success()
	}()
	return status
}

func PromiseScript(quitNotify chan struct{}, scriptContent *string, workDir *string, err *error, successHook func(), failureHook func()) {
	var scriptFile *os.File
	scriptFile, *err = os.CreateTemp("", "*")
	if *err != nil {
		err2 := scriptFile.Close()
		*err = errors.New((*err).Error() + "\n" + err2.Error())
		return
	}
	defer handleScript(scriptFile)
	_, _ = fmt.Fprintln(scriptFile, "#!/bin/sh")
	_, _ = fmt.Fprintln(scriptFile, *scriptContent)

	ctx, cancelFunc := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "sh", scriptFile.Name())

	cmd.Env = append(cmd.Env, "PUBLIC_PATH="+*workDir)
	cmd.Dir = *workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	*err = cmd.Start()
	if *err != nil {
		failureHook()
		cancelFunc()
		return
	}
	select {
	case <-quitNotify:
		{
		}
	case success := <-getProcessStatus(cmd):
		{
			if success {
				successHook()
			} else {
				failureHook()
			}
		}
	}
	cancelFunc()
}
