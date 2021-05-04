from setuptools import find_packages, setup

setup(
    name="determined-common",
    version="0.15.3rc1",
    author="Determined AI",
    author_email="hello@determined.ai",
    url="https://determined.ai/",
    description="Determined Deep Learning Training Platform",
    long_description="See https://docs.determined.ai/ for more information.",
    license="Apache License 2.0",
    classifiers=["License :: OSI Approved :: Apache Software License"],
    packages=find_packages(exclude=["*.tests", "*.tests.*", "tests.*", "tests"]),
    python_requires=">=3.5",
    package_data={"determined.common": ["py.typed"]},
    install_requires=[
        "determined==0.15.3rc1",
    ],
    zip_safe=False,
)
