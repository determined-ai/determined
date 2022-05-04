"""
Shows an example
"""

import argparse
import tensorflow as tf
import matplotlib.pyplot as plt

from determined.experimental import client

from data import download, get_dataset


def generate_and_plot_images(generator: tf.keras.Sequential) -> None:
    path = download("http://efrosgans.eecs.berkeley.edu/pix2pix/datasets/", "facades")
    test_ds = get_dataset(path, 256, 256, set_="test", batch_size=0)
    example_input, example_target = next(iter(test_ds.take(1)))
    prediction = generator(tf.expand_dims(example_input, 0), training=False)

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


def export_model(experiment_id: int) -> tf.keras.Model:
    checkpoint = client.get_experiment(experiment_id).top_checkpoint()
    model = checkpoint.load()  # FIXME: deprecated
    return model


def main():
    parser = argparse.ArgumentParser(description="Pix2Pix model export")
    parser.add_argument(
        "--experiment-id", type=int, required=True, help="Experiment ID to export."
    )
    parser.add_argument(
        "--master-url", type=str, default="", help="URL of the Determined master."
    )
    args = parser.parse_args()

    client.login(args.master_url)
    model = export_model(args.experiment_id)
    generate_and_plot_images(model.generator)


if __name__ == "__main__":
    main()
