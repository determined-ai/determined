from typing import Tuple, List, Any, Optional
from torch import optim
from torch.optim.optimizer import Optimizer
from pytorch_lightning.utilities.exceptions import MisconfigurationException
from pytorch_lightning.trainer.optimizers import _get_default_scheduler_config


def parse_config_optimizer_rv(optim_conf: Any) -> Tuple[List, List]:
    pass
    # if optim_conf is None:
    #     raise Exception('no optimizers configured')

    # optimizers, lr_schedulers, optimizer_frequencies = [], [], []
    # monitor = None

    # # single output, single optimizer
    # if isinstance(optim_conf, Optimizer):
    #     optimizers = [optim_conf]
    # # two lists, optimizer + lr schedulers
    # elif isinstance(optim_conf, (list, tuple)) and len(optim_conf) == 2 and isinstance(optim_conf[0], list):
    #     opt, sch = optim_conf
    #     optimizers = opt
    #     lr_schedulers = sch if isinstance(sch, list) else [sch]
    # # single dictionary
    # elif isinstance(optim_conf, dict):
    #     optimizers = [optim_conf["optimizer"]]
    #     monitor = optim_conf.get('monitor', None)
    #     lr_schedulers = [optim_conf["lr_scheduler"]] if "lr_scheduler" in optim_conf else []
    # # multiple dictionaries
    # elif isinstance(optim_conf, (list, tuple)) and all(isinstance(d, dict) for d in optim_conf):
    #     optimizers = [opt_dict["optimizer"] for opt_dict in optim_conf]
    #     lr_schedulers = [opt_dict["lr_scheduler"] for opt_dict in optim_conf if "lr_scheduler" in opt_dict]
    #     optimizer_frequencies = [
    #         opt_dict["frequency"] for opt_dict in optim_conf if opt_dict.get("frequency", None) is not None
    #     ]
    #     # assert that if frequencies are present, they are given for all optimizers
    #     if optimizer_frequencies and len(optimizer_frequencies) != len(optimizers):
    #         raise ValueError("A frequency must be given to each optimizer.")
    # # single list or tuple, multiple optimizer
    # elif isinstance(optim_conf, (list, tuple)):
    #     optimizers = list(optim_conf)
    # # unknown configuration
    # else:
    #     raise MisconfigurationException(
    #         'Unknown configuration for model optimizers.'
    #         ' Output from `model.configure_optimizers()` should either be:\n'
    #         ' * `torch.optim.Optimizer`\n'
    #         ' * [`torch.optim.Optimizer`]\n'
    #         ' * ([`torch.optim.Optimizer`], [`torch.optim.lr_scheduler`])\n'
    #         ' * {"optimizer": `torch.optim.Optimizer`, (optional) "lr_scheduler": `torch.optim.lr_scheduler`}\n'
    #         ' * A list of the previously described dict format, with an optional "frequency" key (int)'
    #     )

    # lr_schedulers = configure_schedulers(lr_schedulers, monitor=monitor)
    # _validate_scheduler_optimizer(optimizers, lr_schedulers)

    # return optimizers, lr_schedulers, optimizer_frequencies


