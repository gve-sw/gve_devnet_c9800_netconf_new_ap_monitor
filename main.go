/*
Copyright (c) 2022 Cisco and/or its affiliates.
This software is licensed to you under the terms of the Cisco Sample
Code License, Version 1.1 (the "License"). You may obtain a copy of the
License at

	https://developer.cisco.com/docs/licenses

All use of the material herein must be in accordance with the terms of
the License. All rights not expressly granted by the License are
reserved. Unless required by applicable law or agreed to separately in
writing, software distributed under the License is distributed on an "AS
IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express
or implied.
*/
package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gve-sw/gve_devnet_c9800_netconf_new_ap_monitor/models"
	"github.com/scrapli/scrapligo/driver/netconf"
	ncopt "github.com/scrapli/scrapligo/driver/options"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var WLC_user string
var WLC_pass string
var Config models.Configuration

func main() {
	// Open configuration file
	configFile, err := os.Open("config.json")
	if err != nil {
		log.Fatalf("Error opening config file: %v", err)
	}
	// Read in config data
	configData, err := io.ReadAll(configFile)
	if err != nil {
		log.Fatalf("Error opening config file: %v", err)
	}

	err = json.Unmarshal(configData, &Config)
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	// Look up WLC User & Password environmental variables
	var ok bool
	WLC_user, ok = os.LookupEnv("WLC_USER")
	if !ok {
		log.Fatal("Please set environment variable: WLC_USER")
	}
	WLC_pass, ok = os.LookupEnv("WLC_PASSWORD")
	if !ok {
		log.Fatal("Please set environment variable: WLC_PASSWORD")
	}

	// Spin up NETCONF session to each controller
	stop := make(chan bool, len(Config.WirelessControllers))
	for _, wlc := range Config.WirelessControllers {
		go subscribeNETCONF(wlc.Name, int(wlc.Port), stop)
	}

	// Wait for termination signal
	log.Println("Listener running. Press Ctrl-C to quit.")
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig

	// Stop NETCONF subscriptions
	log.Println("Closing NETCONF sessions...")
	for range Config.WirelessControllers {
		stop <- true
	}
	// Allow time for NETCONF sessions to close
	time.Sleep(3 * time.Second)
	log.Println("All NETCONF sessions terminated.")
}

// Creates NETCONF subscription to target device. Continues to monitor
// subscription for messages until program is terminated
func subscribeNETCONF(address string, port int, stop <-chan bool) {
	// Establish NETCONF session
	ncSession := createSession(address, port)
	if ncSession == nil {
		// If session was not established, just quit
		return
	}

	// Establish NETCONF subscription
	resp, err := ncSession.EstablishPeriodicSubscription(Config.Netconf.XpathFilter, Config.Netconf.SubscriptionPeriod)
	if err != nil {
		log.Printf("Error establishing NETCONF subscription to %v: %v\n", address, err)
		return
	}

	log.Printf(" >> Established Subscription to controller at %v. Subscription ID: %v\n", address, resp.SubscriptionID)

	// Loop & check for messages
	for {
		var decodemsg models.ApNotification
		// Check for messages
		msg := ncSession.GetSubscriptionMessages(resp.SubscriptionID)
		select {
		case <-stop:
			// Stop if termination signal received
			log.Printf("NETCONF session for %v quitting...\n", address)
			ncSession.Close()
			return
		default:
			// If there are new messages from WLC
			if len(msg) > 0 {
				log.Printf("%v message Received from %v.\n", len(msg), address)
				// Pull out each MAC & append to apList for processing
				var apList []string
				for _, onemsg := range msg {
					xml.Unmarshal([]byte(onemsg), &decodemsg)
					joinedAPs := decodemsg.PushUpdate.Content.ApGlobalOperData.ApJoinStats
					log.Printf("Message from %v contains %v APs.\n", address, len(joinedAPs))
					for _, ap := range joinedAPs {
						apList = append(apList, ap.ApJoinInfo.ApEthernetMac)
					}
				}
				// Send AP MAC list for processing
				checkAPExists(address, apList)
			}
			// Sleep 1/3 of the subscription period. Example, if subscription is every 10 seconds
			// check for new message every 3 seconds
			wait := (Config.Netconf.SubscriptionPeriod / 1000) / 3
			time.Sleep(time.Duration(wait) * time.Second)
		}
	}
}

// Process list of AP MAC addresses & check if they already exist in MongoDB
// If not, insert into DB & push MAC to sendMQTT()
func checkAPExists(wlc string, apList []string) {
	// Generate timestamp for logging when AP joined
	currentTime := time.Now().Format(time.RFC3339)
	// Create new MongoDB session
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(Config.MongoConfig.Url))
	if err != nil {
		log.Printf("Failed to connect to MongoDB: %v\n", err)
	}
	defer client.Disconnect(ctx)
	// Check through incoming list of AP MACs to see if any exist in Mongo
	for _, apMAC := range apList {
		_, err = findByMac(client, ctx, "wireless", "onboarded_aps", apMAC)
		// If MAC not found, then AP is new
		if err != nil {
			if err.Error() == "mongo: no documents in result" {
				log.Printf(">> New AP MAC Found: %v\n", apMAC)
				err := sendMQTT(wlc, apMAC, currentTime)
				if err != nil {
					log.Printf("Unable to send MQTT message, AP (%v) not added to DB: %v\n", apMAC, err)
					break
				}
				// Create new Document
				doc := bson.D{
					{Key: "mac", Value: apMAC},
					{Key: "controller", Value: wlc},
					{Key: "onboard_time", Value: currentTime},
				}
				// Insert new MAC into MongoDB
				_, err = insertOne(client, ctx, "wireless", "onboarded_aps", doc)
				if err != nil {
					log.Printf("Error inserting AP MAC (%v) into MongoDB: %v\n", apMAC, err)
				}

			} else {
				log.Printf("Failed to connect to MongoDB: %v\n", err)
			}
		}
	}
}

// Sends message to configured MQTT server / topic with format {"wlc": "<WLC ADDRESS>", "mac": "<AP MAC>", {"time": "<JOIN TIME>"},}
func sendMQTT(wlc, apMAC, currentTime string) error {
	// MQTT client configuration
	var broker = Config.MQTTConfig.Broker
	var port = Config.MQTTConfig.Port
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", broker, port))
	opts.SetClientID(Config.MQTTConfig.ClientId)
	opts.SetWriteTimeout(3 * time.Second)
	opts.SetConnectTimeout(3 * time.Second)
	// Connect to MQTT broker
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	// Define MQTT message
	message := models.ApMQTTMessage{
		WLC:  wlc,
		MAC:  apMAC,
		TIME: currentTime,
	}
	messageJSON, _ := json.Marshal(message)
	// Publish to MQTT
	token := client.Publish(Config.MQTTConfig.Topic, 0, false, messageJSON)
	token.Wait()
	log.Printf(">> MAC: %v published to MQTT at %v\n", apMAC, broker)
	return nil
}

// Creates NETCONF session to target device
func createSession(address string, port int) *netconf.Driver {
	// Create new NETCONF driver with target device info
	ncSession, _ := netconf.NewDriver(
		address,
		ncopt.WithAuthNoStrictKey(),
		ncopt.WithAuthUsername(WLC_user),
		ncopt.WithAuthPassword(WLC_pass),
		ncopt.WithPort(830),
	)
	// Open NETCONF session
	err := ncSession.Open()
	if err != nil {
		log.Printf("Failed to connect to WLC at %v: %v\n", address, err.Error())
		return nil
	}

	return ncSession
}

func findByMac(client *mongo.Client, ctx context.Context, dataBase, col string, mac string) (bson.M, error) {
	var onboardedAp bson.M
	collection := client.Database(dataBase).Collection(col)
	if err := collection.FindOne(ctx, bson.M{"mac": mac}).Decode(&onboardedAp); err != nil {
		return nil, err
	}
	return onboardedAp, nil
}

func insertOne(client *mongo.Client, ctx context.Context, dataBase, col string, doc interface{}) (*mongo.InsertOneResult, error) {
	collection := client.Database(dataBase).Collection(col)
	result, err := collection.InsertOne(ctx, doc)
	return result, err
}
