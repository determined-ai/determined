from setuptools import find_packages, setup

setup(
    name="determined",
    version="0.15.0rc1",
    author="Determined AI",
    author_email="hello@determined.ai",
    url="https://determined.ai/",
    description="Determined Deep Learning Training Platform",
    long_description="See https://docs.determined.ai/ for more information.",
    license="Apache License 2.0",
    classifiers=["License :: OSI Approved :: Apache Software License"],
    packages=find_packages(exclude=["*.tests", "*.tests.*", "tests.*", "tests"]),
    python_requires=">=3.6",
    package_data={"determined": ["py.typed"]},
    include_package_data=True,
    install_requires=[
        "dill>=0.2.9",
        # TF 2.2 has strict h5py requirements, which we expose here.
        "h5py>=2.10.0,<2.11.0",
        "matplotlib",
        "packaging",
        "numpy>=1.16.2",
        "psutil",
        "pyzmq>=18.1.0",
        "yogadl==0.1.3",
        # Common:
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
        # CLI:
        "argcomplete>=1.9.4",
        "gitpython>=3.1.3",
        "pyOpenSSL>= 19.1.0",
        "python-dateutil",
        "ruamel.yaml>=0.15.78",
        "tabulate>=0.8.3",
        # Deploy
        "docker[ssh]>=3.7.3",
        "google-api-python-client>=1.12.1",
        "paramiko>=2.4.2",  # explicitly pull in paramiko to prevent DistributionNotFound error
        "docker-compose>=1.13.0",
        "tqdm",
        "appdirs",
    ],
    extras_require={
        "tf-115-cuda102": ["tensorflow-gpu==1.15.5"],
        "tf-115-cpu": ["tensorflow==1.15.5"],
        "tf-240-cuda102": ["tensorflow-gpu==2.4.1"],
        "tf-240-cpu": ["tensorflow==2.4.1"],
        "tf-241-cuda110": ["tensorflow-gpu==2.4.1"],
        "pytorch-18-cuda102": ["torch==1.7.1", "torchvision==0.8.2"],
        "pytorch-18-cuda110": ["torch==1.7.1", "torchvision==0.8.2"],
        "pytorch-18-cpu": ["torch==1.7.1", "torchvision==0.8.2"],
    },
    zip_safe=False,
    entry_points={
        "console_scripts": [
            "det = determined.cli.__main__:main",
            "det-deploy = determined.deploy.__main__:main",
        ]
    },
)
