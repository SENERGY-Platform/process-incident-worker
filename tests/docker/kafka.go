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

package docker

import (
	"context"
	"errors"
	"github.com/segmentio/kafka-go"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/wvanbergen/kazoo-go"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"
)

func Kafka(ctx context.Context, wg *sync.WaitGroup, zookeeperUrl string) (kafkaUrl string, err error) {
	kafkaport, err := GetFreePort()
	if err != nil {
		return kafkaUrl, err
	}
	provider, err := testcontainers.NewDockerProvider(testcontainers.DefaultNetwork("bridge"))
	if err != nil {
		return kafkaUrl, err
	}
	hostIp, err := provider.GetGatewayIP(ctx)
	if err != nil {
		return kafkaUrl, err
	}
	kafkaUrl = hostIp + ":" + strconv.Itoa(kafkaport)
	log.Println("host ip: ", hostIp)
	log.Println("host port: ", kafkaport)
	log.Println("kafkaUrl url: ", kafkaUrl)
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "bitnami/kafka:3.4.0-debian-11-r21",
			Tmpfs: map[string]string{},
			WaitingFor: wait.ForAll(
				wait.ForLog("INFO Awaiting socket connections on"),
				wait.ForListeningPort("9092/tcp"),
			),
			ExposedPorts:    []string{strconv.Itoa(kafkaport) + ":9092"},
			AlwaysPullImage: true,
			Env: map[string]string{
				"ALLOW_PLAINTEXT_LISTENER":             "yes",
				"KAFKA_LISTENERS":                      "OUTSIDE://:9092",
				"KAFKA_ADVERTISED_LISTENERS":           "OUTSIDE://" + kafkaUrl,
				"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP": "OUTSIDE:PLAINTEXT",
				"KAFKA_INTER_BROKER_LISTENER_NAME":     "OUTSIDE",
				"KAFKA_ZOOKEEPER_CONNECT":              zookeeperUrl,
			},
		},
		Started: true,
	})
	if err != nil {
		return kafkaUrl, err
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		log.Println("DEBUG: remove container kafka", c.Terminate(context.Background()))
	}()

	containerPort, err := c.MappedPort(ctx, "9092/tcp")
	if err != nil {
		return kafkaUrl, err
	}
	log.Println("KAFKA_TEST: container-port", containerPort, kafkaport)

	err = Retry(1*time.Minute, func() error {
		return tryKafkaConn(kafkaUrl)
	})
	if err != nil {
		return kafkaUrl, err
	}

	return kafkaUrl, err
}

func tryKafkaConn(kafkaUrl string) error {
	log.Println("try kafka connection to " + kafkaUrl + "...")
	conn, err := kafka.Dial("tcp", kafkaUrl)
	if err != nil {
		log.Println(err)
		return err
	}
	defer conn.Close()
	brokers, err := conn.Brokers()
	if err != nil {
		log.Println(err)
		return err
	}
	if len(brokers) == 0 {
		err = errors.New("missing brokers")
		log.Println(err)
		return err
	}
	log.Println("kafka connection ok")
	return nil
}

func Zookeeper(ctx context.Context, wg *sync.WaitGroup) (hostPort string, ipAddress string, err error) {
	log.Println("start zookeeper")
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "wurstmeister/zookeeper:latest",
			Tmpfs: map[string]string{"/opt/zookeeper-3.4.13/data": "rw"},
			WaitingFor: wait.ForAll(
				wait.ForLog("binding to port"),
				wait.ForListeningPort("2181/tcp"),
				wait.ForNop(Waitretry(1*time.Minute, func(ctx context.Context, target wait.StrategyTarget) error {
					log.Println("try zk connection...")
					zookeeper := kazoo.NewConfig()
					host, err := target.Host(ctx)
					if err != nil {
						log.Println("host", err)
						return err
					}
					port, err := target.MappedPort(ctx, "2181/tcp")
					if err != nil {
						log.Println("port", err)
						return err
					}
					zk, chroot := kazoo.ParseConnectionString(host + ":" + port.Port())
					zookeeper.Chroot = chroot
					kz, err := kazoo.NewKazoo(zk, zookeeper)
					if err != nil {
						log.Println("kazoo", err)
						return err
					}
					_, err = kz.Brokers()
					if err != nil && strings.TrimSpace(err.Error()) != strings.TrimSpace("zk: node does not exist") {
						log.Println("brokers", err)
						return err
					}
					return nil
				}))),
			ExposedPorts:    []string{"2181/tcp"},
			AlwaysPullImage: true,
		},
		Started: true,
	})
	if err != nil {
		return "", "", err
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		log.Println("DEBUG: remove container zookeeper", c.Terminate(context.Background()))
	}()

	ipAddress, err = c.ContainerIP(ctx)
	if err != nil {
		return "", "", err
	}
	temp, err := c.MappedPort(ctx, "2181/tcp")
	if err != nil {
		return "", "", err
	}
	hostPort = temp.Port()

	return hostPort, ipAddress, err
}
