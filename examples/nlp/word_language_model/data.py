import os
from io import open
import torch
from pathlib import Path
import urllib.request
import zipfile
import torch

def load_dataset(path):
    data_dir = path / 'wikitext-2'
    if data_dir.exists():
        return
    url = 'https://s3.amazonaws.com/research.metamind.io/wikitext/wikitext-2-v1.zip'
    filehandle, _ = urllib.request.urlretrieve(url)
    zip_file_object = zipfile.ZipFile(filehandle, 'r')
    zip_file_object.extractall(path)
    extracted = data_dir.glob('*.token')
    for path in extracted:
        path.rename(data_dir / (path.name.split('.')[1] + '.txt'))

class Dictionary(object):
    def __init__(self):
        self.word2idx = {}
        self.idx2word = []

    def add_word(self, word):
        if word not in self.word2idx:
            self.idx2word.append(word)
            self.word2idx[word] = len(self.idx2word) - 1
        return self.word2idx[word]

    def __len__(self):
        return len(self.idx2word)


class Corpus(object):
    def __init__(self, path):
        self.dictionary = Dictionary()
        self.train = self.tokenize(path / 'train.txt')
        self.val = self.tokenize(path / 'valid.txt')
        self.test = self.tokenize(path / 'test.txt')

    def tokenize(self, path):
        """Tokenizes a text file."""
        assert path.exists()
        # Add words to the dictionary
        with open(path, 'r', encoding="utf8") as f:
            for line in f:
                words = line.split() + ['<eos>']
                for word in words:
                    self.dictionary.add_word(word)

        # Tokenize file content
        with open(path, 'r', encoding="utf8") as f:
            idss = []
            for line in f:
                words = line.split() + ['<eos>']
                ids = []
                for word in words:
                    ids.append(self.dictionary.word2idx[word])
                idss.append(torch.tensor(ids).type(torch.int64))
            ids = torch.cat(idss)

        return ids

class WikiTextDataset(torch.utils.data.IterableDataset):
    def __init__(self, path, mode, batch_size=20):
        assert mode in ['train', 'val'], "Data mode for WikiTextDataset must be one of ['train', 'val']"
        load_dataset(path)
        data_path = path / 'wikitext-2'
        self.corpus = Corpus(data_path)
        self.batch_size = batch_size
        self.ntokens = len(self.corpus.dictionary)
        self.process_data(mode)

    def process_data(self, mode):
        self.data = self.corpus.train if mode == 'train' else self.corpus.val
        nbatch = self.data.size(0) // self.batch_size
        # Trim off any extra elements that wouldn't cleanly fit (remainders).
        self.data = self.data.narrow(0, 0, nbatch * self.batch_size)
        # Evenly divide the data across the bsz batches.
        self.data = self.data.view(bsz, -1).t().contiguous()
    
    def __iter__(self):
        pass