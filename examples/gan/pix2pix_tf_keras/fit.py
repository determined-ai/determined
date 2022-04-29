import time

from data import download, get_train_dataset, get_validation_dataset

from pix2pix import Pix2Pix
from plotting import generate_images


pix2pix = Pix2Pix()
pix2pix.compile()


def fit(train_ds, test_ds, steps, preview=0):
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
    train_dataset = get_train_dataset(path, batch_size=1)
    test_dataset = get_validation_dataset(path, batch_size=1)
    fit(train_dataset, test_dataset, steps=200, preview=100)


if __name__ == "__main__":
    main()
