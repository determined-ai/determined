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
rm -f .amis
for region in ${REGIONS[@]}; do
	echo ${region}
	aws ec2 describe-images --region ${region} --owners self --output json | jq ".Images[] | {ImageId, Name, CreationDate}" | jq -s "sort_by(.CreationDate)" | jq -r '.[] | "\(.ImageId) \(.Name) \(.CreationDate)"' >.amis_${region}
	while read image; do
		arr=($image)
		if [[ "${arr[1]}" =~ "det-environments-".* ]]; then
			echo "${region} ${image}" >>.amis
		fi
	done <.amis_${region}
	rm .amis_${region}
done

rm -f .amis_to_deregister

while read ami; do
	amiarray=($ami)
	commit=${amiarray[2]##det-environments-}
	logsearch=$(git log --pretty=oneline -S${commit} -- ./bumpenvs.yaml)
	if [[ "$logsearch" ]]; then
		echo "Commit ${commit} appeared in bumpenvs.yaml here:"
		echo "$logsearch"
	else
		echo "Commit ${commit} not used! Maybe delete ${amiarray[1]}"
		echo "${amiarray[0]} ${amiarray[1]}" >>.amis_to_deregister
	fi
done <.amis

echo "AMIs from environments commits that did not appear in determined:"
cat .amis_to_deregister
wc -l .amis_to_deregister

#while read ami; do
#	ami=($ami)
#	aws ec2 deregister-image --region ${ami[0]} --image-id ${ami[1]}
#done <.amis_to_deregister
