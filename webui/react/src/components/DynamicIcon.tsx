import React, { CSSProperties, useMemo } from 'react';

import { useStore } from 'contexts/Store';
import { hex2hsl, hsl2str } from 'shared/utils/color';
import md5 from 'shared/utils/md5';
import { DarkLight } from 'types';

import css from './DynamicIcon.module.scss';

interface Props {
  name?: string;
  size?: number;
  style?: CSSProperties;
}

const DynamicIcon: React.FC<Props> = ({ name, size = 70, style }: Props) => {
  const { ui } = useStore();

  const nameAcronym = useMemo(() => {
    if (!name) return '-';
    return name
      .split(/\s/).reduce((response, word) => response += word.slice(0, 1), '')
      .slice(0, 2);
  }, [ name ]);

  const color = useMemo(() => {
    if (!name) {
      return hsl2str({ ...hex2hsl('#808080'), l: 90 });
    }
    const hexColor = md5(name).substring(0, 6);
    const hslColor = hex2hsl(hexColor);
    return hsl2str({
      ...hslColor,
      l: ui.darkLight === DarkLight.Dark ? 80 : 90,
      s: ui.darkLight === DarkLight.Dark ? 40 : 77,
    });
  }, [ name, ui.darkLight ]);

  const fontSize = useMemo(() => {
    if (size > 50) return 16;
    if (size > 25) return 12;
    return 10;
  }, [ size ]);

  const borderRadius = useMemo(() => {
    if (size > 50) return 'var(--theme-border-radius-strong)';
    return 'var(--theme-border-radius)';
  }, [ size ]);

  return (
    <div
      className={css.base}
      style={{
        backgroundColor: color,
        borderRadius,
        color: 'black',
        fontSize,
        height: size,
        width: size,
        ...style,
      }}>
      <span>{nameAcronym}</span>
    </div>
  );
};

export default DynamicIcon;
