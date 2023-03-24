from transformers import BertConfig, BertForQuestionAnswering, BertTokenizer

MODEL_CLASSES = {
    "bert": (BertConfig, BertTokenizer, BertForQuestionAnswering),
}
