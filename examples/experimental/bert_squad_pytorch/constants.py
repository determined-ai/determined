from transformers import (
    BertConfig,
    BertTokenizer,
    BertForQuestionAnswering,
    XLNetConfig,
    XLNetTokenizer,
    XLNetForQuestionAnswering,
)

MODEL_CLASSES = {
    "bert": (BertConfig, BertTokenizer, BertForQuestionAnswering),
    "xlnet": (XLNetConfig, XLNetTokenizer, XLNetForQuestionAnswering),
}
