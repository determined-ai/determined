# clearout listeners
kc -n projectcontour patch gateway contour --type=json -p='[{"op": "remove", "path": "/spec/listeners"}]'

# get listeners
kubectl -n default get gateway -o json | jq '.items[].spec.listeners[]'
