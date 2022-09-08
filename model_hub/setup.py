from setuptools import find_packages, setup

setup(
    name="model-hub",
    version="0.19.3-dev0",
    author="Determined AI",
    author_email="hello@determined.ai",
    url="https://determined.ai/",
    description="Model Hub for Determined Deep Learning Training Platform",
    long_description="See https://docs.determined.ai/ for more information.",
    license="Apache License 2.0",
    classifiers=["License :: OSI Approved :: Apache Software License"],
    packages=find_packages(exclude=["*.tests", "*.tests.*", "tests.*", "tests"]),
    python_requires=">=3.6",
    package_data={"model_hub": ["py.typed"]},
    # Versions of model-hub will correspond to specific versions of third party
    # libraries that are guaranteed to work with our code.  Other versions
    # may work with model-hub as well but are not officially supported.
    install_requires=[
        "attrdict",
        "determined>=0.13.11",  # We require custom reducers for PyTorchTrial.
    ],
    zip_safe=False,
)
