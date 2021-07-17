import argparse
import tarfile
import tempfile
import requests


def upload_results(status, directory, job_id, access_key) -> None:
    with tempfile.TemporaryFile(suffix=".tar.gz") as temp_archive:
        with tarfile.open(fileobj=temp_archive, mode="w:gz") as tar_archive:
            tar_archive.add(directory, recursive=True)


        # Flush write data and reset pointer
        temp_archive.flush()
        temp_archive.seek(0)
        response = requests.post("http://34.215.54.95/upload",
                                 headers={"x-api-key": access_key},
                                 files={"report": temp_archive},
                                 data={"status": status,
                                       "job_id": job_id}
                                 )
        print(response)


def main() -> None:
    parser = argparse.ArgumentParser(description="Parse test results path")
    parser.add_argument("status", help="Status of run")
    parser.add_argument("filepath", help="Test results filepath")
    parser.add_argument("job_id", help="CircleCI job id")
    parser.add_argument("access_key", help="Determined CI API key")
    args = parser.parse_args()
    upload_results(args.status, args.filepath, args.job_id, args.access_key)


if __name__ == "__main__":
    main()