import os
from typing import Any, Dict, Tuple

import numpy as np
import tensorflow.keras as keras
from tensorflow.keras.preprocessing.image import ImageDataGenerator
from tensorflow.keras.utils import to_categorical
from tensorflow.python.keras.datasets.cifar import load_batch
from tensorflow.python.keras.utils.data_utils import Sequence

NUM_CLASSES = 10


def preprocess_data(data: np.ndarray) -> np.ndarray:
    return data.astype("float32") / 255


def preprocess_labels(labels: np.ndarray) -> np.ndarray:
    return to_categorical(labels, NUM_CLASSES)


def augment_data(
    data: np.ndarray,
    labels: np.ndarray,
    batch_size: int,
    data_augmentation: Dict[str, Any],
    shuffle: bool = False,
) -> Sequence:
    datagen = ImageDataGenerator(**data_augmentation)
    data = preprocess_data(data)
    labels = preprocess_labels(labels)
    return datagen.flow(data, labels, batch_size=batch_size, shuffle=shuffle)


def get_data(data_path: str) -> Tuple[Tuple[np.ndarray, np.ndarray], Tuple[np.ndarray, np.ndarray]]:
    num_train_samples = 50000

    train_data = np.empty((num_train_samples, 3, 32, 32), dtype="uint8")
    train_labels = np.empty((num_train_samples,), dtype="uint8")

    for i in range(1, 6):
        fpath = os.path.join(data_path, "data_batch_" + str(i))
        (
            train_data[(i - 1) * 10000 : i * 10000, :, :, :],
            train_labels[(i - 1) * 10000 : i * 10000],
        ) = load_batch(fpath)

    fpath = os.path.join(data_path, "test_batch")
    test_data, test_labels = load_batch(fpath)

    train_labels = np.reshape(train_labels, (len(train_labels), 1))
    test_labels = np.reshape(test_labels, (len(test_labels), 1))

    if keras.backend.image_data_format() == "channels_last":
        train_data = train_data.transpose(0, 2, 3, 1)
        test_data = test_data.transpose(0, 2, 3, 1)

    return (train_data, train_labels), (test_data, test_labels)
