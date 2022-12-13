from transformers import (
    AlbertConfig,
    AlbertForQuestionAnswering,
    AlbertTokenizer,
    BertConfig,
    BertForQuestionAnswering,
    BertTokenizer,
)

MODEL_CLASSES = {
    "bert": (BertConfig, BertTokenizer, BertForQuestionAnswering),
    "albert": (
        AlbertConfig,
        AlbertTokenizer,
        AlbertForQuestionAnswering,
    ),
}
