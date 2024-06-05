#!/usr/bin/bash

set -ex

# filename=gateways.gateway.networking.k8s.io.yaml
filename=$1
desired_max=${2:-128}
crd_name=gateways.gateway.networking.k8s.io

function download_version {
    version=$1
    filename=$2
    curl -L https://github.com/kubernetes-sigs/gateway-api/releases/download/$version/experimental-install.yaml -o $filename
}

function set_max_listerners {
    # desired_max=$1
    # cur_value=$(yq '.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.listeners.maxItems' $filename)
    doc=$(yq e 'select(.metadata.name == "gateways.gateway.networking.k8s.io")')
    cur_value=$(echo $doc | yq e '.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.listeners.maxItems')

    if [ $cur_value -eq $desired_max ]; then
        echo "Max listeners already set to $desired_max"
        return
    fi
    # if cur val is null
    if [ $cur_value == "null" ]; then
        exit 1
    fi
    echo "cur max listeners is: $cur_value. setting max listeners to $desired_max"
    yq eval ".spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.listeners.maxItems = $desired_max" -i $filename
    # sed -i '' "s/maxItems: $cur_value/maxItems: $desired_max/" gateways.gateway.networking.k8s.io.yaml
    # yq eval '
    #   del(.spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.listeners["x-kubernetes-validations"])
    # ' -i "$YAML_FILE"
}
function update_existing {
    kubectl get crd gateways.gateway.networking.k8s.io -o yaml >$filename
    set_max_listerners $desired_max
    read -p "Do you want to apply the changes? (y/n) " -n 1 -r
    kubectl delete crd gateways.gateway.networking.k8s.io
    kubectl apply -f $filename
}

if [ -f $filename ]; then
    set_max_listerners
else
    echo "missing file $filename"
    exit 1
fi
