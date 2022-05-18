import React, { CSSProperties, useMemo } from 'react';

import { hex2hsl, hsl2str } from 'utils/color';
import md5 from 'utils/md5';

import css from './WorkspaceIcon.module.scss';

interface Props {
  name?: string;
  size?: number;
  style?: CSSProperties;
}

const WorkspaceIcon: React.FC<Props> = ({ name, size = 70, style }: Props) => {
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
    return hsl2str({ ...hslColor, l: 90 });
  }, [ name ]);

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
