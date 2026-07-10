#!/bin/bash
# Usage: curl -sL "https://github.com/48Club/service_agent/blob/main/upgrade.sh?raw=1" | zsh

source /etc/profile
source ~/.zshrc

# install jq
apt update
apt install -y jq

# update agent version
go install github.com/48Club/service_agent@8963823

# update config
jq '.skip_limit_methods = ["eth_sendRawTransaction","eth_getTransactionCount"]' /root/.config/service_agent/config.json > /root/.config/service_agent/config.json.tmp
mv /root/.config/service_agent/config.json.tmp /root/.config/service_agent/config.json

# restart service
systemctl restart service_agent.service