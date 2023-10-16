import {
  ExclamationCircleOutlined,
  HolderOutlined,
  MinusCircleOutlined,
  ProjectOutlined,
} from '@ant-design/icons';
import React, { useMemo } from 'react';

import Tooltip from 'components/kit/Tooltip';

import ActiveIcon from './Icon/Active';
import css from './Icon/Icon.module.scss';
import QueuedIcon from './Icon/Queue';
import { SpinBowtie, SpinHalf, SpinShadow } from './Icon/Spin';
import AddIcon from './icons/add.svg';
import ArchiveIcon from './icons/archive.svg';
import ArrowDownIcon from './icons/arrow-down.svg';
import ArrowLeftIcon from './icons/arrow-left.svg';
import ArrowRightIcon from './icons/arrow-right.svg';
import ArrowUpIcon from './icons/arrow-up.svg';
import CancelledIcon from './icons/cancelled.svg';
import CheckmarkIcon from './icons/checkmark.svg';
import CheckpointIcon from './icons/checkpoint.svg';
import ClipboardIcon from './icons/clipboard.svg';
import CloseIcon from './icons/close.svg';
import CloudIcon from './icons/cloud.svg';
import ClusterIcon from './icons/cluster.svg';
import CollapseIcon from './icons/collapse.svg';
import ColumnsIcon from './icons/columns.svg';
import CommandIcon from './icons/command.svg';
import DaiLogoIcon from './icons/dai-logo.svg';
import DashboardIcon from './icons/dashboard.svg';
import DebugIcon from './icons/debug.svg';
import DocsIcon from './icons/docs.svg';
import DocumentIcon from './icons/document.svg';
import DownloadIcon from './icons/download.svg';
import ErrorIcon from './icons/error.svg';
import ExpandIcon from './icons/expand.svg';
import ExperimentIcon from './icons/experiment.svg';
import EyeCloseIcon from './icons/eye-close.svg';
import EyeOpenIcon from './icons/eye-open.svg';
import FilterIcon from './icons/filter.svg';
import ForkIcon from './icons/fork.svg';
import FourSquaresIcon from './icons/four-squares.svg';
import FullscreenIcon from './icons/fullscreen.svg';
import GridIcon from './icons/grid.svg';
import GroupIcon from './icons/group.svg';
import HeatIcon from './icons/heat.svg';
import HeatmapIcon from './icons/heatmap.svg';
import HomeIcon from './icons/home.svg';
import InfoIcon from './icons/info.svg';
import JupyterLabIcon from './icons/jupyter-lab.svg';
import LearningIcon from './icons/learning.svg';
import ListIcon from './icons/list.svg';
import LockIcon from './icons/lock.svg';
import LogsIcon from './icons/logs.svg';
import ModelIcon from './icons/model.svg';
import NotebookIcon from './icons/notebook.svg';
import OptionsIcon from './icons/options.svg';
import OverflowHorizontalIcon from './icons/overflow-horizontal.svg';
import OverflowVerticalIcon from './icons/overflow-vertical.svg';
import PanelOnIcon from './icons/panel-on.svg';
import PanelIcon from './icons/panel.svg';
import ParcoordsIcon from './icons/parcoords.svg';
import PauseIcon from './icons/pause.svg';
import PencilIcon from './icons/pencil.svg';
import PinIcon from './icons/pin.svg';
import PlayIcon from './icons/play.svg';
import PopoutIcon from './icons/popout.svg';
import PowerIcon from './icons/power.svg';
import QueueIcon from './icons/queue.svg';
import ResetIcon from './icons/reset.svg';
import RowExtraLargeIcon from './icons/row-extra-large.svg';
import RowLargeIcon from './icons/row-large.svg';
import RowMediumIcon from './icons/row-medium.svg';
import RowSmallIcon from './icons/row-small.svg';
import ScatterPlotIcon from './icons/scatter-plot.svg';
import ScrollIcon from './icons/scroll.svg';
import SearchIcon from './icons/search.svg';
import SearcherAdaptiveIcon from './icons/searcher-adaptive.svg';
import SearcherGridIcon from './icons/searcher-grid.svg';
import SearcherRandomIcon from './icons/searcher-random.svg';
import SettingsIcon from './icons/settings.svg';
import ShellIcon from './icons/shell.svg';
import SpinnerIcon from './icons/spinner.svg';
import StarIcon from './icons/star.svg';
import StopIcon from './icons/stop.svg';
import TasksIcon from './icons/tasks.svg';
import TensorBoardIcon from './icons/tensor-board.svg';
import TensorboardIcon from './icons/tensorboard.svg';
import UndoIcon from './icons/undo.svg';
import UserIcon from './icons/user.svg';
import WarningIcon from './icons/warning.svg';
import WorkspacesIcon from './icons/workspaces.svg';
import { XOR } from './internal/types';

export const IconSizeArray = [
  'tiny',
  'small',
  'medium',
  'large',
  'big',
  'great',
  'huge',
  'enormous',
  'giant',
  'jumbo',
  'mega',
] as const;

export type IconSize = (typeof IconSizeArray)[number];

export const svgIcons = [
  'columns',
  'filter',
  'four-squares',
  'options',
  'panel',
  'panel-on',
  'row-large',
  'row-medium',
  'row-small',
  'row-xl',
  'heatmap',
  'scroll',
  'home',
  'dai-logo',
  'arrow-left',
  'arrow-right',
  'add',
  'search',
  'arrow-down',
  'arrow-up',
  'cancelled',
  'group',
  'workspaces',
  'archive',
  'queue',
  'model',
  'fork',
  'pause',
  'play',
  'stop',
  'reset',
  'undo',
  'learning',
  'heat',
  'scatter-plot',
  'parcoords',
  'pencil',
  'settings',
  'docs',
  'power',
  'close',
  'dashboard',
  'checkmark',
  'cloud',
  'document',
  'logs',
  'tasks',
  'checkpoint',
  'download',
  'debug',
  'error',
  'warning',
  'info',
  'clipboard',
  'fullscreen',
  'eye-close',
  'eye-open',
  'user',
  'jupyter-lab',
  'lock',
  'popout',
  'spinner',
  'collapse',
  'expand',
  'tensorboard',
  'cluster',
  'command',
  'experiment',
  'grid',
  'list',
  'notebook',
  'overflow-horizontal',
  'overflow-vertical',
  'shell',
  'star',
  'tensor-board',
  'searcher-random',
  'searcher-grid',
  'searcher-adaptive',
  'critical',
  'trace',
  'webhooks',
  'external',
  'pin',
] as const;

const svgIconMap: Partial<Record<IconName, React.FC>> = {
  'add': AddIcon,
  'archive': ArchiveIcon,
  'arrow-down': ArrowDownIcon,
  'arrow-left': ArrowLeftIcon,
  'arrow-right': ArrowRightIcon,
  'arrow-up': ArrowUpIcon,
  'cancelled': CancelledIcon,
  'checkmark': CheckmarkIcon,
  'checkpoint': CheckpointIcon,
  'clipboard': ClipboardIcon,
  'close': CloseIcon,
  'cloud': CloudIcon,
  'cluster': ClusterIcon,
  'collapse': CollapseIcon,
  'columns': ColumnsIcon,
  'command': CommandIcon,
  'critical': ErrorIcon, // duplicate of error
  'dai-logo': DaiLogoIcon,
  'dashboard': DashboardIcon,
  'debug': DebugIcon,
  'docs': DocsIcon,
  'document': DocumentIcon,
  'download': DownloadIcon,
  'error': ErrorIcon,
  'expand': ExpandIcon,
  'experiment': ExperimentIcon,
  'external': GroupIcon, // duplicate of group
  'eye-close': EyeCloseIcon,
  'eye-open': EyeOpenIcon,
  'filter': FilterIcon,
  'fork': ForkIcon,
  'four-squares': FourSquaresIcon,
  'fullscreen': FullscreenIcon,
  'grid': GridIcon,
  'group': GroupIcon,
  'heat': HeatIcon,
  'heatmap': HeatmapIcon,
  'home': HomeIcon,
  'info': InfoIcon,
  'jupyter-lab': JupyterLabIcon,
  'learning': LearningIcon,
  'list': ListIcon,
  'lock': LockIcon,
  'logs': LogsIcon,
  'model': ModelIcon,
  'notebook': NotebookIcon,
  'options': OptionsIcon,
  'overflow-horizontal': OverflowHorizontalIcon,
  'overflow-vertical': OverflowVerticalIcon,
  'panel': PanelIcon,
  'panel-on': PanelOnIcon,
  'parcoords': ParcoordsIcon,
  'pause': PauseIcon,
  'pencil': PencilIcon,
  'pin': PinIcon,
  'play': PlayIcon,
  'popout': PopoutIcon,
  'power': PowerIcon,
  'queue': QueueIcon,
  'reset': ResetIcon,
  'row-large': RowLargeIcon,
  'row-medium': RowMediumIcon,
  'row-small': RowSmallIcon,
  'row-xl': RowExtraLargeIcon,
  'scatter-plot': ScatterPlotIcon,
  'scroll': ScrollIcon,
  'search': SearchIcon,
  'searcher-adaptive': SearcherAdaptiveIcon,
  'searcher-grid': SearcherGridIcon,
  'searcher-random': SearcherRandomIcon,
  'settings': SettingsIcon,
  'shell': ShellIcon,
  'spinner': SpinnerIcon,
  'star': StarIcon,
  'stop': StopIcon,
  'tasks': TasksIcon,
  'tensor-board': TensorBoardIcon,
  'tensorboard': TensorboardIcon,
  'trace': DaiLogoIcon, // duplicate of dai-logo
  'undo': UndoIcon,
  'user': UserIcon,
  'warning': WarningIcon,
  'webhooks': SearcherRandomIcon, // duplicate of searcher-random
  'workspaces': WorkspacesIcon,
} as const;

const antdIcons = ['exclamation-circle', 'holder', 'minus-circle', 'project'] as const;

const antdIconMap: Partial<Record<IconName, React.FC>> = {
  'exclamation-circle': ExclamationCircleOutlined,
  'holder': HolderOutlined,
  'minus-circle': MinusCircleOutlined,
  'project': ProjectOutlined,
};

const componentIcons = ['active', 'spin-bowtie', 'spin-half', 'spin-shadow', 'queued'] as const;

const componentIconMap: Partial<Record<IconName, React.FC>> = {
  'active': ActiveIcon,
  'queued': QueuedIcon,
  'spin-bowtie': SpinBowtie,
  'spin-half': SpinHalf,
  'spin-shadow': SpinShadow,
};

export const IconNameArray = [...svgIcons, ...antdIcons, ...componentIcons];

export type IconName = (typeof IconNameArray)[number];

type CommonProps = {
  color?: 'cancel' | 'error' | 'success';
  size?: IconSize;
  showTooltip?: boolean;
  name: IconName;
  backgroundColor?: React.CSSProperties['backgroundColor']; // currently only supported by Queued
  opacity?: React.CSSProperties['opacity']; // currently only supported by Queued
};
export type Props = CommonProps &
  XOR<
    {
      title: string;
    },
    {
      decorative: true;
    }
  >;
const Icon: React.FC<Props> = ({
  name,
  size = 'medium',
  color,
  decorative,
  title,
  showTooltip = false,
  backgroundColor,
  opacity,
}: Props) => {
  const classes = [css.base];

  const iconComponent = useMemo(() => {
    if (name === 'queued')
      return <QueuedIcon backgroundColor={backgroundColor} opacity={opacity} />;
    const MappedIcon = svgIconMap[name] ?? antdIconMap[name] ?? componentIconMap[name];
    return MappedIcon && <MappedIcon />;
  }, [backgroundColor, name, opacity]);

  if (size) classes.push(css[size]);
  if (color) classes.push(css[color]);

  const icon = (
    <span aria-label={title} className={classes.join(' ')}>
      {iconComponent}
    </span>
  );
  return showTooltip && !decorative ? <Tooltip content={title}>{icon}</Tooltip> : icon;
};

export default Icon;
