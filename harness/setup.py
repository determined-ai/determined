from setuptools import find_packages, setup

setup(
    name="determined",
    version="0.14.4.dev0",
    author="Determined AI",
    author_email="hello@determined.ai",
    url="https://determined.ai/",
    description="Determined Deep Learning Training Platform",
    long_description="See https://docs.determined.ai/ for more information.",
    license="Apache License 2.0",
    classifiers=["License :: OSI Approved :: Apache Software License"],
    packages=find_packages(exclude=["*.tests", "*.tests.*", "tests.*", "tests"]),
    python_requires=">=3.6",
    package_data={"determined": ["py.typed"]},
    install_requires=[
        "cloudpickle==0.5.3",
        "determined-common==0.14.4.dev0",
        "dill>=0.2.9",
        # TF 2.2 has strict h5py requirements, which we expose here.
        "h5py>=2.10.0,<2.11.0",
        "matplotlib",
        "packaging",
        "numpy>=1.16.2",
        "psutil",
        "pyzmq>=18.1.0",
        "yogadl==0.1.3",
    ],
    extras_require={
        "tf-115-cuda101": ["tensorflow-gpu==2.4.1"],
        "tf-115-cpu": ["tensorflow==2.4.1"],
        "pytorch-14-cuda100": ["torch==1.7.1+cu100", "torchvision==0.8.2+cu100"],
        "pytorch-14-cpu": ["torch==1.7.1", "torchvision==0.8.2"],
    },
    zip_safe=False,
)
