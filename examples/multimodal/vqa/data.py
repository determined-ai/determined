"""
From: https://github.com/Cyanogenoid/pytorch-vqa/blob/master/data.py
"""
import json
import os
import os.path
import re
from io import BytesIO

from PIL import Image
import h5py
import torch
import torch.utils.data as data
import torchvision.transforms as transforms

from google.cloud import storage
from determined.util import download_gcs_blob_with_backoff


def path_for(
    task="OpenEnded",
    dataset="mscoco",
    qa_path="vqa",
    train=False,
    val=False,
    test=False,
    question=False,
    answer=False,
):
    assert train + val + test == 1
    assert question + answer == 1
    assert not (
        test and answer
    ), "loading answers from test split not supported"  # if you want to eval on test, you need to implement loading of a VQA Dataset without given answers yourself
    if train:
        split = "train2014"
    elif val:
        split = "val2014"
    else:
        split = "test2015"
    if question:
        fmt = "{0}_{1}_{2}_questions.json"
    else:
        fmt = "{1}_{2}_annotations.json"
    s = fmt.format(task, dataset, split)
    return os.path.join(qa_path, s)


def get_transform(target_size, central_fraction=1.0):
    return transforms.Compose(
        [
            transforms.Scale(int(target_size / central_fraction)),
            transforms.CenterCrop(target_size),
            transforms.ToTensor(),
            transforms.Normalize(mean=[0.485, 0.456, 0.406], std=[0.229, 0.224, 0.225]),
        ]
    )


def collate_fn(batch):
    # put question lengths in descending order so that we can use packed sequences later
    batch.sort(key=lambda x: x[-1], reverse=True)
    return data.dataloader.default_collate(batch)


class VQA(data.Dataset):
    """ VQA dataset, open-ended """

    def __init__(
        self,
        vocabulary_path,
        questions_path,
        answers_path,
        coco_dataset,
        answerable_only=False,
    ):
        super(VQA, self).__init__()
        with open(questions_path, "r") as fd:
            questions_json = json.load(fd)
        with open(answers_path, "r") as fd:
            answers_json = json.load(fd)
        with open(vocabulary_path, "r") as fd:
            vocab_json = json.load(fd)
        self._check_integrity(questions_json, answers_json)

        # vocab
        self.vocab = vocab_json
        self.token_to_index = self.vocab["question"]
        self.answer_to_index = self.vocab["answer"]
        # reading answer classes from the vocabulary
        self.answer_words = ["unk"] * len(self.answer_to_index)
        for w, idx in self.answer_to_index.items():
            self.answer_words[idx] = w

        # q and a
        self.raw_questions = list(prepare_questions(questions_json))
        self.raw_answers = list(prepare_answers(answers_json))
        self.questions = [self._encode_question(q) for q in self.raw_questions]
        self.answers = [self._encode_answers(a) for a in self.raw_answers]

        # v
        self.coco_ids = [q["image_id"] for q in questions_json["questions"]]
        self.coco_dataset = coco_dataset
        self.id_to_coco_ind = self.coco_dataset.id_to_ind

        # only use questions that have at least one answer?
        self.answerable_only = answerable_only
        if self.answerable_only:
            self.answerable = self._find_answerable()
        self.length = None

    @property
    def max_question_length(self):
        if not hasattr(self, "_max_length"):
            self._max_length = max(map(len, self.raw_questions))
        return self._max_length

    @property
    def num_tokens(self):
        return len(self.token_to_index) + 1  # add 1 for <unknown> token at index 0

    def _check_integrity(self, questions, answers):
        """ Verify that we are using the correct data """
        qa_pairs = list(zip(questions["questions"], answers["annotations"]))
        assert all(
            q["question_id"] == a["question_id"] for q, a in qa_pairs
        ), "Questions not aligned with answers"
        assert all(
            q["image_id"] == a["image_id"] for q, a in qa_pairs
        ), "Image id of question and answer don't match"
        assert questions["data_type"] == answers["data_type"], "Mismatched data types"
        assert (
            questions["data_subtype"] == answers["data_subtype"]
        ), "Mismatched data subtypes"

    def _find_answerable(self):
        """ Create a list of indices into questions that will have at least one answer that is in the vocab """
        answerable = []
        for i, answers in enumerate(self.answers):
            answer_has_index = len(answers.nonzero()) > 0
            # store the indices of anything that is answerable
            if answer_has_index:
                answerable.append(i)
        return answerable

    def _encode_question(self, question):
        """ Turn a question into a vector of indices and a question length """
        vec = torch.zeros(self.max_question_length).long()
        for i, token in enumerate(question):
            index = self.token_to_index.get(token, 0)
            vec[i] = index
        return vec, len(question)

    def _encode_answers(self, answers):
        """ Turn an answer into a vector """
        # answer vec will be a vector of answer counts to determine which answers will contribute to the loss.
        # this should be multiplied with 0.1 * negative log-likelihoods that a model produces and then summed up
        # to get the loss that is weighted by how many humans gave that answer
        answer_vec = torch.zeros(len(self.answer_to_index))
        for answer in answers:
            index = self.answer_to_index.get(answer)
            if index is not None:
                answer_vec[index] += 1
        return answer_vec

    def __getitem__(self, item):
        if self.answerable_only:
            # change of indices to only address answerable questions
            item = self.answerable[item]

        q, q_length = self.questions[item]
        a = self.answers[item]
        image_id = self.coco_ids[item]
        v = self.coco_dataset[self.id_to_coco_ind[image_id]]
        # since batches are re-ordered for PackedSequence's, the original question order is lost
        # we return `item` so that the order of (v, q, a) triples can be restored if desired
        # without shuffling in the dataloader, these will be in the order that they appear in the q and a json's.
        return v, q, a, item, q_length

    def __len__(self):
        if self.length is None:
            if self.answerable_only:
                return len(self.answerable)
            else:
                return len(self.questions)
        return self.length


# this is used for normalizing questions
_special_chars = re.compile("[^a-z0-9 ]*")

# these try to emulate the original normalization scheme for answers
_period_strip = re.compile(r"(?!<=\d)(\.)(?!\d)")
_comma_strip = re.compile(r"(\d)(,)(\d)")
_punctuation_chars = re.escape(r';/[]"{}()=+\_-><@`,?!')
_punctuation = re.compile(r"([{}])".format(re.escape(_punctuation_chars)))
_punctuation_with_a_space = re.compile(
    r"(?<= )([{0}])|([{0}])(?= )".format(_punctuation_chars)
)


def prepare_questions(questions_json):
    """ Tokenize and normalize questions from a given question json in the usual VQA format. """
    questions = [q["question"] for q in questions_json["questions"]]
    for question in questions:
        question = question.lower()[:-1]
        yield question.split(" ")


def prepare_answers(answers_json):
    """ Normalize answers from a given answer json in the usual VQA format. """
    answers = [
        [a["answer"] for a in ans_dict["answers"]]
        for ans_dict in answers_json["annotations"]
    ]
    # The only normalization that is applied to both machine generated answers as well as
    # ground truth answers is replacing most punctuation with space (see [0] and [1]).
    # Since potential machine generated answers are just taken from most common answers, applying the other
    # normalizations is not needed, assuming that the human answers are already normalized.
    # [0]: http://visualqa.org/evaluation.html
    # [1]: https://github.com/VT-vision-lab/VQA/blob/3849b1eae04a0ffd83f56ad6f70ebd0767e09e0f/PythonEvaluationTools/vqaEvaluation/vqaEval.py#L96

    def process_punctuation(s):
        # the original is somewhat broken, so things that look odd here might just be to mimic that behaviour
        # this version should be faster since we use re instead of repeated operations on str's
        if _punctuation.search(s) is None:
            return s
        s = _punctuation_with_a_space.sub("", s)
        if re.search(_comma_strip, s) is not None:
            s = s.replace(",", "")
        s = _punctuation.sub(" ", s)
        s = _period_strip.sub("", s)
        return s.strip()

    for answer_list in answers:
        yield list(map(process_punctuation, answer_list))


class CocoImages(data.Dataset):
    """ Dataset for MSCOCO images located in a folder on the filesystem """

    def __init__(self, path, transform=None):
        super(CocoImages, self).__init__()
        self.path = path
        self.id_to_filename = self._find_images()
        self.sorted_ids = sorted(
            self.id_to_filename.keys()
        )  # used for deterministic iteration order
        print("found {} images in {}".format(len(self), self.path))
        self.transform = transform

    def _find_images(self):
        id_to_filename = {}
        for filename in os.listdir(self.path):
            if not filename.endswith(".jpg"):
                continue
            id_and_extension = filename.split("_")[-1]
            id = int(id_and_extension.split(".")[0])
            id_to_filename[id] = filename
        return id_to_filename

    def __getitem__(self, item):
        id = self.sorted_ids[item]
        path = os.path.join(self.path, self.id_to_filename[id])
        img = Image.open(path).convert("RGB")

        if self.transform is not None:
            img = self.transform(img)
        return id, img

    def __len__(self):
        return len(self.sorted_ids)


def load_image(path):
    # Helper function from https://pytorch.org/docs/stable/_modules/torchvision/datasets/folder.html#ImageFolder
    with open(path, "rb") as f:
        img = Image.open(f)
        return img.convert("RGB")


def list_blobs(storage_client, bucket_name, prefix=None):
    # Helper functions for GCP from https://cloud.google.com/storage/docs/listing-objects#code-samples
    """Lists all the blobs in the bucket."""
    blobs = storage_client.list_blobs(bucket_name, prefix=prefix)
    return blobs


class COCO2014Dataset(data.Dataset):
    def __init__(self, bucket_path, bucket_name, transform=None):
        """
        Args:
            split: train or validation split to return the right dataset
            directory: root directory for imagenet where "train" and "validation" folders reside
        """
        # If bucket name is None, we will generate random data.
        self._bucket_name = bucket_name

        self._source_dir = bucket_path
        self._transform = transform

        self._storage_client = storage.Client()
        self._bucket = self._storage_client.bucket(bucket_name)

        self.id_to_filename = self._find_images()
        self.sorted_ids = sorted(
            self.id_to_filename.keys()
        )  # used for deterministic iteration order
        self.id_to_ind = {id: ind for ind, id in enumerate(self.sorted_ids)}
        print(
            "found {} images in {}".format(len(self.id_to_filename), self._source_dir)
        )
        self.transform = transform

    def _find_images(self):
        id_to_filename = {}

        # Get blobs from GCP
        blobs = list_blobs(
            self._storage_client, self._bucket_name, prefix=self._source_dir
        )

        for b in blobs:
            filename = b.name
            if not filename.endswith(".jpg"):
                continue
            id_and_extension = filename.split("_")[-1]
            id = int(id_and_extension.split(".")[0])
            id_to_filename[id] = filename
        return id_to_filename

    def __getitem__(self, item):
        id = self.sorted_ids[item]
        img_path = self.id_to_filename[id]
        blob = self._bucket.blob(img_path)
        img_str = download_gcs_blob_with_backoff(blob)
        img_bytes = BytesIO(img_str)
        img = Image.open(img_bytes)
        img = img.convert("RGB")

        if self.transform is not None:
            img = self.transform(img)
        return img

    def __len__(self):
        return len(self.sorted_ids)


class Composite(data.Dataset):
    """ Dataset that is a composite of several Dataset objects. Useful for combining splits of a dataset. """

    def __init__(self, *datasets):
        self.datasets = datasets
        self.id_to_ind = self.datasets[0].id_to_ind
        for d in self.datasets[1:]:
            start_ind = len(self.id_to_ind)
            for id, ind in d.id_to_ind.items():
                self.id_to_ind[id] = start_ind + ind

    def __getitem__(self, item):
        current = self.datasets[0]
        for d in self.datasets:
            if item < len(d):
                return d[item]
            item -= len(d)
        else:
            raise IndexError("Index too large for composite dataset")

    def __len__(self):
        return sum(map(len, self.datasets))


def get_dataset(bucket, image_size, central_fraction, train=False, val=False):
    """ Returns a data loader for the desired split """
    assert train + val == 1, "need to set exactly one of {train, val} to True"
    split = "train" if train else "val"
    transform = get_transform(image_size, central_fraction)
    paths = ["val2014"]
    datasets = [COCO2014Dataset(path, bucket, transform=transform) for path in paths]
    dataset = Composite(*datasets)
    split = VQA(
        "vocab.json",
        path_for(train=train, val=val, question=True),
        path_for(train=train, val=val, answer=True),
        dataset,
        answerable_only=train,
    )
    return split
