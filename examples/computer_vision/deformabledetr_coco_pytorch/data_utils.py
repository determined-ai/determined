import os
import asyncio
from io import BytesIO
from tempfile import NamedTemporaryFile
import requests
from shutil import unpack_archive


def download_file(url, output_dir):
    local_filename = os.path.join(output_dir, url.split("/")[-1])
    with requests.get(url, stream=True) as r:
        r.raise_for_status()
        with open(local_filename, "wb") as f:
            for chunk in r.iter_content(chunk_size=8192):
                # If you have chunk encoded response uncomment the line below and set chunk_size parameter to None.
                # if chunk:
                f.write(chunk)
    return local_filename


async def download_and_extract_url(zipurl, outdir):
    filename = download_file(zipurl, outdir)
    with open(filename, "rb") as f, NamedTemporaryFile() as tfile:
        tfile.write(f.read())
        tfile.seek(0)
        unpack_archive(tfile.name, outdir, format="zip")
        print("finished extracting: {}".format(zipurl))
    await asyncio.sleep(1)


def async_download_url_list(url_list, outdir):
    loop = asyncio.get_event_loop()
    tasks = [
        asyncio.ensure_future(download_and_extract_url(url, outdir)) for url in url_list
    ]
    loop.run_until_complete(asyncio.gather(*tasks))


def download_coco_from_source(data_dir):
    url_list = [
        "http://images.cocodataset.org/zips/train2017.zip",
        "http://images.cocodataset.org/zips/val2017.zip",
    ]
    async_download_url_list(url_list, data_dir)
    done_path = os.path.join(data_dir, "done.txt")
    with open(done_path, "w") as f:
        f.write("done")


if __name__ == "__main__":
    download_coco_from_source("/tmp")
