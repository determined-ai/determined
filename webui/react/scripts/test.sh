#!/bin/bash

# set up test server with
# proxy.js -- $PREVIEW_CLUSTER

api_path='/api/v1/experiments/metrics-stream/metric-names?'
direct_cluster=$PREVIEW_CLUSTER

# ids=(101 102 103 118 119 120 121 122 128 129 130 131 132 134 135 136 137 139 142 143 144 145 146 147 148 149 150 151 152 153 154 155 156 157 158 159 160 161 162 163 164 165 166 167 168 169 170 171 172 173 174 175 176 177 178 179 180 181 182 183 184 190 191 192 193 194 196 197 199 202 207 208 232 233 234 235 236 237 532)
ids=$(det -m "$direct_cluster" dev curl '/api/v1/experiments' | jq '.experiments[].id')

# server_paths=("$direct_cluster" "$PREVIEW_CLUSTER_PROXY" http://localhost:8080/fixed)
server_paths=("$direct_cluster" http://localhost:8100/fixed)

token=$(det -m $direct_cluster dev auth-token)

for id in $ids; do
    api_path="$api_path&ids=$id"
done

for server_path in "${server_paths[@]}"; do
    echo "Testing $server_path"
    # token=$(det -m $server_path dev auth-token)
    # curl -I -H "Authorization: Bearer $token" "$server_path$api_path"
    curl --silent -H "Authorization: Bearer $token" "$server_path$api_path"
    echo
    sleep 1
done
