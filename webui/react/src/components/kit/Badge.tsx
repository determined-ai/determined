import React, { CSSProperties, useMemo } from 'react';

import useUI, { DarkLight, getCssVar } from 'components/kit/Theme';
import { hsl2str, HslColor, str2hsl } from 'utils/color';

import css from './Badge.module.scss';

export interface BadgeProps {
  text: string;
  backgroundColor?: HslColor;
  dashed?: boolean;
}

const fontColorLight = '#FFFFFF';
const fontColorDark = '#000810';

const Badge: React.FC<BadgeProps> = ({
  text,
  backgroundColor = str2hsl(getCssVar('var(--theme-surface)')),
  dashed = false,
  ...props
}: BadgeProps) => {
  const { ui } = useUI();

  const { classes, style } = useMemo(() => {
    const classes = [css.base];

    const style: CSSProperties = {
      backgroundColor: hsl2str(backgroundColor),
      border: backgroundColor.l < 15 ? '1px solid #646464' : '',
      color: backgroundColor.l > 70 ? fontColorDark : fontColorLight,
    };
    if (dashed) classes.push(css.dashed);
    const isDark = ui.darkLight === DarkLight.Dark;
    style.backgroundColor = hsl2str({
      ...backgroundColor,
      s: backgroundColor.s > 0 ? (isDark ? 70 : 50) : 0,
    });

    return { classes, style };
  }, [dashed, backgroundColor, ui.darkLight]);

  return (
    // Need this wrapper for tooltip to apply
    <span {...props}>
      <span className={classes.join(' ')} style={style}>
        {text}
      </span>
    </span>
  );
};

export default Badge;
