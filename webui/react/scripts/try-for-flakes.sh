# eg ./bin/try-for-flakes.sh "npm run test -- --watchAll=false src/components/Timeago.test.tsx"

test_cmd=${1:-"npm run test -- --watchAll=false"}

c=0

while true; do
    c=$((c + 1))
    echo "run #$c"
    ${test_cmd}
done

echo "result: test failure at run #$c of $(git rev-parse --short HEAD)"
# TODO use trap to show successful result upon SIGTERM SIGINT
