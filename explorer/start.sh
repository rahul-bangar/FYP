#!/bin/bash
cd ~/fabric-samples/explorer/organizations
rm -rf *
cp -r ~/fabric-samples/test-network/organizations/   ~/fabric-samples/explorer/organizations
cd peerOrganizations/org1.example.com/users/User1@org1.example.com/msp/keystore
filename=$(ls)
mv $filename priv_sk
cd ~/fabric-samples/explorer
docker-compose up -d