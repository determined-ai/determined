import React, { CSSProperties, useMemo } from 'react';

import useUI, { DarkLight, getCssVar } from 'components/kit/Theme';
import { hsl2str, HslColor, str2hsl } from 'utils/color';

import css from './Badge.module.scss';

export interface BadgeColor {
  backgroundColor: HslColor;
  color: HslColor;
}

export interface BadgeProps {
  text: string;
  badgeColor?: BadgeColor;
  dashed?: boolean;
}

const Badge: React.FC<BadgeProps> = ({
  text,
  badgeColor = {
    backgroundColor: str2hsl(getCssVar('var(--theme-surface)')),
    color: str2hsl(getCssVar('var(--theme-surface-on)')),
  },
  dashed = false,
}: BadgeProps) => {
  const { ui } = useUI();

  const { classes, style } = useMemo(() => {
    const classes = [css.base];

    const { backgroundColor, color } = badgeColor;
    const style: CSSProperties = {
      backgroundColor: hsl2str(backgroundColor),
      color: hsl2str(color),
    };
    if (dashed) classes.push(css.dashed);
    const isDark = ui.darkLight === DarkLight.Dark;
    style.backgroundColor = hsl2str({
      ...backgroundColor,
      s: backgroundColor.s > 0 ? (isDark ? 70 : 50) : 0,
    });

    return { classes, style };
  }, [dashed, badgeColor, ui.darkLight]);

  return (
    <span className={classes.join(' ')} style={style}>
      {text}
    </span>
  );
};

export default Badge;
