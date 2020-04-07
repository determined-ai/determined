import pathlib

from setuptools import find_packages, setup

version_file = pathlib.Path(__file__).absolute().parents[1].joinpath("VERSION")
version = version_file.read_text()

setup(
    name="determined",
    version=version,
    author="Determined AI",
    author_email="hello@determined.ai",
    url="https://determined.ai/",
    description="Determined Deep Learning Training Platform",
    long_description="See https://docs.determined.ai/ for more information.",
    classifiers=["License :: OSI Approved :: Apache Software License"],
    packages=find_packages(exclude=["*.tests", "*.tests.*", "tests.*", "tests"]),
    python_requires=">=3.6",
    package_data={"determined": [str(version_file), "py.typed"]},
    install_requires=[
        "boto3>=1.9.220",
        "cloudpickle==0.5.3",
        "dill>=0.2.9",
        "h5py>=2.9.0",
        "lomond==0.3.3",
        "matplotlib",
        "packaging==19.0",
        "numpy>=1.16.2",
        "psutil",
        "pyzmq==18.1.0",
        "requests>=2.20.0",
        "simplejson==3.16.0",
        "GPUtil==1.4.0",
        f"determined-common=={version}",
    ],
    extras_require={
        "tf-114-cuda100": ["tensorflow-gpu==1.14.0"],
        "tf-114-cpu": ["tensorflow==1.14.0"],
        "pytorch-14-cuda100": ["torch==1.4.0+cu100", "torchvision==0.5.0+cu100"],
        "pytorch-14-cpu": ["torch==1.4.0", "torchvision==0.5.0"],
    },
    zip_safe=False,
)
