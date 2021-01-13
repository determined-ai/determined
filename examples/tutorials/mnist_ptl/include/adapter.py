from typing import Callable, NewType, Any
import pytorch_lightning as pl


GH = NewType('GH', Callable[[str], Any])


class DETLightningModule(pl.LightningModule):
    def __init__(self, get_hparam: GH, *args, **kwargs):  # Py QUESTION should I add this is kwarg?
        super().__init__(*args, **kwargs)
        self.get_hparam = get_hparam
