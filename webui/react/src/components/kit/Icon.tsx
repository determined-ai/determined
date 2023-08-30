import React, { useMemo } from 'react';

import css from 'components/kit/Icon.module.scss';
import ColumnsIcon from 'components/kit/icons/ColumnsIcon.svg';
import FilterIcon from 'components/kit/icons/FilterIcon.svg';
import FourSquaresIcon from 'components/kit/icons/FourSquaresIcon.svg';
import HeatmapIcon from 'components/kit/icons/HeatmapIcon.svg';
import OptionsIcon from 'components/kit/icons/OptionsIcon.svg';
import PanelIcon from 'components/kit/icons/PanelIcon.svg';
import PanelOnIcon from 'components/kit/icons/PanelOnIcon.svg';
import RowIconExtraLarge from 'components/kit/icons/RowIconExtraLarge.svg';
import RowIconLarge from 'components/kit/icons/RowIconLarge.svg';
import RowIconMedium from 'components/kit/icons/RowIconMedium.svg';
import RowIconSmall from 'components/kit/icons/RowIconSmall.svg';
import ScrollIcon from 'components/kit/icons/ScrollIcon.svg';
import Tooltip from 'components/kit/Tooltip';

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

const fontIcons = [
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
  'external',
] as const;

type FontIconName = (typeof fontIcons)[number];

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
] as const;

type SvgIconName = (typeof svgIcons)[number];

export const IconNameArray = [...fontIcons, ...svgIcons];

export type IconName = (typeof IconNameArray)[number];

// intersection here is to ensure the index access in the component returns
// undefined | React.FC and not any
const svgIconMap: Record<SvgIconName, React.FC> & {
  [x in FontIconName]?: never;
} = {
  'columns': ColumnsIcon,
  'filter': FilterIcon,
  'four-squares': FourSquaresIcon,
  'heatmap': HeatmapIcon,
  'options': OptionsIcon,
  'panel': PanelIcon,
  'panel-on': PanelOnIcon,
  'row-large': RowIconLarge,
  'row-medium': RowIconMedium,
  'row-small': RowIconSmall,
  'row-xl': RowIconExtraLarge,
  'scroll': ScrollIcon,
};

type CommonProps = {
  color?: 'cancel' | 'error' | 'success';
  name: IconName;
  size?: IconSize;
  showTooltip?: boolean;
};
export type Props = CommonProps &
  (
    | {
        title: string;
        decorative?: never;
      }
    | {
        decorative: true;
      }
  );
const Icon: React.FC<Props> = (props: Props) => {
  const { name, size = 'medium', color } = props;
  const showTooltip = 'decorative' in props ? false : props.showTooltip ?? false;
  const title = 'decorative' in props ? undefined : props.title;
  const decorative = 'decorative' in props;
  const classes = [css.base];

  const svgIcon = useMemo(() => {
    const MappedIcon = svgIconMap[name];
    return MappedIcon && <MappedIcon />;
  }, [name]);

  if (name) classes.push(`icon-${name}`);
  if (size) classes.push(css[size]);
  if (color) classes.push(css[color]);

  const icon = (
    <span aria-label={decorative ? undefined : title} className={classes.join(' ')}>
      {svgIcon}
    </span>
  );
  return showTooltip ? <Tooltip content={title}>{icon}</Tooltip> : icon;
};

export default Icon;
