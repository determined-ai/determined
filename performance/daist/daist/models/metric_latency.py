from pathlib import Path
from typing import Dict, ItemsView, Iterable, List, NewType, Optional, Union

import matplotlib.pyplot as plt
import numpy as np
from matplotlib import ticker
from matplotlib.axes import Axes
from numpy import ndarray
from numpy.typing import ArrayLike

from ..framework.paths import PkgPath
from ..utils.misc import to_snake_case
from .base import BaseDict, Format
from .config import Determined
from .result import FileMeta
from .session import session

try:
    import tkinter
    plt.switch_backend('TkAgg')
except ImportError:
    plt.switch_backend('Agg')


HISTOGRAM_BINS = 51
METRIC_RESULTS_DIR = 'metrics'


class BaseDictWithNumpy(BaseDict):
    @staticmethod
    def _serialize_to_list(value: Iterable) -> list:
        if isinstance(value, ndarray):
            return value.tolist()
        else:
            return BaseDict._serialize_to_list(value)


class OpHist(BaseDictWithNumpy):
    class Key:
        type_ = NewType('OpHist.Key', str)
        READ: type_ = 'read'
        WRITE: type_ = 'write'

        all_ = (READ, WRITE)

    TestParams = dict()

    def __init__(self, dict_=None, /,
                 samples: Dict['OpHist.Key.type_',
                               Dict['ConcurrencyHist.Key.type_',
                                    Union[ArrayLike, List[float]]]] = None,
                 **kw):
        super().__init__(dict_, **kw)
        if samples is None:
            self._samples = dict()
            for key in self.Key.all_:
                self._samples[key] = dict()
        else:
            self._samples = samples

    @property
    def read(self) -> 'ConcurrencyHist':
        return self[self.Key.READ]

    @property
    def write(self) -> 'ConcurrencyHist':
        return self[self.Key.WRITE]

    def __getitem__(self, key: 'OpHist.Key.type_') -> 'ConcurrencyHist':
        if key in self.Key.all_:
            if key not in self:
                self[key] = ConcurrencyHist(samples=self._samples[key])
        return super().__getitem__(key)

    def _deserialize(self, key, value):
        if key in self.Key.all_:
            if key not in self:
                value = ConcurrencyHist(samples=self._samples[key])
                self[key] = value
            else:
                value = ConcurrencyHist(value, samples=self._samples[key])
        value._samples = self._samples.get(key)
        return super()._deserialize(key, value)

    def _serialize(self, key: 'OpHist.Key.type_', value: 'ConcurrencyHist'):
        for conc, hist in value.items():
            self._samples[key][conc] = hist.samples

        if key in self.Key.all_:
            value = self._serialize_to_dict(value)
        return super()._serialize(key, value)

    def make_plots(self, test_id: str, test_params: Optional[Dict] = None):
        if test_params is None:
            test_params = self.TestParams

        for op, conc_hist in self.items():
            conc_hist.save_plot(op, test_id, test_params=test_params)
            for conc, hist in conc_hist.items():
                if isinstance(hist, Hist):
                    hist.save_plot(op, conc, test_id, test_params=test_params)

    def save_to_results(self, test_id: str, test_params: Optional[Dict] = None):
        self.make_plots(test_id, test_params=test_params)
        session.result.add_obj(self,
                               METRIC_RESULTS_DIR / self.get_filename(),
                               meta=FileMeta({FileMeta.Key.TEST_ID: test_id}))

    @staticmethod
    def show():
        plt.show()

    def items(self) -> ItemsView['OpHist.Key.type_', 'ConcurrencyHist']:
        for op, conc_hist in super().items():
            yield op, conc_hist


class MetricLatencyOpHist(OpHist):
    class TestParamKey:
        METRIC_COUNT = 'metric_count'

    class TestParamVal:
        METRIC_COUNT = 256

    TestParams = {
        TestParamKey.METRIC_COUNT: TestParamVal.METRIC_COUNT
    }


class ConcurrencyHist(BaseDictWithNumpy):
    """
    Dict[int "concurrency", Hist]
    """
    HIST_TICK_COUNT = 15

    class Key:
        type_ = NewType('ConcurrencyHist.Key',  int)
        TOTAL: type_ = 0
        MAP: type_ = -1

        _str_map = {
            TOTAL: 'total',
            MAP: 'map'
        }

        @classmethod
        def get_str(cls, key: type_) -> str:
            return cls._str_map.get(key, str(key))

    def __init__(self, dict_=None, /,
                 samples: Dict['ConcurrencyHist.Key.type_',
                               Union[ArrayLike, List[float]]] = None,
                 **kw):
        super().__init__(dict_, **kw)
        if samples is None:
            raise ValueError('samples cannot be None')
        self._samples = samples

    @property
    def concurrencies(self) -> List[int]:
        return sorted([conc for conc in self.keys() if conc > self.Key.TOTAL])

    @property
    def total_hist(self) -> 'Hist':
        if self.Key.TOTAL not in self:
            self[self.Key.TOTAL] = self.get_total_hist()
        return self[self.Key.TOTAL]

    @property
    def total_map(self) -> List[ArrayLike]:
        if self.Key.MAP not in self:
            self[self.Key.MAP] = self.get_total_map()
        return self[self.Key.MAP]

    def get_total_hist(self) -> 'Hist':
        samples_list = [sample for samples in self._samples.values() for sample in samples]
        samples = np.array(samples_list)
        total_hist = Hist(samples=samples)
        self[self.Key.TOTAL] = total_hist
        return total_hist

    def get_total_map(self) -> List[ArrayLike]:
        hists_2d = list()
        concurrencies = [conc for conc in self.keys() if conc > self.Key.TOTAL]
        concurrencies.sort()
        for concurrency in concurrencies:
            hist = Hist(samples=self[concurrency].samples, bins=self.total_hist.bins)
            hists_2d.append(np.array(hist.counts)/sum(hist.counts))
        return hists_2d

    def make_plot(self, title: str, test_params: Optional[Dict] = None):
        fig, axs = plt.subplots(2, constrained_layout=True, figsize=(8, 5))
        axs: List[Axes]

        ax: Axes = axs[0]
        total_trials = sum(concurrency for concurrency in self.keys()
                           if concurrency > self.Key.TOTAL)
        ctx = \
            (f'{title.capitalize()} Metric Latencies by Concurrency\n\n'
             f'context:\n'
             f'  {Determined.Key.DET_MASTER}: {session.determined.host}\n'
             f'  samples (batches) / trial: {sum(self.total_hist.counts) // total_trials}\n'
             '{test_params}'
             f'  versions:\n'
             f'    determined\n'
             f'      client: {session.result.version.determined.client}\n'
             f'      server: {session.result.version.determined.server}\n'
             f'    {PkgPath.PATH.name}: {session.result.version.daist}\n')

        if not test_params:
            ctx = ctx.format(test_params='')
        else:
            str_list = ['  Test Parameters:']
            for key, val in test_params.items():
                str_list.append(f'    {key}: {val}')
            str_list.append('')
            ctx = ctx.format(test_params='\n'.join(str_list))

        ax.set_axis_off()
        ax.text(0, 0, ctx)

        ax = axs[1]

        # Calculate histogram bins
        bins = self.total_hist.bins
        heatmap = ax.imshow(np.array(self.total_map), aspect=len(bins) / len(self.concurrencies))

        pos = range(len(bins))
        labels = [f'{b:.2f}' for b in bins]
        ax.set_xticks(pos, labels)
        ax.xaxis.set_minor_locator(ticker.AutoMinorLocator(n=2))
        ax.xaxis.set_minor_locator(ticker.FixedLocator(ax.xaxis.get_ticklocs(minor=True)))
        ax.xaxis.set_minor_formatter(ticker.FixedFormatter(labels))
        ax.xaxis.set_major_formatter(ticker.NullFormatter())
        ax.tick_params(axis='x', which='both', rotation=90, labelsize=8)

        ax.set_yticks(np.arange(len(self.concurrencies)),
                      self.concurrencies, fontdict={'fontsize': 8})
        ax.set_ylabel('concurrency (trials)')
        ax.set_xlabel('time (seconds)')

        cbar = ax.figure.colorbar(heatmap, ax=ax, fraction=0.05, location='right')
        cbar.ax.set_ylabel('probability', rotation=-90, va="bottom")

        ax.axis('tight')

        return fig, axs

    def save_plot(self, title: str, test_id: str,
                  path: Union[str, Path, None] = None,
                  test_params: Optional[Dict] = None):
        meta = FileMeta({FileMeta.Key.TEST_ID: test_id})
        fig, _ = self.make_plot(title, test_params=test_params)
        if path is None:
            path = METRIC_RESULTS_DIR / self.get_filename(
                fmt=Format.PNG,
                tags=(f'{to_snake_case(title)}',))
        path_to_save_to = session.result.touch(path, meta=meta)
        fig.savefig(path_to_save_to)

    @staticmethod
    def show():
        plt.show()

    def items(self) -> ItemsView['ConcurrencyHist.Key.type_', 'Hist']:
        for conc, hist in super().items():
            yield conc, hist

    def _deserialize(self, key, value) -> Union['Hist', List[ArrayLike]]:
        if key >= self.Key.TOTAL:
            obj = super()._deserialize(key, Hist(value))
            obj._samples = self._samples.get(key)
        elif key == self.Key.TOTAL:
            obj = super()._deserialize(key, Hist(value))
        else:
            obj = super()._deserialize(key, value)
        return obj

    def _serialize(self, key, value: 'Hist'):
        if key >= self.Key.TOTAL:
            self._samples[key] = value.samples
            value = self._serialize_to_dict(value)
        elif key == self.Key.TOTAL:
            value = self._serialize_to_dict(value)
        elif key == self.Key.MAP:
            value = list([self._serialize_to_list(inner) for inner in value])
            value = self._serialize_to_list(value)
        return super()._serialize(key, value)

    def __getitem__(self, key: int) -> Union['Hist', List[ArrayLike]]:
        return super().__getitem__(key)


class Hist(BaseDictWithNumpy):
    class Key:
        COUNTS = 'counts'
        BINS = 'bins'

    def __init__(self, dict_=None, /,
                 bins: Union[ArrayLike, int] = HISTOGRAM_BINS,
                 samples: Union[ArrayLike, List[float]] = None,
                 **kw):
        if samples is not None:
            kw[self.Key.COUNTS], kw[self.Key.BINS] = np.histogram(samples, bins)
        self._samples = samples
        super().__init__(dict_, **kw)

    @property
    def counts(self) -> List[float]:
        return self[self.Key.COUNTS]

    @counts.setter
    def counts(self, value: List[float]):
        self[self.Key.COUNTS] = value

    @property
    def bins(self) -> List[float]:
        return self[self.Key.BINS]

    @bins.setter
    def bins(self, value: List[float]):
        self[self.Key.BINS] = value

    @property
    def samples(self) -> Optional[List[float]]:
        return self._samples

    def make_plot(self, title: str, concurrency: ConcurrencyHist.Key.type_,
                  test_params: Optional[Dict] = None):
        fig, axs = plt.subplots(2, constrained_layout=True)
        axs: List[Axes]

        ax: Axes = axs[0]
        ctx = \
            (f'{title.capitalize()} Metric Latencies\n\n'
             f'context:\n'
             f'  {Determined.Key.DET_MASTER}: {session.determined.host}\n'
             f'  concurrency (trials): {ConcurrencyHist.Key.get_str(concurrency)}\n'
             '{samples}\n'
             '{test_params}'
             f'  versions:\n'
             f'    determined\n'
             f'      client: {session.result.version.determined.client}\n'
             f'      server: {session.result.version.determined.server}\n'
             f'    {PkgPath.PATH.name}: {session.result.version.daist}\n')

        if concurrency > ConcurrencyHist.Key.TOTAL:
            samples_str = f'  samples (batches) / trial: {sum(self.counts) // concurrency}'
        elif concurrency == ConcurrencyHist.Key.TOTAL:
            samples_str = f'  samples: {sum(self.counts)}'
        else:
            samples_str = ''

        if test_params:
            str_list = ['  Test Parameters:']
            for key, val in test_params.items():
                str_list.append(f'    {key}: {val}')
            str_list.append('')
            test_params_str = '\n'.join(str_list)
        else:
            test_params_str = ''

        ctx = ctx.format(samples=samples_str, test_params=test_params_str)

        ax.set_axis_off()
        ax.text(0, 0, ctx)

        ax = axs[1]
        bins = self.bins
        counts = self.counts

        ax.bar(bins[:-1], counts, width=(bins[-1] - bins[-2]), align='edge')
        ax.set_xlabel('time (seconds)')
        ax.set_ylabel('frequency')

        return fig, axs

    def save_plot(self, title: str, concurrency: ConcurrencyHist.Key.type_, test_id: str,
                  path: Union[str, Path, None] = None,
                  test_params: Optional[Dict] = None):
        meta = FileMeta({FileMeta.Key.TEST_ID: test_id})
        fig, _ = self.make_plot(title, concurrency, test_params=test_params)
        if path is None:
            path = METRIC_RESULTS_DIR / self.get_filename(
                fmt=Format.PNG,
                tags=(f'{to_snake_case(title)}', str(concurrency)))
        path_to_save_to = session.result.touch(path, meta=meta)
        fig.savefig(path_to_save_to)

    @staticmethod
    def show():
        plt.show()

    @classmethod
    def _serialize(cls, key, value):
        return super()._serialize(key, cls._serialize_to_list(value))
