from typing import Optional
from io import open
import torch
from torch.utils.data import Dataset
from pathlib import Path
import urllib.request
import zipfile
import torch


def load_and_cache_dataset(path: Path, use_cached: Optional[bool] = True):
    data_dir = path / "wikitext-2"
    if not data_dir.exists():
        url = "https://s3.amazonaws.com/research.metamind.io/wikitext/wikitext-2-v1.zip"
        filehandle, _ = urllib.request.urlretrieve(url)
        zip_file_object = zipfile.ZipFile(filehandle, "r")
        zip_file_object.extractall(path)
        extracted = data_dir.iterdir()
        for p in extracted:
            p.rename(data_dir / (p.name.split(".")[1] + ".txt"))
    cache_dir = data_dir / "cache"
    if not (cache_dir.exists() and use_cached):
        cache_dir.mkdir(parents=True, exist_ok=True)
        corpus = Corpus(data_dir)
        torch.save(corpus.train, cache_dir / "train.pt")
        torch.save(corpus.val, cache_dir / "val.pt")
        torch.save(corpus.test, cache_dir / "test.pt")
        torch.save(torch.tensor(corpus.ntokens), cache_dir / "ntokens.pt")
    else:
        corpus = Corpus(
            data_dir,
            use_cache=True,
            train=torch.load(cache_dir / "train.pt"),
            val=torch.load(cache_dir / "val.pt"),
            test=torch.load(cache_dir / "test.pt"),
            ntokens=torch.load(cache_dir / "ntokens.pt").item(),
        )
    return corpus


class Dictionary(object):
    def __init__(self) -> None:
        self.word2idx = {}
        self.idx2word = []

    def add_word(self, word: str) -> int:
        if word not in self.word2idx:
            self.idx2word.append(word)
            self.word2idx[word] = len(self.idx2word) - 1
        return self.word2idx[word]

    def __len__(self) -> int:
        return len(self.idx2word)


class Corpus(object):
    def __init__(
        self,
        path: Path,
        use_cache: bool = False,
        train: Optional[torch.tensor] = None,
        val: Optional[torch.tensor] = None,
        test: Optional[torch.tensor] = None,
        ntokens: Optional[int] = None,
    ) -> None:
        self.dictionary = Dictionary()
        if not use_cache:
            self.train = self.tokenize(path / "train.txt")
            self.val = self.tokenize(path / "valid.txt")
            self.test = self.tokenize(path / "test.txt")
            self.ntokens = len(self.dictionary)
        else:
            assert train is not None, "Train must be specified if use_cache is True"
            assert val is not None, "Val must be specified if use_cache is True"
            assert test is not None, "Test must be specified if use_cache is True"
            assert ntokens is not None, "Ntokens must be specified if use_cache is True"
            self.train = train
            self.val = val
            self.test = test
            self.ntokens = ntokens

    def tokenize(self, path: Path) -> torch.Tensor:
        """Tokenizes a text file."""
        assert path.exists()
        # Add words to the dictionary
        with open(path, "r", encoding="utf8") as f:
            for line in f:
                words = line.split() + ["<eos>"]
                for word in words:
                    self.dictionary.add_word(word)

        # Tokenize file content
        with open(path, "r", encoding="utf8") as f:
            idss = []
            for line in f:
                words = line.split() + ["<eos>"]
                ids = []
                for word in words:
                    ids.append(self.dictionary.word2idx[word])
                idss.append(torch.tensor(ids).type(torch.int64))
            ids = torch.cat(idss)

        return ids


class WikiTextDataset(Dataset):
    def __init__(
        self,
        path: Path,
        batch_size: int = 20,
        valid: bool = False,
        use_cache: bool = True,
    ):
        self.batch_size = batch_size
        self.valid = valid
        self.corpus = load_and_cache_dataset(path, use_cache)
        self.data = self.batchify()

    def batchify(self):
        data = self.corpus.val if self.valid else self.corpus.train
        # Work out how cleanly we can divide the dataset into bsz parts.
        nbatch = data.size(0) // self.batch_size
        # Trim off any extra elements that wouldn't cleanly fit (remainders).
        data = data.narrow(0, 0, nbatch * self.batch_size)
        # Evenly divide the data across the bsz batches.
        data = data.view(self.batch_size, -1).t().contiguous()
        return data

    def __len__(self):
        return len(self.data)

    def __getitem__(self, i):
        return self.data[i]


class BatchSamp:
    def __init__(self, dataset, bptt):
        self.data = dataset
        self.data_length = len(dataset) - 1
        self.bptt = bptt

    def __len__(self):
        return self.data_length // self.bptt

    def __iter__(self):
        for batch in range(0, self.data_length, self.bptt):
            seq_len = min(self.bptt, self.data_length - batch)
            yield self.data[batch : batch + seq_len + 1]
