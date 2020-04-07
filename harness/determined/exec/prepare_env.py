import fnmatch
import sys
import zipfile

script_name = "startup-hook.sh"
if len(sys.argv) > 1:
    script_name = sys.argv[1]

try:
    model_def = zipfile.ZipFile("/run/determined/model_def.zip", "r")
    prepare_env_files = fnmatch.filter(model_def.namelist(), "*/" + script_name)
    sys.stdout.buffer.write(model_def.read(prepare_env_files[0]))
except (KeyError, IndexError, FileNotFoundError):
    print(
        f"""#!/bin/bash
        echo "No {script_name} found. Skipping..."
        """
    )
