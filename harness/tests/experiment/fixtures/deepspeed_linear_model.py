# type: ignore
from typing import Any, Dict, Iterator, Optional, Tuple, Union

import attrdict
import deepspeed
import deepspeed.runtime.dataloader as ds_dataloader
import numpy as np
import torch

import determined.pytorch.deepspeed as det_ds
import tests.experiment.fixtures.pytorch_counter_callback as counter
from determined import pytorch


class MetricsCallbacks(pytorch.PyTorchCallback):
    def __init__(self, trial) -> None:
        self.trial = trial
        super().__init__()

    def on_validation_end(self, metrics: Dict) -> None:
        assert "loss" in metrics.keys()

    def on_checkpoint_upload_end(self, uuid: str) -> None:
        self.trial.checkpoint_uuid = uuid

    def on_checkpoint_load_start(self, checkpoint: Optional[Dict]):
        self.trial.checkpoint_found = checkpoint is not None


class ReproducibilityCallbacks(pytorch.PyTorchCallback):
    def __init__(self, trial) -> None:
        self.trial = trial
        super().__init__()

    def on_validation_end(self, metrics: Dict) -> None:
        self.trial.val_metrics.append(metrics)

    def on_training_workload_end(self, avg_metrics, batch_metrics):
        self.trial.avg_metrics.append(avg_metrics)
        self.trial.batch_metrics.append(batch_metrics)


class TwoEngineMetricsCallbacks(pytorch.PyTorchCallback):
    def __init__(self) -> None:
        super().__init__()

    def on_validation_end(self, metrics: Dict) -> None:
        assert "loss1" in metrics.keys()
        assert "loss2" in metrics.keys()


class LinearDataset(torch.utils.data.Dataset):
    def __init__(self, a: int, b: int, num_samples: int):
        self.a = a
        self.b = b
        self.num_samples = num_samples

    def __len__(self):
        return self.num_samples

    def __getitem__(self, idx) -> Tuple[torch.Tensor, torch.Tensor]:
        x = np.random.uniform() * 10
        noise = np.random.normal()
        val = self.a * x + self.b + noise
        return torch.tensor([x], dtype=torch.float32), torch.tensor([val], dtype=torch.float32)


class LinearDeepSpeedTrial(det_ds.DeepSpeedTrial):
    _searcher_metric = "loss"

    def __init__(self, context: det_ds.DeepSpeedTrialContext, hparams: Dict):
        self.context = context
        self.hparams = attrdict.AttrDict(hparams)
        self.checkpoint_uuid = None
        self.checkpoint_found = None
        if (
            self.hparams.test_manual_init_distributed
            or self.hparams.test_fail_manual_init_distributed
        ):
            assert (
                not torch.distributed.is_initialized()
            ), "distributed backend should not be initialized"
        if (
            self.hparams.test_manual_init_distributed
            and not self.hparams.test_fail_manual_init_distributed
        ):
            deepspeed.init_distributed(auto_mpi_discovery=False)
        if self.hparams.test_manual_grad_acc or self.hparams.test_fail_manual_grad_acc:
            self.context.disable_auto_grad_accumulation()
        if self.hparams.test_manual_dataloader:
            self.context.disable_dataset_reproducibility_checks()
        self.ds_config = attrdict.AttrDict(self.hparams.deepspeed_config)
        model = torch.nn.Linear(1, 1)
        self.model, optimizer, _, _ = deepspeed.initialize(
            model=model,
            config=self.ds_config,
            model_parameters=model.parameters(),
            dist_init_required=False,
        )
        self.model = self.context.wrap_model_engine(self.model)
        self.loss = torch.nn.MSELoss()
        self.reducer = None
        if self.hparams.test_custom_reducer:
            self.reducer = self.context.wrap_reducer(lambda x: np.mean(x) * 2, name="loss_2x")

    def build_callbacks(self) -> Dict[str, pytorch.PyTorchCallback]:
        return {"my_callbacks": MetricsCallbacks(trial=self)}

    def build_training_data_loader(self) -> Union[pytorch.DataLoader, torch.utils.data.DataLoader]:
        dataset = LinearDataset(1, 1, self.ds_config.train_batch_size * 2)
        dataloader = pytorch.DataLoader(
            dataset, batch_size=self.ds_config.train_micro_batch_size_per_gpu
        )
        if self.hparams.test_manual_dataloader or self.hparams.test_fail_dataset_repro_check:
            return ds_dataloader.RepeatingLoader(
                torch.utils.data.DataLoader(
                    dataset, batch_size=self.ds_config.train_micro_batch_size_per_gpu
                )
            )
        return dataloader

    def build_validation_data_loader(
        self,
    ) -> Union[pytorch.DataLoader, torch.utils.data.DataLoader]:
        dataset = LinearDataset(1, 1, self.ds_config.train_batch_size * 10)
        dataloader = pytorch.DataLoader(
            dataset, batch_size=self.ds_config.train_micro_batch_size_per_gpu
        )
        if self.hparams.test_manual_dataloader or self.hparams.test_fail_dataset_repro_check:
            return ds_dataloader.RepeatingLoader(
                torch.utils.data.DataLoader(
                    dataset, batch_size=self.ds_config.train_micro_batch_size_per_gpu
                )
            )
        return dataloader

    def train_batch(
        self,
        dataloader_iter: Optional[Iterator[pytorch.TorchData]],
        epoch_idx: int,
        batch_idx: int,
    ) -> Union[torch.Tensor, Dict[str, Any]]:
        losses = []
        num_batches = 1
        if self.hparams.test_manual_grad_acc:
            num_batches = self.model.gradient_accumulation_steps()
        if self.hparams.test_fail_manual_grad_acc:
            num_batches = self.model.gradient_accumulation_steps() - 1
        for _ in range(num_batches):
            x, y = self.context.to_device(next(dataloader_iter))
            preds = self.model(x)
            loss = self.loss(y, preds)
            self.model.backward(loss)
            self.model.step()
            losses.append(loss.cpu().detach().numpy())
        if self.reducer is not None:
            self.reducer.update(losses)

        if self.hparams.return_non_scalar_metrics:
            return {"loss": np.mean(losses), "losses": losses}
        return {"loss": np.mean(losses)}

    def evaluate_batch(
        self, dataloader_iter: Optional[Iterator[pytorch.TorchData]], batch_idx: int
    ) -> Dict[str, Any]:
        x, y = self.context.to_device(next(dataloader_iter))
        preds = self.model(x)
        loss = self.loss(y, preds)
        if self.reducer is not None:
            self.reducer.update(loss.detach().cpu().numpy())
        if self.hparams.return_non_scalar_metrics:
            return {"loss": loss, "preds": preds}
        return {"loss": loss}


class InvalidTrainMetricTrial(LinearDeepSpeedTrial):
    def train_batch(
        self,
        dataloader_iter: Optional[Iterator[pytorch.TorchData]],
        epoch_idx: int,
        batch_idx: int,
    ) -> Any:
        return (0, 0)


class InvalidValidMetricTrial(LinearDeepSpeedTrial):
    def evaluate_batch(
        self, dataloader_iter: Optional[Iterator[pytorch.TorchData]], batch_idx: int
    ) -> Any:
        return (0, 0)


class DifferingValidMetricKeyTrial(LinearDeepSpeedTrial):
    def evaluate_batch(
        self, dataloader_iter: Optional[Iterator[pytorch.TorchData]], batch_idx: int
    ) -> Dict[str, Any]:
        if batch_idx == 0:
            return {"loss1": 0}
        return {"loss": 0}


class LinearCallbackTrial(LinearDeepSpeedTrial):
    def __init__(self, context: det_ds.DeepSpeedTrialContext, hparams: Dict):
        super().__init__(context, hparams)
        self.counter = counter.Counter()

    def build_callbacks(self) -> Dict[str, pytorch.PyTorchCallback]:
        return {"counter": self.counter}


class LinearTwoEngineTrial(LinearDeepSpeedTrial):
    def __init__(self, context: det_ds.DeepSpeedTrialContext, hparams: Dict):
        self.context = context
        self.hparams = attrdict.AttrDict(hparams)
        self.ds_config = attrdict.AttrDict(self.hparams.deepspeed_config)
        model1 = torch.nn.Linear(1, 1)
        model2 = torch.nn.Linear(1, 1)
        self.loss = torch.nn.MSELoss()
        self.model1, _, _, _ = deepspeed.initialize(
            model=model1, config=self.ds_config, model_parameters=model1.parameters()
        )
        self.model2, _, _, _ = deepspeed.initialize(
            model=model2, config=self.ds_config, model_parameters=model2.parameters()
        )
        self.model1 = self.context.wrap_model_engine(self.model1)
        self.model2 = self.context.wrap_model_engine(self.model2)

    def build_callbacks(self) -> Dict[str, pytorch.PyTorchCallback]:
        return {"my_callbacks": TwoEngineMetricsCallbacks()}

    def train_batch(
        self,
        dataloader_iter: Optional[Iterator[pytorch.TorchData]],
        epoch_idx: int,
        batch_idx: int,
    ) -> Union[torch.Tensor, Dict[str, Any]]:
        x, y = self.context.to_device(next(dataloader_iter))

        def take_step(model):
            preds = model(x)
            loss = self.loss(y, preds)
            model.backward(loss)
            model.step()
            return loss

        return {"loss1": take_step(self.model1), "loss2": take_step(self.model2)}

    def evaluate_batch(
        self, dataloader_iter: Optional[Iterator[pytorch.TorchData]], batch_idx: int
    ) -> Dict[str, Any]:
        x, y = self.context.to_device(next(dataloader_iter))

        def take_step(model):
            preds = model(x)
            loss = self.loss(y, preds)
            return loss

        return {"loss1": take_step(self.model1), "loss2": take_step(self.model2)}


class LinearPipelineEngineTrial(LinearDeepSpeedTrial):
    def __init__(self, context: det_ds.DeepSpeedTrialContext, hparams: Dict):
        self.context = context
        self.hparams = attrdict.AttrDict(hparams)
        self.ds_config = attrdict.AttrDict(self.hparams.deepspeed_config)
        self.avg_metrics = []
        self.batch_metrics = []
        self.val_metrics = []
        model = torch.nn.Linear(1, 1)
        model = deepspeed.PipelineModule(
            layers=[model],
            loss_fn=torch.nn.MSELoss(),
            num_stages=1,
        )
        self.model, _, _, _ = deepspeed.initialize(
            model=model,
            config=self.ds_config,
            model_parameters=[p for p in model.parameters() if p.requires_grad],
        )
        self.model = self.context.wrap_model_engine(self.model)
        self.context.set_mpu(det_ds.make_deepspeed_mpu(self.model.mpu))

    def build_callbacks(self) -> Dict[str, pytorch.PyTorchCallback]:
        return {"my_callbacks": ReproducibilityCallbacks(trial=self)}

    def train_batch(
        self,
        dataloader_iter: Optional[Iterator[pytorch.TorchData]],
        epoch_idx: int,
        batch_idx: int,
    ) -> Union[torch.Tensor, Dict[str, Any]]:
        loss = self.model.train_batch(dataloader_iter)
        return {"loss": loss}

    def evaluate_batch(
        self, dataloader_iter: Optional[Iterator[pytorch.TorchData]], batch_idx: int
    ) -> Dict[str, Any]:
        loss = self.model.eval_batch(dataloader_iter)
        return {"loss": loss}


class InvalidValidDatasetTrial(LinearPipelineEngineTrial):
    def build_validation_data_loader(
        self,
    ) -> Union[pytorch.DataLoader, torch.utils.data.DataLoader]:
        dataset = LinearDataset(1, 1, self.ds_config.train_micro_batch_size_per_gpu)
        dataloader = pytorch.DataLoader(
            dataset, batch_size=self.ds_config.train_micro_batch_size_per_gpu
        )
        if self.hparams.test_manual_dataloader or self.hparams.test_fail_dataset_repro_check:
            return dataloader.get_data_loader(repeat=True)
        return dataloader
