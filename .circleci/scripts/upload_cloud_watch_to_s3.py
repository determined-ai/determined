import argparse
import time
import sys

import boto3

logs = boto3.client("logs")


def upload_cloud_watch_logs(log_group_name: str, bucket_name: str, prefix: str) -> str:
    task_id = logs.create_export_task(
        logGroupName=log_group_name,
        fromTime=0,
        to=int(round(time.time() * 1000)),
        destination=bucket_name,
        destinationPrefix=prefix
    )["taskId"]

    print(f"Uploading logs to s3://{bucket_name}/{prefix}/{task_id}.")
    return task_id


def wait_for_export_task(task_id: str, retries: int = 300) -> None:
    attempts = 0
    while attempts < retries:
        response = logs.describe_export_tasks(taskId=task_id)
        code = response["exportTasks"][0]["status"]["code"]

        if code == "PENDING" or code == "RUNNING":
            print(f"Upload is {code}.")
            attempts += 1
            time.sleep(1)
            continue
        elif code == "COMPLETED":
            print("Upload completed successfully.")
            return
        else:
            print(f"Upload {code}: {response['exportTasks'][0]['status']['message']}.")
            sys.exit(1)

    if attempts == retries:
        print("Export task timed out.")
        sys.exit(1)


def main() -> None:
    parser = argparse.ArgumentParser(description="AWS cloud watch helper.")
    parser.add_argument("log_group_name", help="Name of log group to download logs from.")
    parser.add_argument("bucket_name", help="S3 bucket to move logs to.")
    parser.add_argument("prefix", help="Upload path in S3 bucket.")
    args = parser.parse_args()
    task_id = upload_cloud_watch_logs(args.log_group_name, args.bucket_name, args.prefix)
    wait_for_export_task(task_id)


if __name__ == "__main__":
    main()
