import React from 'react';

import Tooltip from 'components/kit/Tooltip';

import css from './Icon.module.scss';

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

export const IconNameArray = [
  'home',
  'dai-logo',
  'arrow-left',
  'arrow-right',
  'add-small',
  'close-small',
  'search',
  'arrow-down',
  'arrow-up',
  'cancelled',
  'group',
  'warning-large',
  'steering-wheel',
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
  'filter',
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
  'user-small',
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
] as const;

export type IconName = (typeof IconNameArray)[number];

export interface Props {
  color?: 'cancel' | 'error' | 'success';
  name: IconName;
  showTooltip?: boolean;
  size?: IconSize;
  title: string;
}

const Icon: React.FC<Props> = ({
  name,
  showTooltip = false,
  size = 'medium',
  title,
  color,
}: Props) => {
  const classes = [css.base];

  if (name) classes.push(`icon-${name}`);
  if (size) classes.push(css[size]);
  if (color) classes.push(css[color]);

  const icon = <span aria-label={title} className={classes.join(' ')} />;
  return showTooltip ? <Tooltip content={title}>{icon}</Tooltip> : icon;
};

export default Icon;
