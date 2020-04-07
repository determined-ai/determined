import pathlib

from setuptools import find_packages, setup

version_file = pathlib.Path(__file__).absolute().parents[1].joinpath("VERSION")
version = version_file.read_text()

setup(
    name="determined-common",
    version=version,
    author="Determined AI",
    author_email="hello@determined.ai",
    url="https://determined.ai/",
    description="Determined Deep Learning Training Platform",
    long_description="See https://docs.determined.ai/ for more information.",
    classifiers=["License :: OSI Approved :: Apache Software License"],
    packages=find_packages(exclude=["*.tests", "*.tests.*", "tests.*", "tests"]),
    python_requires=">=3.5",
    package_data={"determined-common": [str(version_file), "py.typed"]},
    install_requires=[
        "boto3>=1.9.220",
        "dai-sgqlc-3.5",
        "google-cloud-storage>=1.20.0",
        "graphql-core==2.3.1",
        "hdfs>=2.2.2",
        "lomond==0.3.3",
        "pathspec>=0.6.0",
        "requests>=2.20.0",
        "ruamel.yaml>=0.15.78",
        "simplejson==3.16.0",
    ],
    zip_safe=False,
)
