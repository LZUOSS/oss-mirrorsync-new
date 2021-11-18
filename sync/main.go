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
package main

import (
	"ChimataMS/scheduler"
	"ChimataMS/worker"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	progQuit := make(chan os.Signal)
	worker.LoadConfig()
	scheduler.InitScheduler()
	for {
		signal.Notify(progQuit, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-progQuit:
			{
				scheduler.StopAllSync()
				os.Exit(0)
			}
		default:
		}
	}
}
