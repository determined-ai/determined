"""
Perform inference on pretrained CIFAR10 from https://github.com/huyvnphan/PyTorch_CIFAR10
"""

import tempfile
from typing import Any, Dict, Sequence, Tuple, Union, cast

import ssl
import torch
import torchvision
from torch import nn
from torchvision import transforms
import torchvision.models as models
import urllib.request

from determined.pytorch import DataLoader, PyTorchTrial, PyTorchTrialContext

# Constants about the data set.
IMAGE_SIZE = 32
NUM_CHANNELS = 3
NUM_CLASSES = 10

TorchData = Union[Dict[str, torch.Tensor], Sequence[torch.Tensor], torch.Tensor]


ssl._create_default_https_context = ssl._create_unverified_context
response = urllib.request.urlopen('https://www.python.org')
print(response.read().decode('utf-8'))

def set_parameter_requires_grad(model, feature_extracting):
    if feature_extracting:
        for param in model.parameters():
            param.requires_grad = False

def initialize_resnet18(num_classes, feature_extract, use_pretrained=True):
    # Initialize these variables which will be set in this if statement. Each of these
    #   variables is model specific.
    model_ft = None
    input_size = 0
    model_ft = models.resnet18(pretrained=use_pretrained)
    set_parameter_requires_grad(model_ft, feature_extract)
    num_ftrs = model_ft.fc.in_features
    model_ft.fc = nn.Linear(num_ftrs, num_classes)
    input_size = 224
    return model_ft, input_size

def accuracy_rate(predictions: torch.Tensor, labels: torch.Tensor) -> float:
    """Return the accuracy rate based on dense predictions and sparse labels."""
    assert len(predictions) == len(labels), "Predictions and labels must have the same length."
    assert len(labels.shape) == 1, "Labels must be a column vector."

    return (  # type: ignore
        float((predictions.argmax(1) == labels.to(torch.long)).sum()) / predictions.shape[0]
    )

class CIFARTrial(PyTorchTrial):
    def __init__(self, context: PyTorchTrialContext) -> None:
        self.context = context
        self.download_directory = tempfile.mkdtemp()

	# TODO: Load your trained model. Below are example approaches.

	### Load a checkpoint from the Determined model registry 
	# model = Determined().get_model("mymodel")
	# ckpt_path = self.model.get_version().download()
	# ckpt = torch.load(os.path.join(ckpt_path, 'state_dict.pth'))
	# model.load_state_dict(ckpt['models_state_dict'][0])

	### Load a checkpoint from a previous experiment
	# from determined.experimental import Determined
	# checkpoint = Determined().get_experiment(id).top_checkpoint()
	# model = checkpoint.load()

	### Specify a UUID with `source_trial_id` in the experiment config

	### Load a model that was not trained by Determined
        model, input_size = initialize_resnet18(NUM_CLASSES, True, use_pretrained=True)
        self.model = self.context.wrap_model(model)

        # IGNORE: Dummy optimizer that needs to be specified but is unused

        # IGNORE: Dummy optimizer that needs to be specified but is unused
        self.optimizer = self.context.wrap_optimizer(torch.optim.RMSprop(
            self.model.parameters()))

    def train_batch(
        # IGNORE: No-op train_batch that does not train or generate metrics
        self, batch: TorchData, epoch_idx: int, batch_idx: int
    ) -> Dict[str, torch.Tensor]:
        return {}

    def evaluate_batch(self, batch: TorchData) -> Dict[str, Any]:
        """
        Calculate validation metrics for a batch and return them as a dictionary.
        This method is not necessary if the user defines evaluate_full_dataset().
        """

        # TODO: Perform your evaluation step
        batch = cast(Tuple[torch.Tensor, torch.Tensor], batch)
        data, labels = batch
        output = self.model(data)

        # TODO: Optionally log or save outputs to persistent store
        print(output)
        print(labels)
        '''
        with open("/path/to/output.txt", "w+") as f:
            f.write(output)
            f.write("\n")
        '''

        # TODO: Optionally log metrics to Determined
        accuracy = accuracy_rate(output, labels)
        return {"validation_accuracy": accuracy, "validation_error": 1.0 - accuracy}

    def build_training_data_loader(self) -> Any:
        # IGNORE: Dummy training data loader that must be specified but is unused
        transform = transforms.Compose(
            [transforms.ToTensor(), transforms.Normalize((0.5, 0.5, 0.5), (0.5, 0.5, 0.5))]
        )
        trainset = torchvision.datasets.CIFAR10(
            root=self.download_directory, train=True, download=True, transform=transform
        )
        return DataLoader(trainset, batch_size=self.context.get_per_slot_batch_size())

    def build_validation_data_loader(self) -> Any:
        # TODO: Add your evaluation dataset here
        transform = transforms.Compose(
            [transforms.ToTensor(), transforms.Normalize((0.4914, 0.4822, 0.4465), (0.2471, 0.2435, 0.2616))]
        )
        valset = torchvision.datasets.CIFAR10(
            root=self.download_directory, train=False, download=True, transform=transform
        )

        return DataLoader(valset, batch_size=self.context.get_per_slot_batch_size())
