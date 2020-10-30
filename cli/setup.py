from setuptools import find_packages, setup

setup(
    name="determined-cli",
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
    install_requires=[
        "argcomplete>=1.9.4",
        "gitpython>=3.1.3",
        "packaging",
        "pyOpenSSL>= 19.1.0",
        "python-dateutil",
        "ruamel.yaml>=0.15.78",
        "tabulate>=0.8.3",
        "termcolor==1.1.0",
        "determined-common==0.13.7",
        # requests<2.22.0 requires urllib3<1.25, which is incompatible with boto3>=1.14.11
        "requests>=2.22.0",
        # botocore>1.19.0 has stricter urllib3 requirements than boto3, and pip will not reliably
        # resolve it until the --use-feature=2020-resolver behavior in pip 20.3, so we list it here.
        "urllib3>=1.25.4,<1.26",
    ],
    entry_points={"console_scripts": ["det = determined_cli.__main__:main"]},
)
