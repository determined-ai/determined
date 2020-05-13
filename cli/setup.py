from setuptools import find_packages, setup

setup(
    name="determined-cli",
    version="0.12.4rc0",
    author="Determined AI",
    author_email="hello@determined.ai",
    url="https://determined.ai/",
    description="Determined Deep Learning Training Platform",
    long_description="See https://docs.determined.ai/ for more information.",
    classifiers=["License :: OSI Approved :: Apache Software License"],
    packages=find_packages(exclude=["*.tests", "*.tests.*", "tests.*", "tests"]),
    python_requires=">=3.5",
    install_requires=[
        "argcomplete==1.9.4",
        "gitpython==2.1.11",
        "packaging==19.0",
        "python-dateutil==2.8.0",
        "requests>=2.20.0",
        "ruamel.yaml>=0.15.78",
        "tabulate>=0.8.3",
        "termcolor==1.1.0",
        "determined-common==0.12.4rc0",
    ],
    entry_points={"console_scripts": ["det = determined_cli.__main__:main"]},
)
