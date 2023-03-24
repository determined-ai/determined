from argparse import ArgumentParser
from typing import Any, List

from google.cloud import storage


def list_blobs(storage_client: Any, bucket_name: str, prefix: str = None) -> List:
    # Helper functions for GCP from https://cloud.google.com/storage/docs/listing-objects#code-samples
    """Lists all the blobs in the bucket."""
    blobs = storage_client.list_blobs(bucket_name, prefix=prefix)
    return blobs


if __name__ == "__main__":
    parser = ArgumentParser(
        description="""Generate a listing of all blobs in a given GCS bucket/path for consumption by GCSImageFolder.
        After running, upload the file to the GCS bucket and supply its path in data_config.gcs_train_blob_list_path or
        data_config.gcs_validation_blob_list_path.
        See distributed-imagenet.yaml for an example."""
    )
    parser.add_argument(
        "--bucket-name",
        type=str,
        required=True,
        help="Name of the GCS bucket, without gs:// prefix.",
    )
    parser.add_argument("--bucket-path", type=str, required=True, help="Path prefix.")
    parser.add_argument("--output-file", type=str, required=True, help="File to output listing to.")
    args = parser.parse_args()
    storage_client = storage.Client()
    blobs = list_blobs(storage_client, args.bucket_name, prefix=args.bucket_path)
    with open(args.output_file, "w") as f:
        for b in blobs:
            f.write(b.name + "\n")
