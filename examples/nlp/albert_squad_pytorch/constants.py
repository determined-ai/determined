from transformers import (
    BertConfig,
    BertTokenizer,
    BertForQuestionAnswering,
    AlbertConfig,
    AlbertTokenizer,
    AlbertForQuestionAnswering,
)

MODEL_CLASSES = {
    "bert": (BertConfig, BertTokenizer, BertForQuestionAnswering),
    "albert": (
        AlbertConfig,
        AlbertTokenizer,
        AlbertForQuestionAnswering,
    ),
}
