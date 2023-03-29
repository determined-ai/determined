#!/bin/sh

prog="$0"

print_help() {
    echo "usage: $prog [OPTIONS] VERSION...

where OPTIONS may be any of:
  -h, --help           Show this output.
  -c, --crawler-id ID  The algolia crawler id, default: \$ALGOLIA_CRAWLER_ID.
  -u, --user-id USER   The algolia user id, default: \$ALGOLIA_CRAWLER_USER_ID.
  -k, --api-key KEY    The algolia api key, default: \$ALGOLIA_CRAWLER_API_KEY.

VERSION describes which version subdirectory of the docs algolia should crawl:

    docs.determined.ai/VERSION/**

If multiple VERSION arguments are provided, multiple version subdirectories will
be indexed in a single crawl."
}

if [ -z "$1" ]; then
    # No args provided at all.
    print_help >&2
    exit 1
fi

crawler_id="$ALGOLIA_CRAWLER_ID"
user_id="$ALGOLIA_CRAWLER_USER_ID"
api_key="$ALGOLIA_CRAWLER_API_KEY"
urls=""
have_first_url="n"

# Manually parse args since getopt varies across unices.
while test -n "$1"; do
    case "$1" in
        # flags
        --help) print_help && exit 0 ;;
        -h) print_help && exit 0 ;;

        --crawler-id)
            crawler_id="$2"
            shift
            shift
            ;;
        -c)
            crawler_id="$2"
            shift
            shift
            ;;

        --user-id)
            user_id="$2"
            shift
            shift
            ;;
        -u)
            user_id="$2"
            shift
            shift
            ;;

        --api-key)
            api_key="$2"
            shift
            shift
            ;;
        -k)
            api_key="$2"
            shift
            shift
            ;;

        -*) echo "unrecognized flag: $1" >&2 && exit 1 ;;

        # positional arguments
        *)
            new_url="https://docs.determined.ai/$1"
            if [ "$have_first_url" = "n" ]; then
                have_first_url="y"
                # first url needs no comma
                urls="$urls\"$new_url\""
            else
                # second and later urls need a comma
                urls="$urls, \"$new_url\""
            fi
            shift
            ;;
    esac
done

ok="y"

if [ -z "$crawler_id" ]; then
    echo "--crawler-id not provided and ALGOLIA_CRAWLER_ID not set" >&2
    ok="n"
fi

if [ -z "$user_id" ]; then
    echo "--user-id not provided and ALGOLIA_CRAWLER_USER_ID not set" >&2
    ok="n"
fi

if [ -z "$api_key" ]; then
    echo "--api-key not provided and ALGOLIA_CRAWLER_API_KEY not set" >&2
    ok="n"
fi

if [ -z "$urls" ]; then
    echo "no VERSIONs provided" >&2
    ok="n"
fi

if [ "$ok" != "y" ]; then
    exit 1
fi

# Hit the "crawl a specific url" endpoint.
# Documentation: www.algolia.com/doc/rest-api/crawler/#crawl-specific-urls
curl \
    --user "$user_id:$api_key" \
    -H "Content-Type: application/json" \
    -d '{"urls": ['"$urls"'], "save": false}' \
    "https://crawler.algolia.com/api/1/crawlers/$crawler_id/urls/crawl"
