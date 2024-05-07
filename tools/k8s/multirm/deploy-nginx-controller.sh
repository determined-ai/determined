# https://kind.sigs.k8s.io/docs/user/ingress/#ingress-nginx

# apply the necessary labels to all nodes
# Get all node names
nodes=$(kubectl get nodes --no-headers -o custom-columns=":metadata.name")

# Apply labels to each node
for node in $nodes; do
    echo "Labeling node $node"
    kubectl label nodes $node ingress-ready=true kubernetes.io/os=linux --overwrite
done

echo "All nodes have been labeled successfully."

deploy_file=https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml

kubectl delete -f $deploy_file
kubectl apply -f $deploy_file
kubectl wait --namespace ingress-nginx \
    --for=condition=ready pod \
    --selector=app.kubernetes.io/component=controller \
    --timeout=600s
