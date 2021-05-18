import React from 'react';

import css from './InfoBox.module.scss';

export interface InfoRow {
  content?: React.ReactNode;
  label: string;
  onClick?: () => void,
}

export enum InfoboxStyle {
  Default,
  Boxed,
}

interface Props {
  rows: InfoRow[];
  style?: InfoboxStyle,
}

export const renderRow = ({ content, label, onClick }: InfoRow): React.ReactNode => {
  if (!content) return null;

  const classes = [ css.info ];
  if (onClick) classes.push(css.infoClickable);

  return (
    <div className={classes.join(' ')} key={label} onClick={onClick}>
      <div className={css.label}>{label}</div>
      <div className={css.content}>{content}</div>
    </div>
  );
};

const InfoBox: React.FC<Props> = ({ rows, style = InfoboxStyle.Default }: Props) => {
  const classes = [];
  if (style === InfoboxStyle.Boxed) classes.push(css.boxed);
  if (style === InfoboxStyle.Default) classes.push(css.default);

  return (
    <div className={classes.join(' ')}>
      {rows.map(renderRow)}
    </div>
  );
};

export default InfoBox;
