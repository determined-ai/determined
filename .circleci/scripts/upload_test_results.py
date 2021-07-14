import argparse
import tarfile
import tempfile
import requests


def upload_results(directory, job_id) -> None:
    with tempfile.TemporaryFile(suffix=".tar.gz") as temp_archive:
        with tarfile.open(fileobj=temp_archive, mode="w:gz") as tar_archive:
            tar_archive.add(directory, recursive=True)

        response = requests.post("http://34.215.54.95/upload",
                                 files={"report": temp_archive},
                                 data={"job_id": job_id})
        print(response)



def main() -> None:
    parser = argparse.ArgumentParser(description="Parse test results path")
    parser.add_argument("filepath", help="Test results filepath")
    parser.add_argument("job_id", help="CircleCI job id")
    args = parser.parse_args()
    print(args)
    upload_results(args.filepath, args.job_id)


if __name__ == "__main__":
    main()