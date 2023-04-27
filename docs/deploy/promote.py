import argparse
import os
import pathlib
import time

import boto3

HERE = pathlib.Path(__file__).parent

EXCLUDES = ["release-notes/", "attributions.xml"]

BUILD = str(HERE / ".." / "site" / "xml")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--version",
        type=str,
        help="version to copy over to site root",
    )
    parser.add_argument(
        "--root-prefix",
        type=str,
        default=os.environ.get("DOCSITE_ROOT_KEY", "latest/"),
        help="object key prefix for site root",
    )
    parser.add_argument(
        "--bucket-id",
        type=str,
        default="determined-ai-docs",
        help="S3 bucket ID where doc pages are served from",
    )
    parser.add_argument(
        "--bucket-region",
        type=str,
        default="us-west-2",
        help="S3 bucket region where doc pages are served from",
    )
    parser.add_argument(
        "--cf-distribution",
        type=str,
        default=os.environ.get("CF_DISTRIBUTION_ID", ""),
        help="CloudFront distribution ID to create invalidation for",
    )
    parser.add_argument(
        "--aws-access-key-id",
        type=str,
        default=os.environ.get("AWS_ACCESS_KEY_ID"),
        help="AWS access key ID for uploading to S3",
    )
    parser.add_argument(
        "--aws-secret-access-key",
        type=str,
        default=os.environ.get("AWS_SECRET_ACCESS_KEY"),
        help="AWS secret access key for uploading to S3",
    )
    args = parser.parse_args()

    if "preview" in args.version:
        print("only promote published doc versions, not previews!")
        os.exit(1)

    source_prefix = args.version + "/"
    target_prefix = args.root_prefix
    if target_prefix[-1] != "/":
        # for consistency of use, ensure the prefix ends with a /
        target_prefix = target_prefix + "/"

    s3 = boto3.resource("s3")
    bucket = s3.Bucket(args.bucket_id)
    # confirm the version given actually exists
    source_objects = [o.key for o in bucket.objects.filter(Prefix=source_prefix)]
    if len(source_objects) == 0:
        print("version specified doesn't seem to be uploaded")
        print("run upload.py for a build of that version before retrying")
        os.exit(1)

    # TODO(danh): reference existing Algolia search indices for the version
    # being promoted to ensure only indexed docsites are being promoted

    # track current objects to know what needs to be cleaned up afterwards
    current_objects = [o.key for o in bucket.objects.filter(Prefix=target_prefix)]
    if len(current_objects) > 0:
        print("{} already at site root".format(len(current_objects)))

    # copy over blobs to latest, tracking all files that were copied over
    copied_objects = []
    for src in source_objects:
        copy_source = {
            "Bucket": bucket.name,
            "Key": src,
        }
        src_name = str(pathlib.Path(src).relative_to(source_prefix))
        target_key = target_prefix + src_name
        bucket.copy(copy_source, target_key)
        print("copied over {}".format(target_key))
        copied_objects.append(target_key)

    to_delete = set(current_objects).difference(copied_objects)
    if len(to_delete) > 0:
        print("deleting {} excess objects".format(len(to_delete)))
        delete_request = {"Objects": [{"Key": key} for key in to_delete], "Quiet": True}
        # TODO(danh): just uncomment this whenever the thought's not scary :D
        # bucket.delete_objects(Delete=delete_request)

    print(
        "upload done, {} objects copied, {} objects deleted".format(
            len(copied_objects), len(to_delete)
        )
    )

    # create invalidation if distribution ID provided
    if args.cf_distribution:
        client = boto3.client("cloudfront")
        timestamp = time.time_ns()
        path = "/{}*".format(target_prefix)
        client.create_invalidation(
            DistributionId=args.cf_distribution,
            InvalidationBatch={
                "Paths": {
                    "Quantity": 1,
                    "Items": [
                        path,
                    ],
                },
                "CallerReference": str(timestamp),
            },
        )
        print("invalidation {} created for {}".format(timestamp, path))

    objects_url = "https://{}.s3.{}.amazonaws.com/{}index.html".format(
        args.bucket_id, args.bucket_region, target_prefix
    )
    print("site root available at: " + objects_url)
