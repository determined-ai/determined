import React from 'react';

import css from './InfoBox.module.scss';

export interface InfoRow {
  content?: React.ReactNode;
  label: string;
}

interface Props {
  rows: InfoRow[];
}

export const renderRow = ({ label, content }: InfoRow): React.ReactNode => {
  if (!content) return null;
  return (
    <div className={css.info} key={label}>
      <div className={css.label}>{label}</div>
      <div className={css.content}>{content}</div>
    </div>
  );
};

const InfoBox: React.FC<Props> = ({ rows }: Props) => {
  return (
    <div className={css.base}>
      {rows.map(renderRow)}
    </div>
  );
};

export default InfoBox;
