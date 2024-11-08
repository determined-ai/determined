import setuptools

# open README.md from parent folder and read it into a string
with open("../README.md", "r") as readme:
    markdown_description = "".join(readme.readlines())

setuptools.setup(
    name="determined",
    version="0.35.1-rc0",
    author="Determined AI",
    author_email="ai-open-source@hpe.com",
    url="https://determined.ai/",
    description="Determined AI: The fastest and easiest way to build deep learning models.",
    long_description=markdown_description,
    long_description_content_type="text/markdown",
    license="Apache License 2.0",
    classifiers=["License :: OSI Approved :: Apache Software License"],
    # Use find_namespace_packages because it will include data-only packages (that is, directories
    # containing only non-python files, like our gcp terraform directory).
    packages=setuptools.find_namespace_packages(include=["determined*"]),
    # Technically, we haven't supported 3.6 or tested against it since it went EOL. But some users
    # are still using it successfully so there's hardly a point in breaking them.
    python_requires=">=3.8",
    include_package_data=True,
    install_requires=[
        "matplotlib",
        "packaging",
        "numpy>=1.16.2",
        "psutil",
        "pyzmq>=18.1.0",
        # Common:
        "certifi",
        "docker>=7.1.0",
        "filelock",
        "google-cloud-storage",
        "lomond>=0.3.3",
        "pathspec>=0.6.0",
        "azure-core",
        "azure-storage-blob",
        "termcolor>=1.1.0",
        "boto3",
        "oschmod;platform_system=='Windows'",
        # CLI:
        "argcomplete>=1.9.4",
        "gitpython>=3.1.3",
        "pyOpenSSL>= 19.1.0",
        "python-dateutil",
        "pytz",
        "tabulate>=0.8.3",
        "ruamel.yaml",
        # Deploy
        "docker[ssh]>=3.7.3",
        "google-api-python-client>=1.12.1",
        "paramiko>=2.4.2",  # explicitly pull in paramiko to prevent DistributionNotFound error
        "tqdm",
        "appdirs",
        # Telemetry
        "analytics-python",
    ],
    zip_safe=False,
    entry_points={
        "console_scripts": [
            "det = determined.cli.__main__:main",
        ]
    },
)
