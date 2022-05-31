# Get added files migration files.
ADDED_MIGRATIONS=$(git diff --name-only --diff-filter=A $GITHUB_BASE_REF $GITHUB_HEAD_REF -- master/static/migrations/*.sql) 
if [ ! -z "$ADDED_MIGRATIONS" ]; then
    ADDED_MIGRATIONS=$(echo $ADDED_MIGRATIONS | xargs -n1 basename)
    for val in $ADDED_MIGRATIONS; do # Validate the filenames.
        REGEX="^[0-9]{14}[a-zA-Z_-]+\.tx\.(up|down)\.sql$"
        [[ $val =~ $REGEX ]] ||
            (EXIT=true && echo "migration $val is not in a valid format (does not pass $REGEX)")
    done
    [ -v $EXIT ] && exit 1;    
    
    # Get highest timestamp of migrations from branch you are trying to merge into.
    git checkout $GITHUB_BASE_REF
    HIGHEST_BEFORE=$(find ./master/static/migrations/*.sql -printf "%f\n" | sort -n -t _ -k 1 -s | tail -1)
    git checkout $GITHUB_HEAD_REF

    # Check that the highest timestamp from before is lower than every added timestamp.
    HIGHEST=$(echo $ADDED_MIGRATIONS $HIGHEST_BEFORE | xargs -n1 | sort -n -t _ -k 1 -s | head -1)
    if [ "$HIGHEST_BEFORE" != "$HIGHEST" ]; then
        echo "Migration $HIGHEST has a timestamp smaller than a previously existing one $HIGHEST_BEFORE"
        echo "Please run migration-move-to-top.sh on the migrations if this is not intentional"
        exit 1
    fi
fi
