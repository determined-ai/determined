REGIONS=(
	"ap-northeast-1"
	"ap-northeast-2"
	"ap-southeast-1"
	"ap-southeast-2"
	"eu-central-1"
	"eu-west-1"
	"eu-west-2"
	"us-east-1"
	"us-east-2"
)
for region in ${REGIONS[@]}; do
	aws ec2 describe-images --region ${region} --output json --filters "Name=name,Values=det-environments-*" \
        | jq ".Images[] | {ImageId, Name, CreationDate}" \
        | jq -s "sort_by(.CreationDate)" \
        | jq -r '.[] | "\(.ImageId) \(.Name) \(.CreationDate)"' \
        | sed "s/\$/ ${region}/"
done | sort -u > all_amis_info

cat all_amis_info | awk '{ print $1 }' > all_amis

while read tag; do
    git -C $DET_PROJ show $tag:tools/scripts/bumpenvs.yaml \
        | yq -r ".[].new" \
        | sed -n "/ami/p"
done < det_versions | sort -u > published_amis

for region in ${REGIONS[@]}; do
    aws ec2 describe-images --region ${region} --output json \
        --filters "Name=name,Values=det-environments-*" \
        --query 'Images[?CreationDate>`2023-02-02`]' \
        | jq -r '.[].ImageId'
done | sort -u > recent_amis

cat all_amis published_amis published_amis recent_amis recent_amis \
    | sort | uniq -u > deletable_amis

grep -f deletable_amis all_amis_info > deletable_amis_info

wc -l deletable_amis
wc -l deletable_amis_info

#while read ami; do
#	ami=($ami)
#	aws ec2 deregister-image --region ${ami[0]} --image-id ${ami[1]}
#done <.amis_to_deregister
