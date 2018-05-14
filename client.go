package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"time"
	"fmt"
)

var servers = [...]string{"18.232.86.210", "54.164.151.89", "52.207.222.204", "52.91.154.255"}
var serverClient []*rpc.Client

type Client struct {
	resultChannel chan int64
}

func (c *Client) ReceiveReply(args CommandReply, reply *RPCReply) error {

	timeDifference := time.Now().Sub(args.RequestTimestamp)
	c.resultChannel <- timeDifference.Nanoseconds()
	return nil
}

func (c *Client) callCommand(server int) {

	command := Command{
		ClientAddress: "18.206.100.184",
		CommandType:   "foo",
		Timestamp:     time.Now(),
		ClientID:      0,
	}

	/*client, err := rpc.DialHTTP("tcp", server+":1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}*/

	//pbft := new(PBFT)

	serverClient[server].Go("PBFT.Start", command, nil, nil)

}

/*
 * Bootstraps the response servers by calling the make function and giving them the private and public keys
 *
 */
func Bootstrap() {
	curve := elliptic.P256()

	publicKeys := []ecdsa.PublicKey{}
	privateKeys := []ecdsa.PrivateKey{}

	//Generates the public and private key pairs for each server
	for i := 0; i < len(servers); i++ {
		privateKey, error := ecdsa.GenerateKey(curve, rand.Reader)
		if error != nil {
			log.Print("[Bootstrap] Failed to generate public key")
		}

		publicKey := privateKey.PublicKey

		publicKeys = append(publicKeys, publicKey)
		privateKeys = append(privateKeys, *privateKey)

	}
	log.Print("Generated public keys\n")

	//Now that the keys are generated, send them to the servers
	for i := 0; i < len(servers); i++ {
		args := &MakeArgs{
			servers,
			i,
		}

		client, err := rpc.DialHTTP("tcp", servers[i]+":1234")
		if err != nil {
			log.Fatal("dialing:", err)
		}

		serverClient = append(serverClient, client)

		callDone := client.Go("PBFT.Make", args, nil, nil)

		replyCall := <-callDone.Done
		fmt.Print(" Done with the RPC call\n")
		fmt.Print(replyCall)
		fmt.Println()

		fmt.Print(" Print the error for the call\n")
		fmt.Print(callDone.Error)
		fmt.Println()

	}
	log.Print("Generated private keys\n")

	// client should make it's RPC server as well
	client := &Client{resultChannel: make(chan int64, 1000)}
	rpc.Register(client)
	log.Print("Registered client\n")

	rpc.HandleHTTP()
	log.Print("Handled HTTP\n")

	l, e := net.Listen("tcp", ":1234")
	log.Print("Listened\n")
	if e != nil {
		log.Fatal("listen error:", e)
	}
	log.Print("Checked error\n")
	go http.Serve(l, nil)

	log.Print("Serving server\n")

	// the first command to the servers
	const T = 1000
	timer := time.NewTimer(time.Millisecond * + T)
	timer.Reset(time.Millisecond * time.Duration(T))

	fileHandler, err1 := os.OpenFile("pbft_latency_results.txt", os.O_APPEND|os.O_WRONLY, 0644)
	if err1 != nil {
		log.Fatal(err1)
	}

	defer fileHandler.Close()

	log.Print("About to do the for\n")


	for {
		select {

		case <-timer.C:
			log.Print("time went off \n")
			/*if !timer.Stop() {
				<-timer.C
			}*/
			timer.Reset(time.Millisecond * time.Duration(T))

		case commandDuration := <-client.resultChannel:
			print("got something")
			/*if !timer.Stop() {
				<-timer.C
			}*/
			fmt.Printf("Command duration: %s", commandDuration)
			_, err3 := fileHandler.WriteString(string(commandDuration))
			if err3 != nil {
				log.Fatal(err3)
			}

			//timer.Reset(time.Millisecond * time.Duration(T))
		}

		client.callCommand(0)
		log.Print("Clint sent another to servers\n")
		//timer.Reset(time.Millisecond * time.Duration(T))
	}
}
