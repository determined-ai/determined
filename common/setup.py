from setuptools import find_packages, setup

setup(
    name="determined-common",
    version="0.13.7",
    author="Determined AI",
    author_email="hello@determined.ai",
    url="https://determined.ai/",
    description="Determined Deep Learning Training Platform",
    long_description="See https://docs.determined.ai/ for more information.",
    license="Apache License 2.0",
    classifiers=["License :: OSI Approved :: Apache Software License"],
    packages=find_packages(exclude=["*.tests", "*.tests.*", "tests.*", "tests"]),
    python_requires=">=3.5",
    package_data={"determined_common": ["py.typed"]},
    install_requires=[
        "google-cloud-storage>=1.20.0",
        # google-cloud-core 1.4.2 breaks our windows cli tests for python 3.5.
        "google-cloud-core<1.4.2",
        "hdfs>=2.2.2",
        "lomond>=0.3.3",
        "pathspec>=0.6.0",
        "ruamel.yaml>=0.15.78",
        "simplejson",
        "termcolor>=1.1.0",
        # boto3 1.14.11+ has consistent urllib3 requirements which we have to manually resolve.
        "boto3>=1.14.11",
        # requests<2.22.0 requires urllib3<1.25, which is incompatible with boto3>=1.14.11
        "requests>=2.22.0",
        # botocore>1.19.0 has stricter urllib3 requirements than boto3, and pip will not reliably
        # resolve it until the --use-feature=2020-resolver behavior in pip 20.3, so we list it here.
        "urllib3>=1.25.4,<1.26",
    ],
    zip_safe=False,
)
