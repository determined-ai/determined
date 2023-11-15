import { StateOfUnion, useTheme } from 'hew/Theme';
import Tooltip from 'hew/Tooltip';
import React, { CSSProperties, useMemo } from 'react';

import useUI from 'components/ThemeProvider';
import { stateToLabel } from 'constants/states';
import { ResourceState, RunState, SlotState, ValueOf } from 'types';
import { getStateColorThemeVar, hsl2str, str2hsl } from 'utils/color';

import css from './Badge.module.scss';

export const BadgeType = {
  Default: 'Default',
  Header: 'Header',
  Id: 'Id',
  State: 'State',
} as const;

export type BadgeType = ValueOf<typeof BadgeType>;

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
  const { isDarkMode } = useUI();
  const { getThemeVar } = useTheme();
  const { classes, style } = useMemo(() => {
    const classes = [css.base];
    const style: CSSProperties = {};
    if (type === BadgeType.State) {
      const backgroundColor = str2hsl(getThemeVar(getStateColorThemeVar(state)));
      style.backgroundColor = hsl2str({
        ...backgroundColor,
        l: isDarkMode ? 35 : 45,
        s: backgroundColor.s > 0 ? (isDarkMode ? 70 : 50) : 0,
      });
      style.color = getThemeVar(getStateColorThemeVar(state, { isOn: true }));
      classes.push(css.state);

      if (
        state === SlotState.Free ||
        state === ResourceState.Warm ||
        state === ResourceState.Potential
      ) {
        classes.push(css.neutral);

        if (state === ResourceState.Potential) classes.push(css.dashed);
      }
      if (isDarkMode) classes.push(css.dark);
    } else if (type === BadgeType.Id) {
      classes.push(css.id);
    } else if (type === BadgeType.Header) {
      classes.push(css.header);
    }

    return { classes, style };
  }, [getThemeVar, state, type, isDarkMode]);

  const badge = (
    <span className={classes.join(' ')} style={style}>
      {props.children ? props.children : type === BadgeType.State && state && stateToLabel(state)}
    </span>
  );

  return tooltip ? <Tooltip content={tooltip}>{badge}</Tooltip> : badge;
};

export default Badge;
