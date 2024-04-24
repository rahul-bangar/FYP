#!/bin/bash
cd organizations/
rm -rf *
cp -r ~/fabric-samples/test-network/organizations/   ~/explorer-copy/organizations
cd peerOrganizations/org1.example.com/users/User1@org1.example.com/msp/keystore
filename=$(ls)
mv $filename priv_sk
cd ~/explorer-copy
docker-compose up -d