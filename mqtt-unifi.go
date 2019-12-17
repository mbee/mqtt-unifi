// Copyright (c) 2014 The unifi Authors. All rights reserved.
// Use of this source code is governed by ISC-style license
// that can be found in the LICENSE file.

// Example command log-roaming
// log stations of a given site as they roam
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/dim13/unifi"
	"github.com/yosssi/gmq/mqtt"
	"github.com/yosssi/gmq/mqtt/client"
)

type roaming struct {
	Mac     string `json:"mac"`
	Name    string `json:"name"`
	IP      string `json:"ip"`
	Ap      string `json:"ap"`
	Channel int    `json:"channel"`
	Essid   string `json:"essid"`
}

type roamMap map[string]roaming

var stamap roamMap

var (
	mqttURL      string
	mqttLogin    string
	mqttPassword string
	unifiHost    string
	unifiUser    string
	unifiPass    string
	unifiVersion int
	unifiPort    string
	unifiSiteID  string
	unifiDelay   time.Duration
	cli          *client.Client
	sigc         chan os.Signal
)

func parseInt(data string, def int) int {
	if data == "" {
		return def
	}
	result, err := strconv.ParseInt(data, 10, 32)
	if err != nil {
		return def
	}
	return int(result)
}

func getStringEnv(key string, def string, mandatory bool) string {
	result := os.Getenv(key)
	if result == "" && mandatory {
		log.Errorf("Environment variable[string] %s is mandatory\n", key)
		os.Exit(1)
	}
	if result == "" {
		result = def
	}
	return result
}

func getStringInt(key string, def int, mandatory bool) int {
	result := os.Getenv(key)
	if result == "" && mandatory {
		log.Errorf("Environment variable[int] %s is mandatory\n", key)
		os.Exit(1)
	}
	return parseInt(result, def)
}

func getDurationEnv(key string, def time.Duration, mandatory bool) time.Duration {
	resultString := os.Getenv(key)
	if resultString == "" && mandatory {
		log.Errorf("Environment variable[time.Duration] %s is mandatory\n", key)
		os.Exit(1)
	}
	if resultString == "" {
		return def
	}
	v, err := time.ParseDuration(resultString)
	if err != nil {
		log.Errorf("Unable to parse variable %s with type duration\n", key)
		log.Errorf("%s\n", err.Error())
		os.Exit(1)
	}
	return time.Duration(v)

}

func initVariables() {
	mqttURL = getStringEnv("MQTT_URL", "tcp://localhost:1883", false)
	mqttLogin = getStringEnv("MQTT_LOGIN", "", false)
	mqttPassword = getStringEnv("MQTT_PASSWORD", "", false)
	unifiHost = getStringEnv("UNIFI_HOST", "localhost", false)
	unifiUser = getStringEnv("UNIFI_USER", "", true)
	unifiPass = getStringEnv("UNIFI_PASS", "", true)
	unifiVersion = getStringInt("UNIFI_VERSION", 5, false)
	unifiPort = getStringEnv("UNIFI_PORT", "8443", false)
	unifiSiteID = getStringEnv("UNIFI_SITE_ID", "default", false)
	unifiDelay = getDurationEnv("UNITI_DELAY", 1*time.Second, false)
	if os.Getenv("DEBUG") == "true" || os.Getenv("DEBUG") == "True" ||
		os.Getenv("DEBUG") == "1" {
		log.SetLevel(log.DebugLevel)
	}
}

func initMqtt() {
	// Create an MQTT Client.
	cli = client.New(&client.Options{
		// Define the processing of the error handler.
		ErrorHandler: func(err error) {
			log.Errorf("Error with mqtt %s\n", err.Error())
			os.Exit(1)
		},
	})
	// Connect to the MQTT Server.
	err := cli.Connect(&client.ConnectOptions{
		Network:  "tcp",
		Address:  mqttURL,
		UserName: []byte(mqttLogin),
		Password: []byte(mqttPassword),
		ClientID: []byte("mqtt-unifi"),
	})
	if err != nil {
		log.Errorf("Unable to connect to mqtt %s\n", err.Error())
		os.Exit(1)
	}

}

// send a message
func publish(topic string, data roaming) error {
	// Publish a message.
	payload := new(bytes.Buffer)
	encoder := json.NewEncoder(payload)
	if err := encoder.Encode(data); err != nil {
		log.Error(err)
		return err
	}
	payloadString := strings.TrimSpace(fmt.Sprintf("%s", payload))

	err := cli.Publish(&client.PublishOptions{
		QoS:       mqtt.QoS0,
		TopicName: []byte(topic),
		Message:   []byte(payloadString),
	})
	if err != nil {
		log.Warn(err)
	}
	return err
}

func subscribe() {
	// Subscribe to topics.
	err := cli.Subscribe(&client.SubscribeOptions{
		SubReqs: []*client.SubReq{
			&client.SubReq{
				TopicFilter: []byte("mqtt-unifi/get/host/#"),
				QoS:         mqtt.QoS0,
				// Define the processing of the message handler.
				Handler: func(topicName, message []byte) {
					mac := strings.Split(string(topicName), "/")[3]
					client, found := stamap[mac]
					if found {
						publish(fmt.Sprintf("mqtt-unifi/status/host/%s", mac), client)
					} else {
						publish(fmt.Sprintf("mqtt-unifi/status/host/%s", mac), roaming{
							Mac: mac,
						})
					}
				},
			},
		},
	})
	if err != nil {
		log.Error("Unable to subscribe")
		log.Fatal(err)
	}
}

func loopOnUnifi() {
	u, err := unifi.Login(unifiUser, unifiPass, unifiHost, unifiPort, unifiSiteID, unifiVersion)
	if err != nil {
		log.Error("Unable to log to unifi")
		log.Fatal(err)
	}

	defer u.Logout()

	site, err := u.Site(unifiSiteID)
	if err != nil {
		log.Error("Unable to get the sites from unifi")
		log.Fatal(err)
	}

	apsmap, err := u.UAPMap(site)
	if err != nil {
		log.Error("Unable to get the UAP maps from unifi")
		log.Fatal(err)
	}

	ticker := time.NewTicker(unifiDelay)
	defer ticker.Stop()
	for range ticker.C {
		newmap := make(roamMap)
		sta, err := u.Sta(site)
		if err != nil {
			continue
		}
		for _, s := range sta {
			newmap[s.Mac] = roaming{
				Mac:     s.Mac,
				Name:    s.Name(),
				IP:      s.IP,
				Ap:      apsmap[s.ApMac].Name,
				Channel: s.Channel,
				Essid:   s.ESSID,
			}
		}
		for k, v := range newmap {
			if z, ok := stamap[k]; !ok {
				log.Debugf(" → %s appears on %s/%d %s/%s\n",
					v.Name, v.Ap, v.Channel, v.Essid, v.IP)
				publish(fmt.Sprintf("mqtt-unifi/%s/new", k), v)
			} else if z != v {
				log.Debugf(" ↔ %s roams from %s/%d %s/%s to %s/%d %s/%s\n",
					v.Name,
					z.Ap, z.Channel, z.Essid, z.IP,
					v.Ap, v.Channel, v.Essid, v.IP)
			}
			delete(stamap, k)
		}
		for k, v := range stamap {
			log.Debugf(" ← %s vanishes from %s/%d %s/%s\n",
				v.Name, v.Ap, v.Channel, v.Essid, v.IP)
			publish(fmt.Sprintf("mqtt-unifi/%s/delete", k), v)
		}
		stamap = newmap
	}

}

func main() {
	// Set up channel on which to send signal notifications.
	sigc = make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill)

	initVariables()
	initMqtt()
	log.Debugf("Init Mqtt... OK")

	defer cli.Terminate()

	go func() { loopOnUnifi() }()

	subscribe()

	<-sigc

	log.Info("Party is over")

	// Disconnect the Network Connection.
	if err := cli.Disconnect(); err != nil {
		panic(err)
	}

}
