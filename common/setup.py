from setuptools import find_packages, setup

setup(
    name="determined-common",
    version="0.13.5rc0",
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
        "boto3>=1.9.220",
        "hdfs>=2.2.2",
        "lomond>=0.3.3",
        "pathspec>=0.6.0",
        "requests>=2.20.0",
        "ruamel.yaml>=0.15.78",
        "simplejson",
        "termcolor>=1.1.0",
    ],
    zip_safe=False,
)
