package pbft

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"log"
	"net/rpc"
	"time"
)

var servers = [...]string{"18.232.86.210", "54.164.151.89", "52.207.222.204", "52.91.154.255"}

type Client struct{}

func (c *Client) callCommand(server string) {

	command := Command{
		ClientAddress: "18.206.100.184",
		CommandType:   "foo",
		Timestamp:     time.Now(),
		ClientID:      0,
	}

	client, err := rpc.DialHTTP("tcp", server+":1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}

	pbft := new(PBFT)

	client.Go("PBFT.Start", command, pbft, nil)

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

	//Now that the keys are generated, send them to the servers
	for i := 0; i < len(servers); i++ {
		args := &MakeArgs{
			privateKeys[i],
			publicKeys,
			servers,
			i,
		}

		client, err := rpc.DialHTTP("tcp", servers[i]+":1234")
		if err != nil {
			log.Fatal("dialing:", err)
		}

		client.Call("PBFT.Make", args, nil)

	}

}
