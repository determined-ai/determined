import imgkit
import io
import numpy as np
from PIL import Image
from matplotlib.colors import LinearSegmentedColormap
import matplotlib.pyplot as plt

import torch
import torch.nn as nn
from torchvision.transforms import ToTensor

from captum.attr import IntegratedGradients
from captum.attr import TokenReferenceBase
from captum.attr import (
    visualization,
    configure_interpretable_embedding_layer,
    remove_interpretable_embedding_layer,
)

from utils import FullVQANet, visualize_text


def vqa_resnet_interpret(model, dataset, device, v, raw_q, raw_a, q=None, q_len=None):
    ig = IntegratedGradients(model)
    interpretable_embedding = configure_interpretable_embedding_layer(
        model, "text.embedding"
    )
    default_cmap = LinearSegmentedColormap.from_list(
        "custom blue", [(0, "#ffffff"), (0.25, "#252b36"), (1, "#000000")], N=256
    )
    token_reference = TokenReferenceBase(
        reference_token_idx=dataset.token_to_index["pad"]
    )
    torch.backends.cudnn.enabled = False
    v = v.unsqueeze(0)
    v.requires_grad_()
    if q is None:
        q, q_len = dataset._encode_question(raw_q)
        q = q.to(device)
        q_len = torch.tensor(q_len).to(device)

    q = q[0:q_len]
    q_input_embedding = interpretable_embedding.indices_to_embeddings(q).unsqueeze(0)

    # Making prediction. The output of prediction will be visualized later
    ans = model(v, q_input_embedding, q_len.unsqueeze(0))
    pred, answer_idx = nn.functional.softmax(ans, dim=1).cpu().max(dim=1)

    # generate reference for each sample
    q_reference_indices = token_reference.generate_reference(
        q_len.item(), device=device
    ).unsqueeze(0)
    q_reference = interpretable_embedding.indices_to_embeddings(q_reference_indices).to(
        device
    )
    attributions = ig.attribute(
        inputs=(v, q_input_embedding),
        baselines=(v * 0.0, q_reference),
        target=answer_idx,
        additional_forward_args=q_len.unsqueeze(0),
        n_steps=10,
    )
    # Visualize text attributions
    text_attributions_norm = attributions[1].sum(dim=2).squeeze(0).norm()
    vis_data_records = [
        visualization.VisualizationDataRecord(
            attributions[1].sum(dim=2).squeeze(0) / text_attributions_norm,
            pred[0].item(),
            dataset.answer_words[answer_idx],
            dataset.answer_words[answer_idx],
            raw_a,
            attributions[1].sum(),
            raw_q,
            0.0,
        )
    ]
    html = visualize_text(vis_data_records)
    img = imgkit.from_string(html.data, False, options={"xvfb": "", "quiet": ""})
    img = img[img.find(b"\xff") :]
    img_buf = io.BytesIO(img)
    text_viz = Image.open(img_buf)
    text_viz = ToTensor()(text_viz)

    # visualize image attributions
    original_im_mat = np.transpose(v[0].cpu().detach().numpy(), (1, 2, 0))
    attr = np.transpose(attributions[0].squeeze(0).cpu().detach().numpy(), (1, 2, 0))

    fig, _ = visualization.visualize_image_attr_multiple(
        attr,
        original_im_mat,
        ["original_image", "heat_map"],
        ["all", "absolute_value"],
        titles=["Original Image", "Attribution Magnitude"],
        cmap=default_cmap,
        show_colorbar=True,
    )
    buf = io.BytesIO()
    plt.savefig(buf, format="jpeg")
    buf.seek(0)
    image = Image.open(buf)
    image = ToTensor()(image)
    remove_interpretable_embedding_layer(model, interpretable_embedding)
    torch.backends.cudnn.enabled = True
    plt.close()
    return text_viz, image


if __name__ == "__main__":
    from data import get_dataset, get_transform

    model = FullVQANet(2048, 3000, 15193)
    model.to("cuda:0")
    dataset = get_dataset("determined-ai-coco-dataset", 448, 1, val=True)
    img = Image.open(open("elephant.jpg", "rb")).convert("RGB")
    transform = get_transform(448)
    img = transform(img)
    img = img.to("cuda:0")

    attr_image = vqa_resnet_interpret(
        model, dataset, "cuda:0", img, "what is on the picture".split(" "), "elephant"
    )
