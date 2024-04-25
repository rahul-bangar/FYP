package main

import (
	"bytes"
	"crypto/aes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/gateway"
	"github.com/joho/godotenv"
)

// Define the Device struct type
type Device struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Key    string `json:"key"`
}
type Device_list struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}
type User struct {
	Name     string `json:"username"`
	Password string `json:"password"`
}

type Res_body struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

var result []byte
var err error

func usr_name() string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	// Create a new random number generator
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)

	length := r.Intn(6) + 6 // Generates a random number between 6 and 10
	b := make([]rune, length)
	for i := range b {
		b[i] = letters[r.Intn(len(letters))]
	}
	return string(b)
}
func pwd() string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	// Create a new random number generator
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)

	length := 10 // Set the length to 10
	b := make([]rune, length)
	for i := range b {
		b[i] = letters[r.Intn(len(letters))]
	}
	return string(b)
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
	router.POST("/register", register(contract))

	router.POST("/update", update(contract))

	router.POST("/auth", auth(contract))

	router.POST("/delete", delete(contract))

	router.GET("/getall", GetAll(contract))

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

func register(contract *gateway.Contract) gin.HandlerFunc {
	return func(c *gin.Context) {
		var requestBody struct {
			Esp32ID string `json:"esp32id"`
			Status  string `json:"Status"`
			Key     string `json:"key"`
		}
		if err := c.BindJSON(&requestBody); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}

		if len(requestBody.Key) < 16 {
			// Pad the Key with spaces to make it 16 characters
			requestBody.Key = requestBody.Key + strings.Repeat(" ", 16-len(requestBody.Key))
		}

		// Submit transaction
		_, err := contract.SubmitTransaction("Register", requestBody.Esp32ID, requestBody.Status, requestBody.Key)
		if err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to submit transaction: %s", err)})
			return
		}

		c.JSON(200, gin.H{"message": "Device registered"})
	}
}

func update(contract *gateway.Contract) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

func auth(contract *gateway.Contract) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Load .env file
		er := godotenv.Load(".env")
		if er != nil {
			log.Fatalf("Error loading .env file: %v", er)
		}
		Key := os.Getenv("KEY")
		var requestBody struct {
			Esp32ID string `json:"esp32id"`
			Cipher  string `json:"cipher"`
		}
		if err := c.BindJSON(&requestBody); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request body"})
			return
		}

		// Submit transaction
		asset, err := contract.SubmitTransaction("Auth", requestBody.Esp32ID)

		if err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("%s", err)})
			return
		}
		fmt.Println(string(asset))
		m := make(map[string]string)
		err = json.Unmarshal(asset, &m)
		if err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("%s", err)})
			return
		}
		var key = m["Key"]
		fmt.Printf("Fetched Key is: %s\n", key)

		encryptedBytes, err := hex.DecodeString(requestBody.Cipher)
		if err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("%s", err)})
			return
		}

		block, err := aes.NewCipher([]byte(key))
		if err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("%s", err)})
			return
		}

		// Trim any null characters used as padding
		decrypted := make([]byte, len(encryptedBytes))
		for bs, be := 0, block.BlockSize(); bs < len(encryptedBytes); bs, be = bs+be, be+be {
			block.Decrypt(decrypted[bs:be], encryptedBytes[bs:be])
		}
		decryptedText := string(decrypted)

		fmt.Printf("Decrypted: %s\n", decryptedText)

		// converting json here

		var data map[string]string
		if err := json.Unmarshal([]byte(decryptedText), &data); err != nil {
			return
		}

		// json convert end

		fmt.Printf("Data: %v, ID: %s\n", data, data["id"])

		if data["id"] != m["ID"] {
			c.JSON(500, gin.H{"error": "Some error occurred"})
			return
		}

		url := "http://159.89.173.20:18083/api/v5/authentication/password_based%3Abuilt_in_database/users"
		method := "POST"

		payload := &bytes.Buffer{}
		writer := multipart.NewWriter(payload)
		// GENERATE RANDOM USERNAME AND PASSWORD HERE
		b := usr_name()
		x := pwd()
		fmt.Println("username = " + string(b))
		fmt.Println("password = " + string(x))
		_ = writer.WriteField("password", x)
		_ = writer.WriteField("user_id", b)
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

		req.Header.Add("Authorization", Key)

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
		var Res Res_body
		err = json.Unmarshal(body, &Res)
		if err != nil {
			fmt.Println("Error parsing JSON:", err)
			return
		}
		if res.StatusCode == 409 {
			c.JSON(409, gin.H{"error": Res.Message})
			return
		}
		if res.StatusCode == 201 {
			user := User{
				Name:     b,
				Password: x,
			}
			c.JSON(201, user)
			return
		}

		c.JSON(res.StatusCode, gin.H{"error": "Some error occurred"})
	}
}

func delete(contract *gateway.Contract) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

func GetAll(contract *gateway.Contract) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err = contract.EvaluateTransaction("GetAll")
		if err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to submit transaction: %s", err)})
			return
		}
		var devices []Device_list
		if result == nil {
			c.JSON(200, gin.H{"devices": "No devices registered"})
			return
		}
		if err := json.Unmarshal(result, &devices); err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to parse result: %s", err)})
			return
		}
		c.JSON(200, gin.H{"devices": devices})
	}
}
