from setuptools import find_packages, setup

setup(
    name="determined-deploy",
    version="0.13.7",
    author="Determined AI",
    author_email="hello@determined.ai",
    url="https://determined.ai/",
    description="Determined Deep Learning Training Platform",
    long_description="See https://docs.determined.ai/ for more information.",
    license="Apache License 2.0",
    classifiers=["License :: OSI Approved :: Apache Software License"],
    packages=find_packages(exclude=["*.tests", "*.tests.*", "tests.*", "tests"]),
    include_package_data=True,
    python_requires=">=3.6",
    install_requires=[
        "docker[ssh]>=3.7.3",
        "google-api-python-client>=1.12.1",
        "paramiko>=2.4.2",  # explicitly pull in paramiko to prevent DistributionNotFound error
        "docker-compose>=1.13.0",
        "determined-common==0.13.7",
        # requests<2.22.0 requires urllib3<1.25, which is incompatible with boto3>=1.14.11
        "requests>=2.22.0",
        # botocore>1.19.0 has stricter urllib3 requirements than boto3, and pip will not reliably
        # resolve it until the --use-feature=2020-resolver behavior in pip 20.3, so we list it here.
        "urllib3>=1.25.4,<1.26",
        "tqdm",
    ],
    entry_points={"console_scripts": ["det-deploy = determined_deploy.__main__:main"]},
)
