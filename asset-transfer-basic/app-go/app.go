package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/gateway"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

// Define the Device struct type
type Device struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func main() {
	// Initialize Gin
	router := gin.Default()

	// Initialize a gateway connection
	log.Println("============ application-golang starts ============")

	err := os.Setenv("DISCOVERY_AS_LOCALHOST", "true")
	if err != nil {
		log.Fatalf("Error setting DISCOVERY_AS_LOCALHOST environment variable: %v", err)
	}

	walletPath := "wallet"
	// remove any existing wallet from prior runs
	os.RemoveAll(walletPath)
	wallet, err := gateway.NewFileSystemWallet(walletPath)
	if err != nil {
		log.Fatalf("Failed to create wallet: %v", err)
	}

	if !wallet.Exists("appUser") {
		err = populateWallet(wallet)
		if err != nil {
			log.Fatalf("Failed to populate wallet contents: %v", err)
		}
	}

	ccpPath := filepath.Join(
		"..",
		"..",
		"test-network",
		"organizations",
		"peerOrganizations",
		"org1.example.com",
		"connection-org1.yaml",
	)

	gw, err := gateway.Connect(
		gateway.WithConfig(config.FromFile(filepath.Clean(ccpPath))),
		gateway.WithIdentity(wallet, "appUser"),
	)
	if err != nil {
		log.Fatalf("Failed to connect to gateway: %v", err)
	}
	defer gw.Close()

	channelName := "mychannel"
	if cname := os.Getenv("CHANNEL_NAME"); cname != "" {
		channelName = cname
	}

	log.Println("--> Connecting to channel", channelName)
	network, err := gw.GetNetwork(channelName)
	if err != nil {
		log.Fatalf("Failed to get network: %v", err)
	}

	chaincodeName := "basic"
	if ccname := os.Getenv("CHAINCODE_NAME"); ccname != "" {
		chaincodeName = ccname
	}

	log.Println("--> Using chaincode", chaincodeName)
	contract := network.GetContract(chaincodeName)

	result, err := contract.SubmitTransaction("InitLedger")
	if err != nil {
		log.Fatalf("Failed to Submit transaction: %v", err)
	}
	log.Println(string(result))
	contract.SubmitTransaction("Delete", "D0")

	// Define routes
	router.POST("/register", func(c *gin.Context) {
		var requestBody struct {
			Esp32ID string `json:"esp32id"`
			Status  string `json:"Status"`
		}
		if err := c.BindJSON(&requestBody); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}

		// Submit transaction
		_, err := contract.SubmitTransaction("Register", requestBody.Esp32ID, requestBody.Status)
		if err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to submit transaction: %s", err)})
			return
		}

		c.JSON(200, gin.H{"message": "Device registered"})
	})

	router.POST("/update", func(c *gin.Context) {
		var requestBody struct {
			Esp32ID string `json:"esp32id"`
			Status  string `json:"Status"`
		}
		if err := c.BindJSON(&requestBody); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}

		// Submit transaction
		_, err := contract.SubmitTransaction("Update", requestBody.Esp32ID, requestBody.Status)
		if err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to submit transaction: %s", err)})
			return
		}

		c.JSON(200, gin.H{"message": "Device status updated"})
	})

	router.POST("/auth", func(c *gin.Context) {
		var requestBody struct {
			Esp32ID string `json:"esp32id"`
		}
		if err := c.BindJSON(&requestBody); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}

		// Submit transaction
		_, err := contract.SubmitTransaction("Auth", requestBody.Esp32ID)
		if err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("%s", err)})
			return
		}

		url := "http://139.59.41.48:18083/api/v5/authentication/password_based%3Abuilt_in_database/users"
		method := "POST"

		payload := &bytes.Buffer{}
		writer := multipart.NewWriter(payload)
		// GENERATE RANDOM USERNAME AND PASSWORD HERE
		_ = writer.WriteField("password", "kya_fayda")
		_ = writer.WriteField("user_id", "rahul")
		err = writer.Close()
		if err != nil {
			fmt.Println(err)
			return
		}

		client := &http.Client{}
		req, err := http.NewRequest(method, url, payload)

		if err != nil {
			fmt.Println(err)
			return
		}
		// PUT THIS HEADER INTO THE ENV FILE
		req.Header.Add("Authorization", "Basic MGQyNTdhNTEwMDdlNGJmMzpJSmJOdkhOOUEwbWUwQTlBd25jcHY5QU1Qc3dDWUpHSnY4dkZUdUFvaHRIdFVN")

		req.Header.Set("Content-Type", writer.FormDataContentType())
		res, err := client.Do(req)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer res.Body.Close()

		body, err := io.ReadAll(res.Body)
		if err != nil {
			fmt.Println(err)
			return
		}
		if string(body) == `{"code":"ALREADY_EXISTS","message":"User already exists"}` {
			c.JSON(200, gin.H{"message": "User already exists."})
			return
		}

		c.JSON(200, gin.H{"message": "Device authenticated"})
	})

	router.POST("/delete", func(c *gin.Context) {
		var requestBody struct {
			Esp32ID string `json:"esp32id"`
		}
		if err := c.BindJSON(&requestBody); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}

		// Submit transaction
		_, err := contract.SubmitTransaction("Delete", requestBody.Esp32ID)
		if err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to submit transaction: %s", err)})
			return
		}

		c.JSON(200, gin.H{"message": "Device deleted"})
	})

	router.GET("/getall", func(c *gin.Context) {
		result, err = contract.EvaluateTransaction("GetAll")
		if err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to submit transaction: %s", err)})
			return
		}
		var devices []Device // Assuming Device is the struct type representing a device
		if result == nil {
			c.JSON(200, gin.H{"devices": "No devices registered"})
			return
		}
		if err := json.Unmarshal(result, &devices); err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to parse result: %s", err)})
			return
		}
		c.JSON(200, gin.H{"devices": devices})
	})

	// Run the server
	if err := router.Run(":3001"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
func populateWallet(wallet *gateway.Wallet) error {
	log.Println("============ Populating wallet ============")
	credPath := filepath.Join(
		"..",
		"..",
		"test-network",
		"organizations",
		"peerOrganizations",
		"org1.example.com",
		"users",
		"User1@org1.example.com",
		"msp",
	)

	certPath := filepath.Join(credPath, "signcerts", "cert.pem")
	// read the certificate pem
	cert, err := os.ReadFile(filepath.Clean(certPath))
	if err != nil {
		return err
	}

	keyDir := filepath.Join(credPath, "keystore")
	// there's a single file in this dir containing the private key
	files, err := os.ReadDir(keyDir)
	if err != nil {
		return err
	}
	if len(files) != 1 {
		return fmt.Errorf("keystore folder should have contain one file")
	}
	keyPath := filepath.Join(keyDir, files[0].Name())
	key, err := os.ReadFile(filepath.Clean(keyPath))
	if err != nil {
		return err
	}

	identity := gateway.NewX509Identity("Org1MSP", string(cert), string(key))

	return wallet.Put("appUser", identity)
}
