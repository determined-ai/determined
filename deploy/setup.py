import pathlib

from setuptools import find_packages, setup

version_file = pathlib.Path(__file__).absolute().parents[1].joinpath("VERSION")
version = version_file.read_text()

setup(
    name="determined-deploy",
    version=version,
    author="Determined AI",
    author_email="hello@determined.ai",
    url="https://determined.ai/",
    description="Determined Deep Learning Training Platform",
    long_description="See https://docs.determined.ai/ for more information.",
    classifiers=["License :: OSI Approved :: Apache Software License"],
    packages=find_packages(exclude=["*.tests", "*.tests.*", "tests.*", "tests"]),
    include_package_data=True,
    python_requires=">=3.6",
    package_data={"determined-deploy": [str(version_file)]},
    install_requires=[
        "requests>=2.20.0",
        "docker-compose>=1.13.0",
        f"determined-common=={version}",
    ],
    entry_points={"console_scripts": ["det-deploy = determined_deploy.__main__:main"]},
)
