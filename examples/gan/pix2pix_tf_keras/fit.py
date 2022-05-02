import matplotlib.pyplot as plt
import time

from data import download, get_dataset

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

    for step, data in train_ds.repeat().take(steps).enumerate():
        if preview and ((step) % preview == 0):
            # display.clear_output(wait=True)

            if step != 0:
                print(f"Time taken for {preview} steps: {time.time()-start:.2f} sec\n")

            start = time.time()

            generate_images(pix2pix.generator, example_input, example_target)
            print(f"Step: {step}")

        pix2pix.train_step(data)

        # Training step
        if (step + 1) % 10 == 0:
            print(".", end="", flush=True)


def main():
    path = download(0)
    train_dataset = get_dataset(path, batch_size=1)
    test_dataset = get_dataset(path, "test", batch_size=1)
    fit(train_dataset, test_dataset, steps=200, preview=100)


if __name__ == "__main__":
    main()
