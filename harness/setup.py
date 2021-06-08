from setuptools import find_packages, setup

packages = find_packages(exclude=["*.tests", "*.tests.*", "tests.*", "tests"])
packages += [f"determined._swagger.{pkg}" for pkg in find_packages(where="determined/_swagger")]

setup(
    name="determined",
    version="0.15.6.dev0",
    author="Determined AI",
    author_email="hello@determined.ai",
    url="https://determined.ai/",
    description="Determined Deep Learning Training Platform",
    long_description="See https://docs.determined.ai/ for more information.",
    license="Apache License 2.0",
    classifiers=["License :: OSI Approved :: Apache Software License"],
    packages=packages,
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
        "yogadl==0.1.4",
        # Common:
        "backoff",
        "filelock",
        "google-cloud-storage>=1.20.0",
        # google-cloud-core 1.4.2 breaks our windows cli tests for python 3.5.
        "google-cloud-core<1.4.2",
        "hdfs>=2.2.2",
        "lomond>=0.3.3",
        "pathspec>=0.6.0",
        "simplejson",
        "termcolor>=1.1.0",
        "boto3",
        # CLI:
        "argcomplete>=1.9.4",
        "gitpython>=3.1.3",
        "pyOpenSSL>= 19.1.0",
        "python-dateutil",
        "tabulate>=0.8.3",
        # det preview-search "pretty-dumps" a sub-yaml with an API added in 0.15.29
        "ruamel.yaml>=0.15.29",
        # Deploy
        "docker[ssh]>=3.7.3",
        "google-api-python-client>=1.12.1",
        "paramiko>=2.4.2",  # explicitly pull in paramiko to prevent DistributionNotFound error
        "docker-compose>=1.13.0",
        "tqdm",
        "appdirs",
        # docker-compose has a requirement not properly propagated with semi-old pip installations;
        # so we expose that requirement here.
        "websocket-client<1",
        # Swagger-codegen: python requirements
        "certifi>=2017.4.17",
        "python-dateutil>=2.1",
        "six>=1.10",
        "urllib3>=1.23",
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
