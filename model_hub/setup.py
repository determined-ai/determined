from setuptools import find_packages, setup

setup(
    name="model-hub",
    version="0.14.4.dev0",
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
        "transformers==4.2.2",
        "datasets==1.2.1",
    ],
    extras_require={
        "pytorch-17-cuda101": ["torch==1.7.1+cu101", "torchvision==0.8.2+cu101"],
        "pytorch-17-cuda110": ["torch==1.7.1+cu110", "torchvision==0.8.2+cu110"],
        "pytorch-17-cpu": ["torch==1.7.0", "torchvision==0.8.2"],
    },
    zip_safe=False,
)
