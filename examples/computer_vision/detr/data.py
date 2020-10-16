from detr.util.misc import nested_tensor_from_tensor_list


def unwrap_collate_fn(batch):
    batch = list(zip(*batch))
    batch[0] = nested_tensor_from_tensor_list(batch[0])
    batch[0] = {"tensors": batch[0].tensors, "mask": batch[0].mask}
    return tuple(batch)
