"""
Shows an example of how model trained in Determined can be easily exported and used.
"""

import argparse

import matplotlib.pyplot as plt
import tensorflow as tf

from determined.experimental import client


def generate_and_plot_images(generator: tf.keras.Sequential, noise_dim: int) -> None:
    # Notice `training` is set to False.
    # This is so all layers run in inference mode (batchnorm).
    seed = tf.random.normal([16, noise_dim])
    predictions = generator(seed, training=False)

    plt.figure(figsize=(4, 4))

    for i in range(predictions.shape[0]):
        plt.subplot(4, 4, i + 1)
        plt.imshow(predictions[i, :, :, 0] * 127.5 + 127.5, cmap="gray")
        plt.axis("off")
    plt.show()


def export_model(experiment_id: int) -> tf.keras.Model:
    checkpoint = client.get_experiment(experiment_id).top_checkpoint()
    model = checkpoint.load()
    return model


def main():
    parser = argparse.ArgumentParser(description="DCGan Model Export")
    parser.add_argument("--experiment-id", type=int, required=True, help="Experiment ID to export.")
    parser.add_argument("--master-url", type=str, default="", help="URL of the Determined master.")
    parser.add_argument(
        "--noise-dim",
        type=int,
        default=128,
        help="Needs to match noise dim during training.",
    )
    args = parser.parse_args()

    client.login(args.master_url)
    model = export_model(args.experiment_id)
    generate_and_plot_images(model.generator, args.noise_dim)


if __name__ == "__main__":
    main()
