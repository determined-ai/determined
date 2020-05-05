from typing import Any, Dict, List, Tuple

import boto3.exceptions


class MockS3Client:
    def __init__(self, faulty: bool = False) -> None:
        self.objects = {}  # type: Dict[Tuple[str, str], str]
        self.faulty = faulty

    def put_object(self, **kwargs: str) -> None:
        if self.faulty:
            raise boto3.exceptions.S3UploadFailedError()
        self.objects[(kwargs["Bucket"], kwargs["Key"])] = kwargs["Body"]

    def upload_file(self, path: str, bucket: str, key: str) -> None:
        with open(path, "r") as fp:
            self.put_object(Bucket=bucket, Key=key, Body=fp.read())

    def download_file(self, bucket: str, key: str, path: str) -> None:
        with open(path, "w") as fp:
            fp.write(self.objects[(bucket, key)])

    # kwargs are capital to match the signature of the boto3 s3 client
    def delete_objects(self, Bucket: str, Delete: Dict[str, List[Dict[str, str]]]) -> None:
        assert "Objects" in Delete
        keys = Delete["Objects"]
        for key in keys:
            del self.objects[(Bucket, key["Key"])]


def s3_client(_1: str, **_2: Any) -> MockS3Client:
    return MockS3Client()


def s3_faulty_client(_1: str, **_2: Any) -> MockS3Client:
    return MockS3Client(faulty=True)
