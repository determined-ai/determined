import torch.utils
from torch import nn as nn

import data

import model
from IPython.core.display import HTML
from captum.attr import visualization as viz

import resnet as caffe_resnet


class FullVQANet(model.Net):
    def __init__(self, output_features, max_answers, embedding_tokens):
        super().__init__(output_features, max_answers, embedding_tokens)
        self.resnet_layer4 = Net()

    def forward(self, v, q, q_len):
        v = self.resnet_layer4(v)
        return super().forward(v, q, q_len)


def visualize_text(datarecords, legend=True):
    dom = ["<table width: 100%>"]
    rows = [
        "<tr><th>True Label</th>"
        "<th>Predicted Label</th>"
        "<th>Attribution Label</th>"
        "<th>Attribution Score</th>"
        "<th>Word Importance</th>"
    ]
    for datarecord in datarecords:
        rows.append(
            "".join(
                [
                    "<tr>",
                    viz.format_classname(datarecord.true_class),
                    viz.format_classname(
                        "{0} ({1:.2f})".format(
                            datarecord.pred_class, datarecord.pred_prob
                        )
                    ),
                    viz.format_classname(datarecord.attr_class),
                    viz.format_classname("{0:.2f}".format(datarecord.attr_score)),
                    viz.format_word_importances(
                        datarecord.raw_input, datarecord.word_attributions
                    ),
                    "<tr>",
                ]
            )
        )

    if legend:
        dom.append(
            '<div style="border-top: 1px solid; margin-top: 5px; \
            padding-top: 5px; display: inline-block">'
        )
        dom.append("<b>Legend: </b>")

        for value, label in zip([-1, 0, 1], ["Negative", "Neutral", "Positive"]):
            dom.append(
                '<span style="display: inline-block; width: 10px; height: 10px; \
                border: 1px solid; background-color: \
                {value}"></span> {label}  '.format(
                    value=viz._get_color(value), label=label
                )
            )
        dom.append("</div>")

    dom.append("".join(rows))
    dom.append("</table>")
    return HTML("".join(dom))


class Net(nn.Module):
    def __init__(self):
        super(Net, self).__init__()
        self.model = caffe_resnet.resnet50(pretrained=True)

        def save_output(module, input, output):
            self.buffer = output

        self.model.layer4.register_forward_hook(save_output)

    def forward(self, x):
        self.model(x)
        return self.buffer


def create_coco_loader(
    bucket, batch_size, data_workers, image_size, central_fraction, *paths
):
    transform = data.get_transform(image_size, central_fraction)
    datasets = [
        data.COCO2014Dataset(path, bucket, transform=transform) for path in paths
    ]
    dataset = data.Composite(*datasets)
    data_loader = torch.utils.data.DataLoader(
        dataset,
        batch_size=batch_size,
        num_workers=data_workers,
        shuffle=False,
        pin_memory=True,
    )
    return data_loader
