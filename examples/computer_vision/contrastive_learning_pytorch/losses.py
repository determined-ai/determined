"""
Modified from https://github.com/HobbitLong/SupContrast to support distributed training. 

License for the SupContrast code repository is reproduced below.

=============================================================================
BSD 2-Clause License

Copyright (c) 2020, Yonglong Tian
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
   list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice,
   this list of conditions and the following disclaimer in the documentation
   and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
"""

import torch
import torch.nn as nn

from determined.horovod import hvd


class SupConLoss(nn.Module):
    """Supervised Contrastive Learning: https://arxiv.org/pdf/2004.11362.pdf.
    It also supports the unsupervised contrastive loss in SimCLR"""

    def __init__(
        self, temperature=0.07, base_temperature=0.07, rank=0, distributed=False
    ):
        super(SupConLoss, self).__init__()
        self.temperature = temperature
        self.base_temperature = base_temperature
        self.rank = rank
        self.distributed = distributed

    def forward(self, features, labels=None):
        """Compute loss for model. If `labels` is None,
        it degenerates to SimCLR unsupervised loss:
        https://arxiv.org/pdf/2002.05709.pdf
        Args:
            features: hidden vector of shape [bsz, n_views, ...].
            labels: ground truth of shape [bsz].
        Returns:
            A loss scalar.
        """
        device = torch.device("cuda") if features.is_cuda else torch.device("cpu")

        if len(features.shape) < 3:
            raise ValueError(
                "`features` needs to be [bsz, n_views, ...],"
                "at least 3 dimensions are required"
            )
        if len(features.shape) > 3:
            features = features.view(features.shape[0], features.shape[1], -1)

        if self.distributed:
            # Note: horovod's allgather function supports backproping through
            # a gathered tensor if gradients are required for the said tensor.
            all_features = hvd.allgather(features, "all_features")
        else:
            all_features = features

        contrast_count = all_features.shape[1]
        contrast_feature = torch.cat(torch.unbind(all_features, dim=1), dim=0)
        anchor_feature = torch.cat(torch.unbind(features, dim=1), dim=0)
        anchor_count = contrast_count

        # compute logits
        anchor_dot_contrast = torch.div(
            torch.matmul(anchor_feature, contrast_feature.T), self.temperature
        )
        # for numerical stability
        logits_max, _ = torch.max(anchor_dot_contrast, dim=1, keepdim=True)
        logits = anchor_dot_contrast - logits_max.detach()

        # get masks
        batch_size = features.shape[0]
        batch_size = torch.as_tensor([batch_size], device=device)
        total_batch_size = all_features.shape[0]
        if self.distributed:
            batch_sizes = hvd.allgather(batch_size, "batch_sizes")
            sum_batch_sizes = torch.cumsum(batch_sizes, dim=0)
            start_ind = 0 if self.rank == 0 else sum_batch_sizes[self.rank - 1]
            end_ind = sum_batch_sizes[self.rank]
            index_ids = torch.arange(start_ind, end_ind)
            all_ids = torch.arange(sum_batch_sizes[-1])
            if labels is not None:
                all_labels = hvd.allgather(labels, "all_labels")
            else:
                labels = index_ids
                all_labels = all_ids
        else:
            index_ids = all_ids = torch.arange(total_batch_size)
            if labels is not None:
                all_labels = labels
            else:
                labels = all_labels = all_ids
        labels = labels.contiguous().view(-1, 1)
        all_labels = all_labels.contiguous().view(-1, 1)
        mask = torch.eq(labels, all_labels.T).float().to(device)

        # mask-out self-contrast cases
        index_ids = index_ids.contiguous().view(-1, 1)
        all_ids = all_ids.contiguous().view(-1, 1)
        index_mask = torch.eq(index_ids, all_ids.T).float().to(device)
        logits_mask = 1 - torch.block_diag(index_mask, index_mask)

        # tile mask
        mask = mask.repeat(anchor_count, contrast_count)

        mask = mask * logits_mask

        # compute log_prob
        exp_logits = torch.exp(logits) * logits_mask
        log_prob = logits - torch.log(exp_logits.sum(1, keepdim=True))

        # compute mean of log-likelihood over positive
        mean_log_prob_pos = (mask * log_prob).sum(1) / mask.sum(1)

        # loss
        loss = -(self.temperature / self.base_temperature) * mean_log_prob_pos
        loss = loss.view(anchor_count, batch_size.item()).mean()

        return loss
