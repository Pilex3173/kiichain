#!/bin/bash

# Set the env vars
# Initial version defines the version on which we start the upgrade
# This tag should be available on the kiichain repo
INITIAL_VERSION=v4.0.0
# Upgrade tag defines the version to which we upgrade
# This tag should be available on the kiichain repo
UPGRADE_TAG=v5.0.0-test
# Upgrade name is the name of the upgrade proposal
# This name should be on the code as the expected upgrade name
UPGRADE_NAME=v5.0.0
PROJECT_DIR=$(pwd)

# wait_for_height waits for a specific height to be reached
wait_for_height() {
    local target_height="$1"
    local node="${2:-http://localhost:26657}"

    echo "⏳ Waiting for block height $target_height..."

    while true; do
        current_height=$(curl -s "$node/status" | jq -r .result.sync_info.latest_block_height)

        if [[ "$current_height" =~ ^[0-9]+$ ]] && ((current_height >= target_height)); then
            echo "✅ Reached block height $current_height"
            break

        fi
        echo "Current height: $current_height, waiting..."
        sleep 1
    done
}

# Clone the Kiichain
rm -rf /tmp/kiichain
git clone git@github.com:KiiChain/kiichain.git /tmp/kiichain
cd /tmp/kiichain
git checkout $INITIAL_VERSION
make install
kiichaind version
cd $PROJECT_DIR

# Update json file with the target
jq --arg new_name "$UPGRADE_NAME" '.messages[0].plan.name = $new_name' contrib/upgrade/upgrade_json.json >tmp.json && mv tmp.json contrib/upgrade/upgrade_json.json

# Start the new node
nohup /tmp/kiichain/contrib/local_node.sh -y --no-install >node.log 2>&1 &
wait_for_height 5

# Apply the upgrade proposal
kiichaind tx gov submit-proposal contrib/upgrade/upgrade_json.json --keyring-backend test --from mykey --fees 1000000000000000000akii -y
sleep 5

# Vote for the proposal
kiichaind tx gov vote 1 yes --keyring-backend test --from mykey --fees 1000000000000000000akii -y
wait_for_height 15
sleep 5

# Kill the node
pkill kiichaind

# Install the new version
cd /tmp/kiichain
git checkout $UPGRADE_TAG
make install
kiichaind version
cd $PROJECT_DIR

# Start the new node with the new version
kiichaind start --minimum-gas-prices=0.0001akii
