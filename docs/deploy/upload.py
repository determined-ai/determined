import argparse
import hashlib
import mimetypes
import os
import pathlib

import boto3

HERE = pathlib.Path(__file__).parent

if __name__ == "__main__":

    def dir_path(string):
        if os.path.isdir(string):
            return string
        else:
            raise NotADirectoryError(string)

    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--preview",
        action="store_true",
        default=True,
        help="whether the upload should go under the short-lived /previews path",
    )
    parser.add_argument(
        "--version-file",
        type=str,
        default=HERE / ".." / ".." / "VERSION",
        help="file containing version string for local docs build",
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
        "--local-path",
        type=dir_path,
        default=HERE / ".." / "site" / "html",
        help="path to local site html",
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

    # check version file to determine upload path
    with args.version_file.open() as f:
        version = f.read().strip()

    # ya know, jic
    if version == "latest":
        print("no no no, we don't do that here")
        print("upload a version first, then use the promote.py script")
        os.exit(1)

    upload_root = version
    if args.preview:
        # we need some ID for the preview build
        if os.getenv("CIRCLECI"):
            # try branch name hash
            branch_hash = hashlib.md5(os.environ.get("CIRCLE_BRANCH", "MEH").encode("utf-8"))
            # breadcrumbs: md5 of "MEH" is "aea387423450ebf8c23aad69cbe364ed"
            upload_root = "previews/" + branch_hash.hexdigest()
        else:
            # local dev build? try username
            upload_root = "previews/" + os.environ.get("USER", "USERLESS")

    print(
        "uploading docs version {} to bucket {} under {}".format(
            version, args.bucket_id, upload_root
        )
    )

    s3 = boto3.resource("s3")
    bucket = s3.Bucket(args.bucket_id)
    # track current files for deletion, if any
    # WARN: be very careful when modifying this line, make sure you're listing
    # only the files at the exact upload root and not just the whole bucket
    current_objects = [o.key for o in bucket.objects.filter(Prefix=upload_root)]
    if len(current_objects) > 0:
        print("{} already at destination".format(len(current_objects)))

    # try uploading site files to upload path
    to_upload = pathlib.Path(args.local_path).rglob("*")
    uploaded_objects = []
    for f in to_upload:
        upload_path = str(upload_root / f.relative_to(args.local_path))
        if not f.is_file():
            continue
        # boto3 won't just automatically infer the mime-type
        mimetype, _ = mimetypes.guess_type(f)
        if mimetype is None:
            mimetype = "binary/octet-stream"
        with f.open("rb") as data:
            bucket.upload_fileobj(
                Fileobj=data, Key=upload_path, ExtraArgs={"ContentType": mimetype}
            )
            print("uploaded {}".format(upload_path))
            uploaded_objects.append(upload_path)

        with f.open("rb") as data:
            print("uploaded {}".format(upload_path))

    to_delete = set(current_objects).difference(uploaded_objects)
    if len(to_delete) > 0:
        print("deleting {} excess objects".format(len(to_delete)))
        delete_request = {"Objects": [{"Key": key} for key in to_delete], "Quiet": True}
        # TODO(danh): just uncomment this whenever the thought's not scary :D
        # bucket.delete_objects(Delete=delete_request)

    print(
        "upload done, {} objects uploaded, {} objects deleted".format(
            len(uploaded_objects), len(to_delete)
        )
    )

    # create invalidation if distribution ID provided and not a preview site
    # NOTE: we're explicitly avoiding invalidations for preview site uploads as
    # we have a quota on how many invalidations we can create, and we really
    # shouldn't waste them on what could potentially be dozens of changesets per
    # docs PR.
    if args.cf_distribution and not args.preview:
        client = boto3.client("cloudfront")
        timestamp = time.time_ns()
        path = "/{}/*".format(upload_root)
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

    objects_url = "https://{}.s3.{}.amazonaws.com/{}/index.html".format(
        args.bucket_id, args.bucket_region, upload_root
    )
    print("preview ready at: " + objects_url)
