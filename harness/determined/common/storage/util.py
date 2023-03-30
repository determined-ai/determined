import urllib

from determined.common import storage


def from_string(shortcut) -> storage.base.StorageManager:
    p: urllib.parse.ParseResult = urllib.parse.urlparse(shortcut)
    if p.scheme == "ms":
        container = p.netloc
        connection_string = p.fragment
        kwargs = urllib.parse.parse_qs(p.query)
        account_url = kwargs.get("account_url", [None])[0]
        credential = kwargs.get("credential", [None])[0]
        temp_dir = kwargs.get("temp_dir", [None])[0]
        return storage.AzureStorageManager(
            container=container,
            connection_string=connection_string,
            account_url=account_url,
            credential=credential,
            temp_dir=temp_dir,
        )
    elif p.scheme == "gs":
        bucket = p.netloc
        prefix = p.path.lstrip("/")
        kwargs = urllib.parse.parse_qs(p.query)
        temp_dir = kwargs.get("temp_dir", [None])[0]
        print(f"{kwargs=}")
        return storage.gcs.GCSStorageManager(bucket=bucket, prefix=prefix, temp_dir=temp_dir)
    elif p.scheme == "s3":
        bucket = p.netloc
        prefix = p.path.lstrip("/")
        kwargs = urllib.parse.parse_qs(p.query)
        access_key = kwargs.get("access_key", [None])[0]
        secret_key = kwargs.get("secret_key", [None])[0]
        endpoint_url = kwargs.get("endpoint_url", [None])[0]
        temp_dir = kwargs.get("temp_dir", [None])[0]
        return storage.s3.S3StorageManager(
            bucket=bucket,
            prefix=prefix,
            access_key=access_key,
            secret_key=secret_key,
            endpoint_url=endpoint_url,
            temp_dir=temp_dir,
        )
    else:
        raise ValueError(f'Could not understand storage manager scheme "{p.scheme}"')
