import model_hub.huggingface as hf


def test_get_label_list() -> None:
    labels = ["c", "d", "ab", "b", "d"]
    result = hf.get_label_list(labels)
    assert result == ["ab", "b", "c", "d"]
