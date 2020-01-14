/*
 * Copyright 2019 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tests

import (
	"context"
	"github.com/SENERGY-Platform/process-incident-worker/lib"
	"github.com/SENERGY-Platform/process-incident-worker/lib/camunda"
	"github.com/SENERGY-Platform/process-incident-worker/lib/configuration"
	"github.com/SENERGY-Platform/process-incident-worker/lib/database"
	"github.com/SENERGY-Platform/process-incident-worker/lib/messages"
	"github.com/SENERGY-Platform/process-incident-worker/lib/source"
	"github.com/SENERGY-Platform/process-incident-worker/tests/server"
	"testing"
	"time"
)

func TestDatabase(t *testing.T) {
	defaultConfig, err := configuration.LoadConfig("../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	defaultConfig.Debug = true

	ctx, cancel := context.WithCancel(context.Background())
	defer time.Sleep(10 * time.Second) //wait for docker cleanup
	defer cancel()

	config, err := server.New(ctx, defaultConfig)
	if err != nil {
		t.Error(err)
		return
	}

	err = lib.StartWith(ctx, config, source.Factory, camunda.Factory, database.Factory, func(err error) {
		t.Errorf("ERROR: %+v", err)
	})
	if err != nil {
		t.Error(err)
		return
	}

	incident := messages.KafkaIncidentMessage{
		Id:                  "foo_id",
		MsgVersion:          2,
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid",
		ProcessDefinitionId: "pdid",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Now(),
		DeploymentName:      "pdid",
	}

	t.Run("send incident", func(t *testing.T) {
		sendIncidentToKafka(t, config, incident)
	})

	time.Sleep(10 * time.Second)

	t.Run("check database", func(t *testing.T) {
		checkIncidentInDatabase(t, config, incident)
	})
}

func TestCamunda(t *testing.T) {
	defaultConfig, err := configuration.LoadConfig("../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	defaultConfig.Debug = true

	ctx, cancel := context.WithCancel(context.Background())
	defer time.Sleep(10 * time.Second) //wait for docker cleanup
	defer cancel()

	config, err := server.New(ctx, defaultConfig)
	if err != nil {
		t.Error(err)
		return
	}

	err = lib.StartWith(ctx, config, source.Factory, camunda.Factory, database.Factory, func(err error) {
		t.Errorf("ERROR: %+v", err)
	})
	if err != nil {
		t.Error(err)
		return
	}

	definitionId := ""
	t.Run("deploy process", func(t *testing.T) {
		definitionId = deployProcess(t, config)
	})

	time.Sleep(10 * time.Second)

	instanceId := ""
	t.Run("start process", func(t *testing.T) {
		instanceId = startProcess(t, config, definitionId)
	})

	t.Run("check process", func(t *testing.T) {
		checkProcess(t, config, instanceId, true)
	})

	incident := messages.KafkaIncidentMessage{
		Id:                  "foo_id",
		MsgVersion:          2,
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   instanceId,
		ProcessDefinitionId: definitionId,
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Now(),
	}

	t.Run("send incident", func(t *testing.T) {
		sendIncidentToKafka(t, config, incident)
	})

	time.Sleep(10 * time.Second)

	incident.DeploymentName = "test"
	t.Run("check database", func(t *testing.T) {
		checkIncidentInDatabase(t, config, incident)
	})

	t.Run("check process", func(t *testing.T) {
		checkProcess(t, config, instanceId, false)
	})

}

func TestDeleteByDeploymentId(t *testing.T) {
	defaultConfig, err := configuration.LoadConfig("../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	defaultConfig.Debug = true

	ctx, cancel := context.WithCancel(context.Background())
	defer time.Sleep(10 * time.Second) //wait for docker cleanup
	defer cancel()

	config, err := server.New(ctx, defaultConfig)
	if err != nil {
		t.Error(err)
		return
	}

	err = lib.StartWith(ctx, config, source.Factory, camunda.Factory, database.Factory, func(err error) {
		t.Errorf("ERROR: %+v", err)
	})
	if err != nil {
		t.Error(err)
		return
	}

	incident11 := messages.KafkaIncidentMessage{
		Id:                  "a",
		MsgVersion:          2,
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid1",
		ProcessDefinitionId: "pdid1",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Time{},
		DeploymentName:      "pdid1",
	}
	incident12 := messages.KafkaIncidentMessage{
		Id:                  "b",
		MsgVersion:          2,
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid1",
		ProcessDefinitionId: "pdid2",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Time{},
		DeploymentName:      "pdid2",
	}
	incident21 := messages.KafkaIncidentMessage{
		Id:                  "c",
		MsgVersion:          2,
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid2",
		ProcessDefinitionId: "pdid1",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Time{},
		DeploymentName:      "pdid1",
	}
	incident22 := messages.KafkaIncidentMessage{
		Id:                  "d",
		MsgVersion:          2,
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid2",
		ProcessDefinitionId: "pdid2",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Time{},
		DeploymentName:      "pdid2",
	}

	t.Run("send incidents", func(t *testing.T) {
		sendIncidentToKafka(t, config, incident11)
		sendIncidentToKafka(t, config, incident12)
		sendIncidentToKafka(t, config, incident21)
		sendIncidentToKafka(t, config, incident22)
	})

	time.Sleep(10 * time.Second)

	t.Run("send delete by deplymentId", func(t *testing.T) {
		sendDefinitionDeleteToKafka(t, config, "pdid1")
	})

	time.Sleep(10 * time.Second)

	t.Run("check database", func(t *testing.T) {
		checkIncidentsInDatabase(t, config, incident12, incident22)
	})
}

func TestDeleteByInstanceId(t *testing.T) {
	defaultConfig, err := configuration.LoadConfig("../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	defaultConfig.Debug = true

	ctx, cancel := context.WithCancel(context.Background())
	defer time.Sleep(10 * time.Second) //wait for docker cleanup
	defer cancel()

	config, err := server.New(ctx, defaultConfig)
	if err != nil {
		t.Error(err)
		return
	}

	err = lib.StartWith(ctx, config, source.Factory, camunda.Factory, database.Factory, func(err error) {
		t.Errorf("ERROR: %+v", err)
	})
	if err != nil {
		t.Error(err)
		return
	}

	incident11 := messages.KafkaIncidentMessage{
		Id:                  "a",
		MsgVersion:          2,
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid1",
		ProcessDefinitionId: "pdid1",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Time{},
		DeploymentName:      "pdid1",
	}
	incident12 := messages.KafkaIncidentMessage{
		Id:                  "b",
		MsgVersion:          2,
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid1",
		ProcessDefinitionId: "pdid2",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Time{},
		DeploymentName:      "pdid2",
	}
	incident21 := messages.KafkaIncidentMessage{
		Id:                  "c",
		MsgVersion:          2,
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid2",
		ProcessDefinitionId: "pdid1",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Time{},
		DeploymentName:      "pdid1",
	}
	incident22 := messages.KafkaIncidentMessage{
		Id:                  "d",
		MsgVersion:          2,
		ExternalTaskId:      "task_id",
		ProcessInstanceId:   "piid2",
		ProcessDefinitionId: "pdid2",
		WorkerId:            "w",
		ErrorMessage:        "error message",
		Time:                time.Time{},
		DeploymentName:      "pdid2",
	}

	t.Run("send incidents", func(t *testing.T) {
		sendIncidentToKafka(t, config, incident11)
		sendIncidentToKafka(t, config, incident12)
		sendIncidentToKafka(t, config, incident21)
		sendIncidentToKafka(t, config, incident22)
	})

	time.Sleep(10 * time.Second)

	t.Run("send delete by instance", func(t *testing.T) {
		sendInstanceDeleteToKafka(t, config, "piid1")
	})

	time.Sleep(10 * time.Second)

	t.Run("check database", func(t *testing.T) {
		checkIncidentsInDatabase(t, config, incident21, incident22)
	})
}
