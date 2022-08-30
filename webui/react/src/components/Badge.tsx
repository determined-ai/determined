import { Tooltip } from 'antd';
import React, { CSSProperties, useMemo } from 'react';

import { stateToLabel } from 'constants/states';
import { useStore } from 'contexts/Store';
import { DarkLight, getCssVar } from 'shared/themes';
import { hsl2str, str2hsl } from 'shared/utils/color';
import { getStateColorCssVar, StateOfUnion } from 'themes';
import { ResourceState, RunState, SlotState } from 'types';

import css from './Badge.module.scss';

export enum BadgeType {
  Default = 'Default',
  Header = 'Header',
  Id = 'Id',
  State = 'State',
}

export interface BadgeProps {
  children?: React.ReactNode;
  state?: StateOfUnion;
  tooltip?: string;
  type?: BadgeType;
}

const Badge: React.FC<BadgeProps> = ({
  state = RunState.Active,
  tooltip,
  type = BadgeType.Default,
  ...props
}: BadgeProps) => {
  const { ui } = useStore();

  const { classes, style } = useMemo(() => {
    const isDark = ui.darkLight === DarkLight.Dark;
    const classes = [ css.base ];
    const style: CSSProperties = {};

    if (type === BadgeType.State) {
      const backgroundColor = str2hsl(getCssVar(getStateColorCssVar(state)));
      style.backgroundColor = hsl2str({
        ...backgroundColor,
        l: isDark ? 35 : 45,
        s: backgroundColor.s > 0 ? (isDark ? 70 : 50) : 0,
      });
      style.color = getStateColorCssVar(state, { isOn: true });
      classes.push(css.state);

      if (state === SlotState.Free
        || state === ResourceState.Warm
        || state === ResourceState.Potential) {
        classes.push(css.neutral);

        if (state === ResourceState.Potential) classes.push(css.dashed);
      }
      if (ui.darkLight === DarkLight.Dark) classes.push(css.dark);
    } else if (type === BadgeType.Id) {
      classes.push(css.id);
    } else if (type === BadgeType.Header) {
      classes.push(css.header);
    }

    return { classes, style };
  }, [ state, type, ui.darkLight ]);

  const badge = (
    <span className={classes.join(' ')} style={style}>
      {props.children ? props.children : type === BadgeType.State && state && stateToLabel(state)}
    </span>
  );

  return tooltip ? <Tooltip title={tooltip}>{badge}</Tooltip> : badge;
};

export default Badge;
