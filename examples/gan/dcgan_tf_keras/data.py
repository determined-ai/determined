import tensorflow as tf


def get_train_dataset(worker_rank: int):
    (train_images, _), (_, _) = tf.keras.datasets.mnist.load_data(path=f"mnist-{worker_rank}.npz")

    train_images = train_images.reshape(train_images.shape[0], 28, 28, 1).astype('float32')
    train_images = (train_images - 127.5) / 127.5  # Normalize the images to [-1, 1]

    # Batch and shuffle the data
    train_dataset = tf.data.Dataset.from_tensor_slices(train_images).shuffle(50000)
    return train_dataset


def get_validation_dataset(worker_rank: int):
    (_, _), (test_images, _) = tf.keras.datasets.mnist.load_data(path=f"mnist-{worker_rank}.npz")

    test_images = test_images.reshape(test_images.shape[0], 28, 28, 1).astype('float32')
    test_images = (test_images - 127.5) / 127.5  # Normalize the images to [-1, 1]

    # Batch and shuffle the data
    train_dataset = tf.data.Dataset.from_tensor_slices(test_images).shuffle(50000)
    return train_dataset
