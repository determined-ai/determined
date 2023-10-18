"""
Shows an example
"""

import argparse
import os

import matplotlib.pyplot as plt
import tensorflow as tf
from data import download, load_dataset

from determined import keras
from determined.experimental import client


def generate_and_plot_images(generator: tf.keras.Sequential) -> None:
    path = download("http://efrosgans.eecs.berkeley.edu/pix2pix/datasets/", "facades")
    test_ds = load_dataset(path, 256, 256, set_="test")
    example_input, example_target = next(iter(test_ds.take(1)))
    prediction = generator(tf.expand_dims(example_input, 0), training=True)

    plt.figure(figsize=(15, 15))

    display_list = [example_input, example_target, prediction[0]]
    title = ["Input Image", "Ground Truth", "Predicted Image"]

    for i in range(3):
        plt.subplot(1, 3, i + 1)
        plt.title(title[i])
        # Getting the pixel values in the [0, 1] range to plot.
        plt.imshow(display_list[i] * 0.5 + 0.5)
        plt.axis("off")
    plt.show()


def export_model(trial_id: int, latest=False) -> tf.keras.Model:
    trial = client.get_trial(trial_id)
    checkpoint: client.Checkpoint = (
        trial.select_checkpoint(latest=True) if latest else trial.top_checkpoint()
    )
    print(f"Checkpoint {checkpoint.uuid}")
    try:
        # Checkpoints from AWS deployment don't have these attributes
        print(f"Trial {checkpoint.trial_id}")
        print(f"Batch {checkpoint.batch_number}")
    except AttributeError:
        pass
    path = checkpoint.download()
    model = keras.load_model_from_checkpoint_path(path)
    return model


def main():
    parser = argparse.ArgumentParser(description="Pix2Pix model export")
    parser.add_argument("--trial-id", type=int, required=True, help="Trial ID to export.")
    parser.add_argument(
        "--master-url",
        type=str,
        default=os.environ["DET_MASTER"],
        help="URL of the Determined master (uses DET_MASTER environment variable by default).",
    )
    parser.add_argument(
        "--latest",
        action="store_true",
        help="Use the latest checkpoint. If omitted, the best checkpoint will be used.",
    )
    args = parser.parse_args()

    client.login(args.master_url)
    model = export_model(args.trial_id, args.latest)
    generate_and_plot_images(model.generator)


if __name__ == "__main__":
    main()
