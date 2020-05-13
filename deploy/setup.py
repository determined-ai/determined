from setuptools import find_packages, setup

setup(
    name="determined-deploy",
    version="0.12.4rc0",
    author="Determined AI",
    author_email="hello@determined.ai",
    url="https://determined.ai/",
    description="Determined Deep Learning Training Platform",
    long_description="See https://docs.determined.ai/ for more information.",
    classifiers=["License :: OSI Approved :: Apache Software License"],
    packages=find_packages(exclude=["*.tests", "*.tests.*", "tests.*", "tests"]),
    include_package_data=True,
    python_requires=">=3.6",
    install_requires=[
        "requests>=2.20.0",
        "docker[ssh]>=3.7.3",  # ssh extra prevents paramiko dependency error
        "docker-compose>=1.13.0",
        "determined-common==0.12.4rc0",
    ],
    entry_points={"console_scripts": ["det-deploy = determined_deploy.__main__:main"]},
)
