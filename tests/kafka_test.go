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
	"encoding/json"
	"github.com/SENERGY-Platform/process-incident-worker/lib/configuration"
	"github.com/SENERGY-Platform/process-incident-worker/lib/messages"
	"github.com/SENERGY-Platform/process-incident-worker/lib/source/util"
	"github.com/pkg/errors"
	"github.com/segmentio/kafka-go"
	"log"
	"os"
	"testing"
	"time"
)

func sendIncidentToKafka(t *testing.T, config configuration.Config, cmd messages.KafkaIncidentMessage) {
	broker, err := util.GetBroker(config.ZookeeperUrl)
	if err != nil {
		err = errors.WithStack(err)
		t.Fatalf("ERROR: %+v", err)
		return
	}
	if len(broker) == 0 {
		t.Fatalf("ERROR: %+v", errors.New("missing kafka broker"))
		return
	}
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:     broker,
		Topic:       config.KafkaIncidentTopic,
		MaxAttempts: 10,
		Logger:      log.New(os.Stdout, "[KAFKA-PRODUCER] ", 0),
	})

	message, err := json.Marshal(cmd)
	if err != nil {
		err = errors.WithStack(err)
		t.Fatalf("ERROR: %+v", err)
		return
	}

	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	err = writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(cmd.Id),
		Value: message,
		Time:  time.Now(),
	})

	if err != nil {
		err = errors.WithStack(err)
		t.Fatalf("ERROR: %+v", err)
		return
	}

	err = writer.Close()
	if err != nil {
		err = errors.WithStack(err)
		t.Fatalf("ERROR: %+v", err)
		return
	}
}