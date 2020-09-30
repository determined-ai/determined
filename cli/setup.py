from setuptools import find_packages, setup

setup(
    name="determined-cli",
    version="0.13.5rc1",
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
        "requests>=2.20.0",
        "ruamel.yaml>=0.15.78",
        "tabulate>=0.8.3",
        "termcolor==1.1.0",
        "determined-common==0.13.5rc1",
    ],
    entry_points={"console_scripts": ["det = determined_cli.__main__:main"]},
)
