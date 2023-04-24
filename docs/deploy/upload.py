import argparse
import glob
import hashlib
import os
import pathlib

import boto3

HERE = pathlib.Path(__file__).parent

if __name__ == "__main__":
    # get path to blob to upload
    # get account creds
    # check if objects already exist at destination
    # check if upload being done is a preview site
    # opt: flag to delete blobs missing from destination

    def dir_path(string):
        if os.path.isdir(string):
            return string
        else:
            raise NotADirectoryError(string)

    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--preview",
        type=bool,
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
        "--local-path", type=dir_path, default="../site/html", help="path to local site html"
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

    upload_root = "/" + version
    if args.preview:
        # we need some ID for the preview build
        if os.getenv("CIRCLECI"):
            # try branch name hash
            branch_hash = hashlib.md5(os.environ.get("CIRCLE_BRANCH", "MEH"))
            # breadcrumbs: md5 of "MEH" is "aea387423450ebf8c23aad69cbe364ed"
            upload_root = "/previews/" + branch_hash
        else:
            # local dev build? try username
            upload_root = "/previews/" + os.environ.get("USER", "USERLESS")

    print(
        "uploading docs version {} to bucket {} under {}".format(
            version, args.bucket_id, upload_root
        )
    )

    s3 = boto3.resource("s3")
    bucket = s3.Bucket(args.bucket_id)
    # track current files for deletion, if any
    current_objects = [o.key for o in bucket.objects.all()]
    if len(current_objects) > 0:
        print("{} already at destination".format(len(current_objects)))

    # try uploading site files to upload path
    to_upload = glob.glob("{}/**".format(args.local_path))
    uploaded_objects = []
    for file_path in to_upload:
        upload_path = upload_root + "/" + file_path
        with open(file_path, "rb") as data:
            # bucket.upload_fileobj(data, upload_path)
            print("uploaded {}".format(upload_path))
            uploaded_objects.append(upload_path)

    to_delete = set(current_objects).difference(uploaded_objects)
    if len(to_delete) > 0:
        print("deleting {} excess objects".format(len(to_delete)))
    for key in to_delete:
        delete_request = {"Objects": [{"Key": key} for key in to_delete], "Quiet": True}
        # bucket.delete_objects(Delete=delete_request)

    objects_url = "https://{}.s3.{}.amazonaws.com/{}/index.html".format(
        args.bucket_id, args.bucket_region, upload_root
    )
    print("upload done, preview ready at: " + objects_url)
