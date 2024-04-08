const express = require("express");

const app = express();
app.use(express.json());

const PORT = process.env.PORT || 3001;

app.listen(PORT, () => {
  init();
  console.log("Server Listening on PORT:", PORT);
});

const { Gateway, Wallets } = require("fabric-network");
const FabricCAServices = require("fabric-ca-client");
const path = require("path");
const {
  buildCAClient,
  registerAndEnrollUser,
  enrollAdmin,
} = require("../../test-application/javascript/CAUtil.js");
const {
  buildCCPOrg1,
  buildWallet,
} = require("../../test-application/javascript/AppUtil.js");

const channelName = "mychannel"; //process.env.CHANNEL_NAME || "mychannel";
const chaincodeName = process.env.CHAINCODE_NAME || "basic";

const mspOrg1 = "Org1MSP";
const walletPath = path.join(__dirname, "wallet");
const org1UserId = "javascriptAppUser";

function prettyJSONString(inputString) {
  return JSON.stringify(JSON.parse(inputString), null, 2);
}

let ccp = null;
let caClient = null;
let wallet = null;
let gateway = null;
let network = null;
let contract = null;

async function init() {
  try {
    // build an in memory object with the network configuration (also known as a connection profile)
    ccp = buildCCPOrg1();

    // build an instance of the fabric ca services client based on
    // the information in the network configuration
    caClient = buildCAClient(FabricCAServices, ccp, "ca.org1.example.com");

    // setup the wallet to hold the credentials of the application user
    wallet = await buildWallet(Wallets, walletPath);

    // in a real application this would be done on an administrative flow, and only once
    await enrollAdmin(caClient, wallet, mspOrg1);

    // in a real application this would be done only when a new user was required to be added
    // and would be part of an administrative flow

    await registerAndEnrollUser(
      caClient,
      wallet,
      mspOrg1,
      org1UserId,
      "org1.department1"
    );

    // Create a new gateway instance for interacting with the fabric network.
    // In a real application this would be done as the backend server session is setup for
    // a user that has been verified.
    gateway = new Gateway();

    try {
      // setup the gateway instance
      // The user will now be able to create connections to the fabric network and be able to
      // submit transactions and query. All transactions submitted by this gateway will be
      // signed by this user using the credentials stored in the wallet.

      await gateway.connect(ccp, {
        wallet,
        identity: org1UserId,
        discovery: { enabled: true, asLocalhost: true }, // using asLocalhost as this gateway is using a fabric network deployed locally
      });


      // Build a network instance based on the channel where the smart contract is deployed
      network = await gateway.getNetwork(channelName);

      
      // Get the contract from the network.
      contract = network.getContract(chaincodeName);

      // Initialize a set of asset data on the channel using the chaincode 'InitLedger' function.
      // This type of transaction would only be run once by an application the first time it was started after it
      // deployed the first time. Any updates to the chaincode deployed later would likely not need to run
      // an "init" type function.

      console.log(
        "\n--> Submit Transaction: InitLedger, function creates the initial set of devices on the ledger"
      );
      await contract.submitTransaction("InitLedger");
      console.log("*** Result: Committed InitLedger");

      // Let's try a query type operation (function).
      // This will be sent to just one peer and the results will be shown.

      // console.log('\n--> Submit Transaction: InitLedger, function creates the initial set of devices on the ledger');
      // await contract.submitTransaction('InitLedger');
      // console.log('*** Result: committed');

      // console.log('\n--> Evaluate Transaction: Get_All_devices, function returns all the current devices on the ledger');
      // let result = contract.evaluateTransaction('Get_All_devices');
      // console.log(result);
      // console.log(`*** Result: ${prettyJSONString(result.toString())}`);
      // response.send(result.toString());
    } finally {
      // console.log('\n--> Disconnecting the gateway');
      // // Disconnect from the gateway when the application is closing
      // // This will close all connections to the network
      // gateway.disconnect();
      // console.log('\n--> Successfully disconnected');
    }
  } catch (error) {
    console.error(`******** FAILED to run the application: ${error}`);
    process.exit(1);
  }
}

app.post("/register", async (request, response) => {
  const body = request.body;
  const deviceId = body.esp32id;
  const status = body.Status;
  console.log(
    "\n--> Submit Transaction: register_device, register new device with ID"
  );
  try {
    let result = await contract.submitTransaction(
      "register_device",
      deviceId,
      status
    );
    response.send("Device Registered");
  } catch (error) {
    response.send(error.message);
    console.log("Error in submitting transaction: ", error);
  }
});

app.post("/update", async (request, response) => {
  const body = request.body;
  const deviceId = body.esp32id;
  const status = body.Status;
  console.log(
    "\n--> Submit Transaction: Update_device, Updates the device with ID"
  );
  try {
    let result = await contract.submitTransaction(
      "Update_device",
      deviceId,
      status
    );
    response.send("Device Status Updated");
  } catch (error) {
    response.send(error.message);
  }
});

app.post("/auth", async (request, response) => {
  const body = request.body;
  const deviceId = body.esp32id;
  console.log(
    "\n--> Evaluate Transaction: Device_Auth, function returns an device with a given deviceID"
  );

  try {
    let result = await contract.evaluateTransaction("Device_Auth", deviceId);
    response.send("Device Authenticated");
  } catch (error) {
    response.send(error.message);
  }
});

app.get("/getall", async (request, response) => {
  console.log(
    "\n--> Evaluate Transaction: Get_All_devices, function returns all the current devices on the ledger"
  );
  result = await contract.evaluateTransaction("Get_All_devices");
  console.log(`*** Result: ${prettyJSONString(result.toString())}`);

  response.send(result);
});

app.post("/delete", async (request, response) => {
  const body = request.body;
  const deviceId = body.esp32id;
  console.log(
    "\n--> Submit Transaction: Delete_device, delete an given device from the world state"
  );
  try {
    let result = await contract.submitTransaction("Delete_device", deviceId);
    response.send("Device Deleted");
  } catch (error) {
    // response.send("Device doesn't found in the list");
    response.send(error.message);
  }
});
