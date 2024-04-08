/*
 * Copyright IBM Corp. All Rights Reserved.
 *
 * SPDX-License-Identifier: Apache-2.0
 */

'use strict';

// Deterministic JSON.stringify()
const stringify  = require('json-stringify-deterministic');
const sortKeysRecursive  = require('sort-keys-recursive');
const { Contract } = require('fabric-contract-api');

class AssetTransfer extends Contract {

    async InitLedger(ctx) {
        const assets = [
            {
                ID: null,
                Status: null,
            }
        ];

        for (const asset of assets) {
            asset.docType = 'device';
            // example of how to write to world state deterministically
            // use convetion of alphabetic order
            // we insert data in alphabetic order using 'json-stringify-deterministic' and 'sort-keys-recursive'
            // when retrieving data, in any lang, the order of data will be the same and consequently also the corresonding hash
            await ctx.stub.putState(asset.ID, Buffer.from(stringify(sortKeysRecursive(asset))));
        }
    }

    // CreateAsset issues a new asset to the world state with given details.
    async register_device(ctx, id, status) {
        const exists = await this.device_exists(ctx, id);
        if (exists) {
            throw new Error(`The asset ${id} already exists`);
        }

        const asset = {
            ID: id,
            Status: status,
        };
        // we insert data in alphabetic order using 'json-stringify-deterministic' and 'sort-keys-recursive'
        await ctx.stub.putState(id, Buffer.from(stringify(sortKeysRecursive(asset))));
        return JSON.stringify(asset);
    }

    // UpdateAsset updates an existing asset in the world state with provided parameters.
    async Update_device(ctx, id, status) {
        const exists = await this.device_exists(ctx, id);
        if (!exists) {
            throw new Error(`The asset ${id} does not exist`);
        }

        // overwriting original asset with new asset
        const updatedAsset = {
            ID: id,
            Status: status,
        };
        // we insert data in alphabetic order using 'json-stringify-deterministic' and 'sort-keys-recursive'
        return ctx.stub.putState(id, Buffer.from(stringify(sortKeysRecursive(updatedAsset))));
    }

    // Device_Auth returns the device stored in the world state with given id.
    async Device_Auth(ctx, id) {

        const exists = await this.device_exists(ctx, id);
        if (!exists) {
            throw new Error(`The device ${id} does not exist`);
        }
        
        const assetJSON = await ctx.stub.getState(id); // get the device from chaincode state
        const asset = JSON.parse(assetJSON);

        // Access the Status property of the object
        const status = asset.Status;

        if (asset && asset.Status === "Inactive") {
            throw new Error(`The device ${id} is blacklisted`);
        }
        
        return assetJSON.toString();
    }

    // Delete_device deletes an given device from the world state.
    async Delete_device(ctx, id) {
        const exists = await this.device_exists(ctx, id);
        if (!exists) {
            throw new Error(`The device ${id} does not exist`);
        }
        return ctx.stub.deleteState(id);
    }

    // device_exists returns true when device with given ID exists in world state.
    async device_exists(ctx, id) {
        const assetJSON = await ctx.stub.getState(id);
        return assetJSON && assetJSON.length > 0;
    }

    // GetAllAssets returns all assets found in the world state.
    async Get_All_devices(ctx) {
        const allResults = [];
        // range query with empty string for startKey and endKey does an open-ended query of all assets in the chaincode namespace.
        const iterator = await ctx.stub.getStateByRange('', '');
        let result = await iterator.next();
        while (!result.done) {
            const strValue = Buffer.from(result.value.value.toString()).toString('utf8');
            let record;
            try {
                record = JSON.parse(strValue);
            } catch (err) {
                console.log(err);
                record = strValue;
            }
            allResults.push(record);
            result = await iterator.next();
        }
        return JSON.stringify(allResults);
    }
}

module.exports = AssetTransfer;