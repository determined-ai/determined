#!/bin/bash

echo "Setting up det k8's contexts for kind-main and kind-extra"
current_context=$(kubectl config current-context)
kubectl config use-context kind-main
cp ~/.kube/config ~/.kube/kindmainconfig
kubectl config use-context kind-extra
cp ~/.kube/config ~/.kube/kindextraconfig
kubectl config use-context $current_context
