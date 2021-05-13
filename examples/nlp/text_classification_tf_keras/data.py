import json
from pathlib import Path
from collections import defaultdict
import re
import string
import tensorflow as tf
from tensorflow.keras.layers.experimental.preprocessing import TextVectorization
import urllib.request
import gzip

# Ratio of testing data to training data
test_train_split = 0.75

# Source dataset gzip location. Can replace this url with any other k-cores.json dataset in
# its parent directory to train on other review sets
dataset_url = 'http://deepyeti.ucsd.edu/jianmo/amazon/categoryFilesSmall/Software_5.json.gz'

# Output file of extrated and downloaded dataset
raw_data_file = Path('reviews.json')

# Directory of processed sample data
dataset_path = Path('dataset')


# Downloads gzipped json file, parses output to training/testing dataset structure,
# and saves to local directories
def initialize_datasets():
    raw_json_samples = download_data(dataset_url, raw_data_file)
    dataset = extract_dataset(raw_json_samples)
    save_files(dataset, dataset_path)


# Download gzip file from url, write to output file, and return json content
def download_data(url, outfile):
    response_data = urllib.request.urlopen(url)
    with gzip.GzipFile(fileobj=response_data) as gzip_file:
        with open(outfile, 'wb') as json_file:
            file_content = gzip_file.readlines()
            json_file.writelines(file_content)
            return file_content


# Processes raw json sample objects into dictionary object organized by
# training/testing data and category labels
def extract_dataset(raw_samples):
    total_samples = len(raw_samples)
    train_test_index = int(total_samples * test_train_split)
    training_data = raw_samples[:train_test_index]
    testing_data = raw_samples[train_test_index:]
    return {
        'train': categorize_data(training_data),
        'test': categorize_data(testing_data)
    }


# Traverses dataset.json and saves samples in appropriate
# directory structure
def save_files(dataset, filepath):
    for path, data in dataset.items():
        if isinstance(data, dict):
            save_files(data, filepath / path)
        else:
            directory = filepath / path
            Path(directory).mkdir(parents=True, exist_ok=True)
            for idx, review in enumerate(data):
                filename = f'{str(idx)}.txt'
                save_path = directory / filename
                with open(save_path, 'w') as file:
                    file.write(review)


# Splits dataset into categories for classification
def categorize_data(dataset):
    parsed_dataset = [(parse_review(dataset_item)) for dataset_item in dataset]
    categorized_dataset = defaultdict(list)
    for label, review in parsed_dataset:
        categorized_dataset[label].append(review)
    return categorized_dataset


# Parses review json data into label, review text tuple
def parse_review(review_data):
    review_json = json.loads(review_data)
    return str(review_json['overall']), review_json.get('reviewText', '')


def load_training_data():
    training_ds = tf.keras.preprocessing.text_dataset_from_directory(
        dataset_path / 'train')
    return training_ds


def load_testing_data():
    testing_ds = tf.keras.preprocessing.text_dataset_from_directory(
        dataset_path / 'test')
    return testing_ds


# Cleans text input: removes tags, punctuation
def standardize_text(input_data):
    lowercase = tf.strings.lower(input_data)
    stripped_html = tf.strings.regex_replace(lowercase, '<br />', ' ')
    return tf.strings.regex_replace(stripped_html,
                                    '[%s]' % re.escape(string.punctuation),
                                    '')


# Create text tokenizer for mapping words to numerican inputs
def create_vectorization_layer(training_text=None):
    max_features = 10000
    sequence_length = 250

    vectorization_layer = TextVectorization(
        standardize=standardize_text,
        max_tokens=max_features,
        output_mode='int',
        output_sequence_length=sequence_length)

    if not training_text:
        training_data = load_training_data()
        training_text = training_data.map(lambda text, label: text)

    vectorization_layer.adapt(training_text)
    return vectorization_layer


if not raw_data_file.exists():
    initialize_datasets()


