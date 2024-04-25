package chaincode

import (
	"crypto/aes"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// SmartContract provides functions for managing an Asset
type SmartContract struct {
	contractapi.Contract
}

// Asset describes basic details of what makes up a simple asset
// Insert struct field in alphabetic order => to achieve determinism across languages
// golang keeps the order when marshal to json but doesn't order automatically
type Asset struct {
	ID     string `json:"ID"`
	Status string `json:"Status"`
	Key   string `json:"Key"`
}

// InitLedger adds a base set of assets to the ledger
func (s *SmartContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	assets := []Asset{
		{ID: "D0", Status: "Admin", Key: "xyz"},
	}

	for _, asset := range assets {
		assetJSON, err := json.Marshal(asset)
		if err != nil {
			return err
		}

		err = ctx.GetStub().PutState(asset.ID, assetJSON)
		if err != nil {
			return fmt.Errorf("failed to put to world state. %v", err)
		}
	}
	return nil
}

// Register issues a new device to the world state with given details.
func (s *SmartContract) Register(ctx contractapi.TransactionContextInterface, id string, status string, key string) error {
	exists, err := s.exists(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("the device %s already exists", id)
	}

	asset := Asset{
		ID:     id,
		Status: status,
		Key: key,
	}
	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, assetJSON)
}

// Auth returns the asset stored in the world state with given id.
func (s *SmartContract) Auth(ctx contractapi.TransactionContextInterface, id string, cipher string) (*Asset, error) {
	assetJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if assetJSON == nil {
		return nil, fmt.Errorf("the device %s does not exist", id)
	}

	var asset Asset
	err = json.Unmarshal(assetJSON, &asset)
	if err != nil {
		return nil, err
	}
	var key = []byte(asset.Key)
	encryptedBytes, err := hex.DecodeString(cipher)
    if err != nil {
        return nil, err
    }
 
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }
 
    // Trim any null characters used as padding
    decrypted := make([]byte, len(encryptedBytes))
    for bs, be := 0, block.BlockSize(); bs < len(encryptedBytes); bs, be = bs+be, be+be {
        block.Decrypt(decrypted[bs:be], encryptedBytes[bs:be])
    }
    decryptedText := string(decrypted)


	// converting json here

	var data map[string]string
    if err := json.Unmarshal([]byte(decryptedText), &data); err != nil {
        return nil, err
    }

	// json convert end

	if(data["id"] != asset.ID){
		return nil, fmt.Errorf("invalid device id")
	}
	if asset.Status != "active" {
		return nil, fmt.Errorf("the device %s is blacklisted", id)
	}

	return &asset, nil
}

// UpdateAsset updates an existing asset in the world state with provided parameters.
func (s *SmartContract) Update(ctx contractapi.TransactionContextInterface, id string, status string) error {
	exists, err := s.exists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the device %s does not exist", id)
	}
	ctx.GetStub().DelState(id)

	// overwriting original asset with new asset
	asset := Asset{
		ID:     id,
		Status: status,
	}
	assetJSON, err := json.Marshal(asset)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, assetJSON)
}

// DeleteAsset deletes an given asset from the world state.
func (s *SmartContract) Delete(ctx contractapi.TransactionContextInterface, id string) error {
	exists, err := s.exists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("the device %s does not exist", id)
	}

	return ctx.GetStub().DelState(id)
}

// AssetExists returns true when asset with given ID exists in world state
func (s *SmartContract) exists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	assetJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}

	return assetJSON != nil, nil
}

// GetAllAssets returns all assets found in world state
func (s *SmartContract) GetAll(ctx contractapi.TransactionContextInterface) ([]*Asset, error) {
	// range query with empty string for startKey and endKey does an
	// open-ended query of all assets in the chaincode namespace.
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var assets []*Asset
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var asset Asset
		err = json.Unmarshal(queryResponse.Value, &asset)
		if err != nil {
			return nil, err
		}
		assets = append(assets, &asset)
	}

	return assets, nil
}