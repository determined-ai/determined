#/bin/sh

set -e

# The bucket to which we publish our docs.
bucket="determined-ai-docs"
# The cloudfront distribution ID for our docs site.
# Can be obtained via `aws cloudfront list-distributions`.
distribution="EXDWBK8432M1U"
# The version we publish to.
version=""
# the location of the built html
html="$(dirname "$0")/../site/html"
dryrun=""

while [ -n "$*" ] ; do
    arg="$1"
    shift;
    case "$arg" in
        --version)
            version="$1"
            shift
            ;;

        --bucket)
            bucket="$1"
            shift
            ;;

        --distribution)
            distribution="$1"
            shift
            ;;

        --html)
            html="$1"
            shift
            ;;

        --dry-run)
            dryrun="yes"
            ;;

        --help)
            echo "usage: $0 --version VERSION \\"
            echo "    [--bucket BUCKET] \\"
            echo "    [--distribution DISTRIBUTION_ID] \\"
            echo "    [--html PATH/TO/SITE/HTML]"
            echo "    [--dry-run]"
            exit 1;;

        *)
            echo "unrecognized argument; try $0 --help"
            exit 1;;
    esac
done

# check script inputs
ok="yes"
if [ -z "$version" ] ; then
    echo "missing --version"
    ok=""
fi
if [ -z "$bucket" ] ; then
    echo "missing --bucket"
    ok=""
fi
if [ -z "$distribution" ] ; then
    echo "missing --distribution"
    ok=""
fi
if [ ! -e "$html/attributions.html" ] ; then
    echo "the --html directory ($html) does not appear to be a sphinx build"
    ok=""
fi
if [ -z "$ok" ] ; then
    echo "try --help"
    exit 1
fi

echo_do () {
    echo "$@"
    if [ -z "$dryrun" ] ; then
        "$@"
    fi
}

# Actually do the publishing.
echo_do aws s3 sync "$build" "s3://$bucket/$version" --delete
echo_do aws cloudfront create-invalidation --distribution-id "$distribution" --paths "/$version/*"
