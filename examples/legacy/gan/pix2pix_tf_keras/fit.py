import time

import matplotlib.pyplot as plt
import tensorflow as tf
from data import download, load_dataset
from pix2pix import Pix2Pix


def generate_images(model, test_input, target):
    prediction = model(test_input, training=True)
    plt.figure(figsize=(15, 15))

    display_list = [test_input[0], target[0], prediction[0]]
    title = ["Input Image", "Ground Truth", "Predicted Image"]

    for i in range(3):
        plt.subplot(1, 3, i + 1)
        plt.title(title[i])
        # Getting the pixel values in the [0, 1] range to plot.
        plt.imshow(display_list[i] * 0.5 + 0.5)
        plt.axis("off")
    plt.show()


def fit(train_ds, test_ds, steps, preview=0):
    pix2pix = Pix2Pix()
    pix2pix.compile()

    example_input, example_target = next(iter(test_ds.take(1)))
    start = time.time()

    for step, batch in train_ds.repeat().take(steps).enumerate():
        # Training step
        losses = pix2pix.train_step(batch, verbose=True)
        if (step + 1) % 10 == 0:
            print(".", end="", flush=True)
        if preview and ((step + 1) % preview == 0):
            if step != 0:
                print(f"Time taken for {preview} steps: {time.time()-start:.2f} sec\n")
                print("g_gan_loss: ", losses["g_gan_loss"])
                print("g_l1_loss:  ", losses["g_l1_loss"])
                print("g_loss:     ", losses["g_loss"])
                print("d_loss:     ", losses["d_loss"])
                print("total_loss: ", losses["total_loss"])
                val_losses = pix2pix.test_step(next(iter(test_ds)), verbose=True)
                print("val_g_gan_loss: ", val_losses["g_gan_loss"])
                print("val_g_l1_loss:  ", val_losses["g_l1_loss"])
                print("val_g_loss:     ", val_losses["g_loss"])
                print("val_d_loss:     ", val_losses["d_loss"])
                print("val_total_loss: ", val_losses["total_loss"])
                generate_images(pix2pix.generator, example_input, example_target)
            print(f"Step: {step + 1}")
            start = time.time()


def main():
    import yaml

    config = yaml.load(open("const.yaml", "r"), Loader=yaml.BaseLoader)
    path, dataset_name = config["data"]["base"], config["data"]["dataset"]
    path = download(path, dataset_name)

    train_dataset = load_dataset(path, 256, 256, "train", jitter=30, mirror=True)
    train_dataset = train_dataset.cache().shuffle(400).batch(40).repeat()
    train_dataset = train_dataset.prefetch(buffer_size=tf.data.experimental.AUTOTUNE)

    test_dataset = load_dataset(path, 256, 256, "test")
    test_dataset = test_dataset.batch(50)
    fit(train_dataset, test_dataset, steps=10, preview=10)


if __name__ == "__main__":
    main()
