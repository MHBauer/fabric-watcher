/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"os"
	"time"

	"github.com/hyperledger/fabric-sdk-go/pkg/client/ledger"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fab/events/client"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/pkg/errors"
)

func main() {
	sdk, err := fabsdk.New(config.FromFile("./first-network.yaml"))
	if err != nil {
		fmt.Println(errors.WithMessage(err, "failed to create SDK"))
		os.Exit(-1)
	}
	defer sdk.Close()

	user := "User1"
	org := "Org1"
	channelName := "mychannel"
	chainCodeID := "mycc"

	clientChannelContext := sdk.ChannelContext(channelName, fabsdk.WithUser(user), fabsdk.WithOrg(org))
	// client for interacting directly with the ledger
	ledger, err := ledger.New(clientChannelContext)
	if err != nil {
		fmt.Print(err)
		os.Exit(-1)
	}

	bci, err := ledger.QueryInfo()
	if err != nil {
		fmt.Print(err)
		os.Exit(-1)
	}
	fmt.Println("blockchain info:", bci)

	chContext, err := clientChannelContext()
	if err != nil {
		fmt.Printf("error getting channel context: %s", err)
	}

	/*
		eventService, err := chContext.ChannelService().EventService()
			if err != nil {
				fmt.Printf("error getting event service: %s", err)
			}
	*/
	eventService, err := chContext.ChannelService().EventService(client.WithBlockEvents())
	if err != nil {
		fmt.Printf("error getting event service: %s", err)
	}

	var breg fab.Registration
	var beventch <-chan *fab.BlockEvent
	breg, beventch, err = eventService.RegisterBlockEvent()
	if err != nil {
		fmt.Printf("Error registering for block events: %s", err)
	}
	defer eventService.Unregister(breg)

	ccreg, cceventch, err := eventService.RegisterChaincodeEvent(chainCodeID, ".*")
	if err != nil {
		fmt.Printf("Error registering for filtered block events: %s", err)
	}
	defer eventService.Unregister(ccreg)

	go func() {
		fmt.Println("receiving block events")
		for {
			event, ok := <-beventch
			if !ok {
				continue
				//test.Failf(t, "unexpected closed channel while waiting for Tx Status event")
			}
			fmt.Printf("Received block event: %#v", event)
			if event.Block == nil {
				continue
			}

			fmt.Println("New block created, number", event.Block.Header.Number)
		}
	}()

	go func() {
		fmt.Println("receiving chaincode events")
		for {
			event, ok := <-cceventch
			if !ok {
				//test.Failf(t, "unexpected closed channel while waiting for Tx Status event")
				continue
			}
			fmt.Printf("Received chaincode event: %#v", event)
			if event.ChaincodeID != chainCodeID {
				//test.Failf(t, "Expecting event for CC ID [%s] but got event for CC ID [%s]", chainCodeID, event.ChaincodeID)
				continue
			}
			if event.Payload != nil {
				//test.Failf(t, "Expecting nil payload for filtered events but got [%s]", event.Payload)
				continue
			}
			if event.SourceURL == "" {
				//test.Failf(t, "Expecting event source URL but got none")
				continue
			}
			if event.BlockNumber == 0 {
				//test.Failf(t, "Expecting non-zero block number")
				continue
			}
		}
	}()

	time.Sleep(1 * time.Hour)
}
