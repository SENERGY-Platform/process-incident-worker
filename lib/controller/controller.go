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

package controller

import (
	"context"
	"encoding/json"
	"github.com/SENERGY-Platform/process-incident-worker/lib/configuration"
	"github.com/SENERGY-Platform/process-incident-worker/lib/interfaces"
	"github.com/SENERGY-Platform/process-incident-worker/lib/messages"
	"log"
	"runtime/debug"
)

type Controller struct {
	config  configuration.Config
	camunda interfaces.Camunda
	db      interfaces.Database
}

func New(ctx context.Context, config configuration.Config, camunda interfaces.Camunda, db interfaces.Database) *Controller {
	return &Controller{config: config, camunda: camunda, db: db}
}

type MsgVersionWrapper struct {
	MsgVersion int64 `json:"msg_version"`
}

func getMsgVersion(msg []byte) (version int64, err error) {
	wrapper := MsgVersionWrapper{}
	err = json.Unmarshal(msg, &wrapper)
	return wrapper.MsgVersion, err
}

func (this *Controller) HandleIncidentMessage(msg []byte) error {
	version, err := getMsgVersion(msg)
	if err != nil {
		log.Println("ERROR: unable to parse msg -> ignore: ", string(msg))
		debug.PrintStack()
		return nil
	}
	if version == 1 || version == 2 {
		incident := messages.Incident{}
		err = json.Unmarshal(msg, &incident)
		if err != nil {
			log.Println("ERROR: unable to parse msg -> ignore: ", string(msg))
			debug.PrintStack()
			return nil
		}
		return this.CreateIncident(incident)
	}
	if version == 3 {
		command := messages.KafkaIncidentsCommand{}
		err = json.Unmarshal(msg, &command)
		if err != nil {
			log.Println("ERROR: unable to parse msg -> ignore: ", string(msg))
			debug.PrintStack()
			return nil
		}
		if command.Command == "PUT" || command.Command == "POST" {
			if command.Incident != nil {
				command.Incident.MsgVersion = command.MsgVersion
				return this.CreateIncident(*command.Incident)
			}
		}
		if command.Command == "DELETE" {
			if command.ProcessDefinitionId != "" {
				return this.DeleteIncidentByProcessDefinitionId(command.ProcessDefinitionId)
			}
			if command.ProcessInstanceId != "" {
				return this.DeleteIncidentByProcessInstanceId(command.ProcessInstanceId)
			}
		}
	}
	return nil
}

func (this *Controller) CreateIncident(incident messages.Incident) (err error) {
	err = this.camunda.StopProcessInstance(incident.ProcessInstanceId, incident.TenantId)
	if err != nil {
		return err
	}
	name, err := this.camunda.GetProcessName(incident.ProcessDefinitionId, incident.TenantId)
	if err != nil {
		log.Println("WARNING: unable to get process name", err)
		incident.DeploymentName = incident.ProcessDefinitionId
	} else {
		incident.DeploymentName = name
	}
	return this.db.Save(incident)
}

func (this *Controller) DeleteIncidentByProcessInstanceId(id string) error {
	return this.db.DeleteByInstanceId(id)
}

func (this *Controller) DeleteIncidentByProcessDefinitionId(id string) error {
	return this.db.DeleteByDefinitionId(id)
}
