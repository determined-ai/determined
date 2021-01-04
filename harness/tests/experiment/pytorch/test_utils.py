from determined.pytorch._pytorch_trial import iterate_batches


def test_iterate_batch_epoch():
    cases = [  # length, start, expected epoch index
        (30, 20, 0),
        (30, 30, 1),
        (30, 0, 0),
        (30, 31, 1),
        (30, 60, 2),
    ]

    for length, start, expected_idx in cases:
        for epoch_idx, _ in iterate_batches(dataset_len=length, start=start, end=start + 1):
            assert epoch_idx == expected_idx
            break


def test_iterate_batch_range():
    dataset_len = 30
    start = 5
    end = 110

    c = start
    for _, batch_range in iterate_batches(dataset_len, start, end):
        for batch_idx in batch_range:
            assert batch_idx == c
            c += 1
