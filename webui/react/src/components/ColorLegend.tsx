import React, { useMemo } from 'react';

import { ColorScale } from 'components/kit/utils/color';

import css from './ColorLegend.module.scss';
import HumanReadableNumber from './HumanReadableNumber';

interface Props {
  colorScale: ColorScale[];
  title?: React.ReactNode;
}

const ColorLegend: React.FC<Props> = ({ colorScale, title }: Props) => {
  const gradientStyle = useMemo(() => {
    return { background: `linear-gradient(90deg, ${colorScale[0].color}, ${colorScale[1].color})` };
  }, [colorScale]);

  return (
    <div className={css.base}>
      {title && <div className={css.title}>{title}</div>}
      <div className={css.gradient} style={gradientStyle} />
      <div className={css.labels}>
        <HumanReadableNumber num={colorScale[0].scale} />
        <HumanReadableNumber num={colorScale[1].scale} />
      </div>
    </div>
  );
};

export default ColorLegend;
