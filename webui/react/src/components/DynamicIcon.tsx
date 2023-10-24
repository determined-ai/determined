import useUI from 'determined-ui/Theme';
import React, { CSSProperties, useMemo } from 'react';

import { useTheme } from 'hooks/useTheme';
import { hex2hsl, hsl2str } from 'utils/color';
import md5 from 'utils/md5';

import css from './DynamicIcon.module.scss';

interface Props {
  name?: string;
  size?: number;
  style?: CSSProperties;
}

const DynamicIcon: React.FC<Props> = ({ name, size = 70, style }: Props) => {
  const { ui } = useUI();
  const { isDarkMode } = useTheme(ui.mode, ui.theme);

  const nameAcronym = useMemo(() => {
    if (!name) return '-';
    return name
      .split(/\s/)
      .reduce((response, word) => (response += word.slice(0, 1)), '')
      .slice(0, 2);
  }, [name]);

  const backgroundColor = useMemo(() => {
    const hslColor = name ? hex2hsl(md5(name).substring(0, 6)) : hex2hsl('#808080');
    return hsl2str({
      ...hslColor,
      l: isDarkMode ? 80 : 90,
      s: isDarkMode ? 40 : 77,
    });
  }, [name, isDarkMode]);

  const fontSize = useMemo(() => {
    if (size > 50) return 16;
    if (size > 25) return 12;
    return 10;
  }, [size]);

  return (
    <div
      className={css.base}
      style={{
        backgroundColor,
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
