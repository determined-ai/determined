import React, { CSSProperties, useMemo } from 'react';

import useTheme from 'hooks/useTheme';
import { hex2hsl, hsl2str } from 'shared/utils/color';
import md5 from 'shared/utils/md5';
import { DarkLight } from 'themes';

import css from './WorkspaceIcon.module.scss';

interface Props {
  name?: string;
  size?: number;
  style?: CSSProperties;
}

const WorkspaceIcon: React.FC<Props> = ({ name, size = 70, style }: Props) => {

  const { mode } = useTheme();
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
    return hsl2str({ ...hslColor, l: mode === DarkLight.Light ? 90 : 10 });
  }, [ name, mode ]);

  const fontSize = useMemo(() => {
    if (size > 50) return 16;
    if (size > 25) return 12;
    return 10;
  }, [ size ]);

  return (
    <div
      className={css.base}
      style={{ backgroundColor: color, fontSize, height: size, width: size, ...style }}>
      <span>{nameAcronym}</span>
    </div>
  );
};

export default WorkspaceIcon;
