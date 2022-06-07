# Get added files migration files.
added_migrations=$(git diff --name-only --diff-filter=A origin/$GITHUB_BASE_REF $GITHUB_SHA -- master/static/migrations/*.sql)
echo "Adding migrations " $added_migrations
if [ ! -z "$added_migrations" ]; then
    added_migrations=$(echo $added_migrations | xargs -n1 basename)
    for val in $added_migrations; do # Validate the filenames.
        regex="^[0-9]{14}[a-zA-Z_-]+\.tx\.(up|down)\.sql$"
        if [[ ! $val =~ $regex ]]; then
            done=1
            echo "migration $val is not in a valid format (does not pass $regex)"
        fi        
    done
    if [[ $done == 1 ]]; then
        exit 1
    fi
    
    echo "Migrations passed validation regex"
    
    # Get highest timestamp of migrations from branch you are trying to merge into.
    git checkout origin/$GITHUB_BASE_REF
    highest_before=$(find ./master/static/migrations/*.sql -printf "%f\n" | sort -n -t _ -k 1 -s | tail -1)
    git checkout $GITHUB_SHA
    
    # Check that the highest timestamp from before is lower than every added timestamp.
    highest=$(echo $added_migrations $highest_before | xargs -n1 | sort -n -t _ -k 1 -s | head -1)
    if [ "$highest_before" != "$highest" ]; then
        echo "Migration $highest has a timestamp smaller than a previously existing one $highest_before"
        echo "Please run migration-move-to-top.sh on the migrations if this is not intentional"
        exit 2
    fi
    echo "Migrations passed timestamp validation"
    exit 0
fi
echo "No migrations added"
