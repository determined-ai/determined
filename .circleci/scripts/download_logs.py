import argparse

import boto3

client = boto3.client("logs")


def get_log_streams(log_group_name):
    response = client.describe_log_streams(logGroupName=log_group_name)
    return [d["logStreamName"] for d in response["logStreams"]]


def get_full_log_group_name(log_group_name):
    response = client.describe_log_groups(logGroupNamePrefix=log_group_name)
    return response["logGroups"][0]["logGroupName"]


def download_log_stream(log_group_name, log_stream_name, log_download_dir):
    response = client.get_log_events(logGroupName=log_group_name, logStreamName=log_stream_name)
    with open(f"{log_download_dir}/{log_stream_name}.log", "w") as f:
        for log_event in response["events"]:
            f.write(f'{log_event["message"]}\n')


def main() -> None:
    parser = argparse.ArgumentParser(description="AWS logs helper.")
    parser.add_argument("log_group_name", help="Name of log group to download logs from.")
    parser.add_argument("log_download_dir", help="Directory to download logs to.")
    args = parser.parse_args()
    log_group_name = get_full_log_group_name(args.log_group_name)
    log_streams = get_log_streams(log_group_name)
    for log_stream in log_streams:
        download_log_stream(log_group_name, log_stream, args.log_download_dir)


if __name__ == "__main__":
    main()
