import json
import os

if __name__ == "__main__":
    # If DET_EXPERIMENT_CONFIG waas not parsable, we won't get here
    # and instead fail in prep_container.py
    expconf = os.environ.get("DET_EXPERIMENT_CONFIG")

    c = json.loads(expconf)

    for k, v in c["data"].items():
        # Print the data attributes
        print(f"DATA: {k}={v}")
