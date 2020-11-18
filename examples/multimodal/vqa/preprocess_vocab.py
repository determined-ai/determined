import json
from collections import Counter
import itertools

import data


def extract_vocab(iterable, top_k=None, start=0):
    """Turns an iterable of list of tokens into a vocabulary.
    These tokens could be single answers or word tokens in questions.
    """
    all_tokens = itertools.chain.from_iterable(iterable)
    counter = Counter(all_tokens)
    if top_k:
        most_common = counter.most_common(top_k)
        most_common = (t for t, c in most_common)
    else:
        most_common = counter.keys()
    # descending in count, then lexicographical order
    tokens = sorted(most_common, key=lambda x: (counter[x], x), reverse=True)
    vocab = {t: i for i, t in enumerate(tokens, start=start)}
    return vocab


def main():
    vocabulary_path = "vocab.json"
    max_answers = 3000
    questions = data.path_for(train=True, question=True)
    answers = data.path_for(train=True, answer=True)

    with open(questions, "r") as fd:
        questions = json.load(fd)
    with open(answers, "r") as fd:
        answers = json.load(fd)

    questions = data.prepare_questions(questions)
    answers = data.prepare_answers(answers)

    question_vocab = extract_vocab(questions, start=1)
    answer_vocab = extract_vocab(answers, top_k=max_answers)

    vocabs = {
        "question": question_vocab,
        "answer": answer_vocab,
    }
    print("num_tokens: {}".format(len(vocabs["question"])))
    with open(vocabulary_path, "w") as fd:
        json.dump(vocabs, fd)


if __name__ == "__main__":
    main()
