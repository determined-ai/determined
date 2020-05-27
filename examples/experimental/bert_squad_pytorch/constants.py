from transformers import (
    BertConfig,
    BertTokenizer,
    BertForQuestionAnswering,
)

MODEL_CLASSES = {
    "bert": (BertConfig, BertTokenizer, BertForQuestionAnswering),
}
