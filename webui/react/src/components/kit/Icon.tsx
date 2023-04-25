import React from 'react';

import Tooltip from 'components/kit/Tooltip';

import css from './Icon.module.scss';

export type IconSize =
  | 'tiny'
  | 'small'
  | 'medium'
  | 'large'
  | 'big'
  | 'great'
  | 'huge'
  | 'enormous'
  | 'giant'
  | 'jumbo'
  | 'mega';

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
  name?: IconName;
  size?: IconSize;
  //style?: CSSProperties;
  title?: string;
}

const Icon: React.FC<Props> = ({
  name = 'star',
  size = 'medium',
  title,
  //style,
  ...rest
}: Props) => {
  const classes = [css.base];

  if (name) classes.push(`icon-${name}`);
  if (size) classes.push(css[size]);

  const icon = <span className={classes.join(' ')} {...rest} />;
  return title ? <Tooltip content={title}>{icon}</Tooltip> : icon;
};

export default Icon;
