import React, { useMemo } from 'react';

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
  'columns',
  'filter',
  'options',
  'panel',
  'row-small',
  'row-medium',
  'row-large',
  'row-xl',
] as const;

const ColumnsIcon: React.FC = () => (
  <svg fill="none" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
    <path
      clipRule="evenodd"
      d="M9 5H5V19H9V5ZM5 20H9H10H14H15H19C19.5523 20 20 19.5523 20 19V5C20 4.44772 19.5523 4 19 4H15H14H10H9H5C4.44772 4 4 4.44772 4 5V19C4 19.5523 4.44772 20 5 20ZM15 19H19V5H15V19ZM14 19V5H10V19H14Z"
      fill="currentcolor"
      fillRule="evenodd"
    />
  </svg>
);

const FilterIcon: React.FC = () => (
  <svg fill="none" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
    <path
      clipRule="evenodd"
      d="M5 7.5C4.72386 7.5 4.5 7.72386 4.5 8C4.5 8.27614 4.72386 8.5 5 8.5H19C19.2761 8.5 19.5 8.27614 19.5 8C19.5 7.72386 19.2761 7.5 19 7.5H5ZM7.5 12C7.5 11.7239 7.72386 11.5 8 11.5H16C16.2761 11.5 16.5 11.7239 16.5 12C16.5 12.2761 16.2761 12.5 16 12.5H8C7.72386 12.5 7.5 12.2761 7.5 12ZM10.5 16C10.5 15.7239 10.7239 15.5 11 15.5H13C13.2761 15.5 13.5 15.7239 13.5 16C13.5 16.2761 13.2761 16.5 13 16.5H11C10.7239 16.5 10.5 16.2761 10.5 16Z"
      fill="currentcolor"
      fillRule="evenodd"
    />
  </svg>
);

const OptionsIcon: React.FC = () => (
  <svg fill="none" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
    <path
      clipRule="evenodd"
      d="M13.5 8C13.5 7.17157 14.1716 6.5 15 6.5C15.8284 6.5 16.5 7.17157 16.5 8C16.5 8.82843 15.8284 9.5 15 9.5C14.1716 9.5 13.5 8.82843 13.5 8ZM12.55 7.5H4C3.72386 7.5 3.5 7.72386 3.5 8C3.5 8.27614 3.72386 8.5 4 8.5H12.55C12.7816 9.64112 13.7905 10.5 15 10.5C16.2095 10.5 17.2184 9.64112 17.45 8.5H20C20.2761 8.5 20.5 8.27614 20.5 8C20.5 7.72386 20.2761 7.5 20 7.5H17.45C17.2184 6.35888 16.2095 5.5 15 5.5C13.7905 5.5 12.7816 6.35888 12.55 7.5ZM3.5 16C3.5 15.7239 3.72386 15.5 4 15.5H6.55001C6.78164 14.3589 7.79052 13.5 9 13.5C10.2095 13.5 11.2184 14.3589 11.45 15.5H20C20.2761 15.5 20.5 15.7239 20.5 16C20.5 16.2761 20.2761 16.5 20 16.5H11.45C11.2184 17.6411 10.2095 18.5 9 18.5C7.79052 18.5 6.78164 17.6411 6.55001 16.5H4C3.72386 16.5 3.5 16.2761 3.5 16ZM9 14.5C9.82843 14.5 10.5 15.1716 10.5 16C10.5 16.8284 9.82843 17.5 9 17.5C8.17157 17.5 7.5 16.8284 7.5 16C7.5 15.1716 8.17157 14.5 9 14.5Z"
      fill="currentcolor"
      fillRule="evenodd"
    />
  </svg>
);

const PanelIcon: React.FC = () => (
  <svg fill="none" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
    <path
      clipRule="evenodd"
      d="M10.72 5H19V19H10.72V5ZM9.72 5H5V19H9.72V5ZM10.72 4H19C19.5523 4 20 4.44772 20 5V19C20 19.5523 19.5523 20 19 20H10.72H9.72H5C4.93096 20 4.86356 19.993 4.79847 19.9797C4.34278 19.8864 4 19.4832 4 19V5C4 4.44772 4.44772 4 5 4H9.72H10.72Z"
      fill="currentcolor"
      fillRule="evenodd"
    />
  </svg>
);

const RowIconLarge: React.FC = () => (
  <svg fill="none" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
    <path
      clipRule="evenodd"
      d="M5 6H19V14H5V6ZM4 6C4 5.44772 4.44772 5 5 5H19C19.5523 5 20 5.44772 20 6V14C20 14.5523 19.5523 15 19 15H5C4.44772 15 4 14.5523 4 14V6ZM4.5 18C4.22386 18 4 18.2239 4 18.5C4 18.7761 4.22386 19 4.5 19H19.5C19.7761 19 20 18.7761 20 18.5C20 18.2239 19.7761 18 19.5 18H4.5Z"
      fill="currentcolor"
      fillRule="evenodd"
    />
  </svg>
);

const RowIconMedium: React.FC = () => (
  <svg fill="none" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
    <path
      clipRule="evenodd"
      d="M19 6H5V10H19V6ZM5 5C4.44772 5 4 5.44772 4 6V10C4 10.5523 4.44772 11 5 11H19C19.5523 11 20 10.5523 20 10V6C20 5.44772 19.5523 5 19 5H5ZM4 14.5C4 14.2239 4.22386 14 4.5 14H19.5C19.7761 14 20 14.2239 20 14.5C20 14.7761 19.7761 15 19.5 15H4.5C4.22386 15 4 14.7761 4 14.5ZM4.5 18C4.22386 18 4 18.2239 4 18.5C4 18.7761 4.22386 19 4.5 19H19.5C19.7761 19 20 18.7761 20 18.5C20 18.2239 19.7761 18 19.5 18H4.5Z"
      fill="currentcolor"
      fillRule="evenodd"
    />
  </svg>
);

const RowIconExtraLarge: React.FC = () => (
  <svg fill="none" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
    <rect height="13" rx="0.5" stroke="currentcolor" width="15" x="4.5" y="5.5" />
  </svg>
);

const RowIconSmall: React.FC = () => (
  <svg fill="none" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
    <path
      clipRule="evenodd"
      d="M4.5 6C4.22386 6 4 6.22386 4 6.5C4 6.77614 4.22386 7 4.5 7H19.5C19.7761 7 20 6.77614 20 6.5C20 6.22386 19.7761 6 19.5 6H4.5ZM4.5 10C4.22386 10 4 10.2239 4 10.5C4 10.7761 4.22386 11 4.5 11H19.5C19.7761 11 20 10.7761 20 10.5C20 10.2239 19.7761 10 19.5 10H4.5ZM4 14.5C4 14.2239 4.22386 14 4.5 14H19.5C19.7761 14 20 14.2239 20 14.5C20 14.7761 19.7761 15 19.5 15H4.5C4.22386 15 4 14.7761 4 14.5ZM4.5 18C4.22386 18 4 18.2239 4 18.5C4 18.7761 4.22386 19 4.5 19H19.5C19.7761 19 20 18.7761 20 18.5C20 18.2239 19.7761 18 19.5 18H4.5Z"
      fill="currentcolor"
      fillRule="evenodd"
    />
  </svg>
);

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

  const svgIcon = useMemo(() => {
    if (name === 'columns') return <ColumnsIcon />;
    if (name === 'filter') return <FilterIcon />;
    if (name === 'options') return <OptionsIcon />;
    if (name === 'panel') return <PanelIcon />;
    if (name === 'row-small') return <RowIconSmall />;
    if (name === 'row-medium') return <RowIconMedium />;
    if (name === 'row-large') return <RowIconLarge />;
    if (name === 'row-xl') return <RowIconExtraLarge />;

    return null;
  }, [name]);

  if (name) classes.push(`icon-${name}`);
  if (size) classes.push(css[size]);
  if (color) classes.push(css[color]);

  const icon = (
    <span aria-label={title} className={classes.join(' ')}>
      {svgIcon}
    </span>
  );
  return showTooltip ? <Tooltip content={title}>{icon}</Tooltip> : icon;
};

export default Icon;
